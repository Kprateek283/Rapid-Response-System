package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/google-hackathon/rapid-response/internal/api/handlers/auth"
	"github.com/google-hackathon/rapid-response/internal/api/handlers/guest"
	"github.com/google-hackathon/rapid-response/internal/api/handlers/iam"
	incidentHandlers "github.com/google-hackathon/rapid-response/internal/api/handlers/incident"
	"github.com/google-hackathon/rapid-response/internal/api/handlers/onboard"
	"github.com/google-hackathon/rapid-response/internal/api/router"
	"github.com/google-hackathon/rapid-response/internal/dispatch"
	"github.com/google-hackathon/rapid-response/internal/mq"
	"github.com/google-hackathon/rapid-response/internal/notification"
	authRepo "github.com/google-hackathon/rapid-response/internal/repository/auth"
	guestRepo "github.com/google-hackathon/rapid-response/internal/repository/guest"
	iamRepo "github.com/google-hackathon/rapid-response/internal/repository/iam"
	incidentRepo "github.com/google-hackathon/rapid-response/internal/repository/incident"
	authService "github.com/google-hackathon/rapid-response/internal/service/auth"
	guestService "github.com/google-hackathon/rapid-response/internal/service/guest"
	iamService "github.com/google-hackathon/rapid-response/internal/service/iam"
	incidentService "github.com/google-hackathon/rapid-response/internal/service/incident"
	onboardService "github.com/google-hackathon/rapid-response/internal/service/onboard"
	"github.com/google-hackathon/rapid-response/internal/ws"
	"github.com/google-hackathon/rapid-response/pkg/config"
	"github.com/google-hackathon/rapid-response/pkg/infra"
)

func main() {
	ctx := context.Background()
	log.Println("Initializing Rapid Crisis Response Backend...")

	// ---------------------------------------------------------
	// 0. LOAD CONFIGURATION
	// ---------------------------------------------------------
	cfg := config.Load()

	// ---------------------------------------------------------
	// 1. INFRASTRUCTURE INITIALIZATION
	// ---------------------------------------------------------
	pgPool, err := infra.NewPostgresPool(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("❌ Postgres init failed: %v", err)
	}
	defer pgPool.Close()

	redisClient, err := infra.NewRedisClient(ctx, cfg.RedisURL)
	if err != nil {
		log.Fatalf("❌ Redis init failed: %v", err)
	}
	defer redisClient.Close()

	rabbitConn, err := infra.NewRabbitMQConnection(cfg.RabbitMQURL)
	if err != nil {
		log.Fatalf("❌ RabbitMQ init failed: %v", err)
	}
	defer rabbitConn.Close()

	minioClient, err := infra.NewMinioClient(cfg.StorageEndpoint, cfg.StorageAccessKey, cfg.StorageSecretKey, cfg.StorageUseSSL)
	if err != nil {
		log.Fatalf("❌ MinIO init failed: %v", err)
	}

	log.Println("✅ All infrastructure connected successfully.")

	// ---------------------------------------------------------
	// 2. RabbitMQ Queue Setup
	// ---------------------------------------------------------
	if err := mq.SetupQueues(rabbitConn); err != nil {
		log.Printf("⚠️ RabbitMQ queue setup warning (non-fatal): %v", err)
	}

	// MQ Publisher
	mqPublisher, err := mq.NewPublisher(rabbitConn)
	if err != nil {
		log.Fatalf("❌ MQ Publisher init failed: %v", err)
	}
	defer mqPublisher.Close()

	// ---------------------------------------------------------
	// 3. DATABASE SEEDING
	// ---------------------------------------------------------
	SeedSuperAdmin(pgPool)

	// ---------------------------------------------------------
	// 4. CORE SERVICES INITIALIZATION
	// ---------------------------------------------------------

	// WebSocket Hub (shared across all incident rooms)
	wsHub := ws.NewHub()

	// FCM Notifier (mocked)
	notifier := notification.NewMockNotifier(pgPool)

	// ---------------------------------------------------------
	// 5. DEPENDENCY INJECTION (DI)
	// ---------------------------------------------------------

	// Auth Domain
	authRepository := authRepo.NewAuthRepository(pgPool, redisClient)
	authSvc := authService.NewAuthService(authRepository, cfg.JWTSecret)
	authHandler := auth.NewAuthHandler(authSvc)

	// IAM Domain — Groups
	groupRepository := iamRepo.NewGroupRepository(pgPool)
	groupSvc := iamService.NewGroupService(groupRepository)
	groupHandler := iam.NewGroupHandler(groupSvc)

	// IAM Domain — Hotels
	hotelRepository := iamRepo.NewHotelRepository(pgPool)
	hotelSvc := iamService.NewHotelService(hotelRepository)
	hotelHandler := iam.NewHotelHandler(hotelSvc)

	// IAM Domain — Rooms
	roomRepository := iamRepo.NewRoomRepository(pgPool)
	roomSvc := iamService.NewRoomService(roomRepository)
	roomHandler := iam.NewRoomHandler(roomSvc)

	// IAM Domain — Staff
	staffRepository := iamRepo.NewStaffRepository(pgPool, redisClient)
	staffSvc := iamService.NewStaffService(staffRepository)
	staffHandler := iam.NewStaffHandler(staffSvc)

	// Guest Domain
	guestRepository := guestRepo.NewGuestRepository(pgPool)
	guestSvc := guestService.NewGuestService(guestRepository, roomRepository, authSvc, redisClient)
	guestHandler := guest.NewGuestHandler(guestSvc)

	// Onboard Domain
	onboardSvc := onboardService.NewOnboardService(roomRepository, guestRepository, authSvc)
	onboardHandler := onboard.NewOnboardHandler(onboardSvc)

	// Incident Domain
	incRepo := incidentRepo.NewIncidentRepository(pgPool)
	dispatchEngine := dispatch.NewEngine(pgPool, redisClient, wsHub, notifier, incRepo)
	incSvc := incidentService.NewIncidentService(incRepo, redisClient, wsHub, mqPublisher, dispatchEngine, minioClient, cfg.StorageBucket)
	incHandler := incidentHandlers.NewIncidentHandler(incSvc)
	smsHandler := incidentHandlers.NewSMSHandler(incRepo, wsHub)
	wsHandlerInc := incidentHandlers.NewWSHandler(wsHub)

	// ---------------------------------------------------------
	// 6. MQ CONSUMERS (background goroutines)
	// ---------------------------------------------------------
	mqConsumer := mq.NewConsumer(rabbitConn, incRepo, dispatchEngine, wsHub)
	if err := mqConsumer.StartConsumers(); err != nil {
		log.Printf("⚠️ MQ Consumer start warning (non-fatal): %v", err)
	}

	// ---------------------------------------------------------
	// 7. ROUTER SETUP
	// ---------------------------------------------------------
	r := router.SetupRoutes(
		authHandler,
		groupHandler,
		hotelHandler,
		roomHandler,
		staffHandler,
		guestHandler,
		onboardHandler,
		incHandler,
		smsHandler,
		wsHandlerInc,
		redisClient,
		cfg.JWTSecret,
	)

	// ---------------------------------------------------------
	// 8. SERVER & GRACEFUL SHUTDOWN
	// ---------------------------------------------------------
	srv := &http.Server{
		Addr:         ":" + cfg.ServerPort,
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		log.Printf("🚀 Server running on http://localhost:%s", cfg.ServerPort)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server startup failed: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	<-quit
	log.Println("🛑 Shutting down server gracefully...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("💤 Server exited properly.")
}
