package router

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	chiMiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/google-hackathon/rapid-response/internal/api/handlers/auth"
	"github.com/google-hackathon/rapid-response/internal/api/handlers/guest"
	"github.com/google-hackathon/rapid-response/internal/api/handlers/iam"
	incidentHandler "github.com/google-hackathon/rapid-response/internal/api/handlers/incident"
	"github.com/google-hackathon/rapid-response/internal/api/handlers/onboard"
	"github.com/google-hackathon/rapid-response/internal/api/middleware"
	"github.com/redis/go-redis/v9"
)

// SetupRoutes configures all application routes with dependency-injected handlers.
func SetupRoutes(
	authHandler *auth.AuthHandler,
	groupHandler *iam.GroupHandler,
	hotelHandler *iam.HotelHandler,
	roomHandler *iam.RoomHandler,
	staffHandler *iam.StaffHandler,
	guestHandler *guest.GuestHandler,
	onboardHandler *onboard.OnboardHandler,
	incHandler *incidentHandler.IncidentHandler,
	smsHandler *incidentHandler.SMSHandler,
	wsHandler *incidentHandler.WSHandler,
	redisClient *redis.Client,
	jwtSecret string,
) *chi.Mux {
	r := chi.NewRouter()

	r.Use(chiMiddleware.Logger)
	r.Use(chiMiddleware.Recoverer)
	r.Use(chiMiddleware.RealIP)

	// CORS middleware for Flutter web
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
			if r.Method == "OPTIONS" {
				w.WriteHeader(http.StatusOK)
				return
			}
			next.ServeHTTP(w, r)
		})
	})

	r.Get("/ping", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":"ok","service":"rapid-crisis-response"}`))
	})

	r.Route("/api/v1", func(r chi.Router) {

		// ──────────────────────────────────────────────
		// AUTH ROUTES
		// ──────────────────────────────────────────────
		r.Route("/auth", func(r chi.Router) {
			r.Post("/login", authHandler.Login)

			r.Group(func(r chi.Router) {
				r.Use(middleware.RequireAuth(redisClient, jwtSecret))
				r.Post("/logout", authHandler.Logout)
				r.With(middleware.RequireRole("SUPER_ADMIN")).
					Post("/create-group-manager", authHandler.CreateGroupManager)
				r.With(middleware.RequireRole("SUPER_ADMIN", "GROUP_MANAGER")).
					Post("/create-hotel-manager", authHandler.CreateHotelManager)
			})
		})

		// ──────────────────────────────────────────────
		// ADMIN / IAM ROUTES
		// ──────────────────────────────────────────────
		r.Route("/admin", func(r chi.Router) {
			r.Use(middleware.RequireAuth(redisClient, jwtSecret))

			r.With(middleware.RequireRole("SUPER_ADMIN")).
				Post("/groups", groupHandler.CreateGroup)

			r.With(middleware.RequireRole("SUPER_ADMIN", "GROUP_MANAGER")).
				Post("/hotels", hotelHandler.CreateHotel)

			r.Route("/hotels/{hotel_id}", func(r chi.Router) {
				r.With(middleware.RequireRole("HOTEL_MANAGER")).
					Post("/rooms/bulk", roomHandler.BulkCreateRooms)
				r.With(middleware.RequireRole("HOTEL_MANAGER")).
					Post("/staff/bulk", staffHandler.BulkCreateStaff)

				r.Route("/rooms/{room_id}", func(r chi.Router) {
					r.With(middleware.RequireRole("HOTEL_MANAGER")).
						Post("/checkin", guestHandler.CheckIn)
					r.With(middleware.RequireRole("HOTEL_MANAGER")).
						Post("/checkout", guestHandler.CheckOut)
				})
			})
		})

		// ──────────────────────────────────────────────
		// ONBOARD ROUTES (Guest QR Scan + BLE)
		// ──────────────────────────────────────────────
		r.Route("/onboard", func(r chi.Router) {
			r.Post("/scan", onboardHandler.Scan)

			r.Group(func(r chi.Router) {
				r.Use(middleware.RequireAuth(redisClient, jwtSecret))
				r.With(middleware.RequireRole("GUEST")).
					Post("/pair-status", onboardHandler.PairStatus)
			})
		})

		// ──────────────────────────────────────────────
		// INCIDENT ROUTES
		// ──────────────────────────────────────────────
		r.Route("/incident", func(r chi.Router) {
			// Public: SMS webhook (Twilio calls this)
			r.Post("/trigger/sms-webhook", smsHandler.HandleSMSWebhook)

			// Protected routes
			r.Group(func(r chi.Router) {
				r.Use(middleware.RequireAuth(redisClient, jwtSecret))

				// Trigger init — GUEST only
				r.With(middleware.RequireRole("GUEST")).
					Post("/trigger/init", incHandler.TriggerInit)

				// Incident-scoped routes
				r.Route("/{incident_id}", func(r chi.Router) {
					// Media ready — GUEST or SYSTEM
					r.Post("/media-ready", incHandler.MediaReady)

					// Dispatch accept/decline — staff/managers
					r.Post("/dispatch/accept", incHandler.DispatchAccept)
					r.Post("/dispatch/decline", incHandler.DispatchDecline)

					// Transit GPS ping
					r.Post("/transit/ping", incHandler.TransitPing)

					// Staff arrival + ground truth
					r.Post("/staff/arrive", incHandler.StaffArrive)
					r.Post("/staff/ground-truth", incHandler.GroundTruth)

					// Branching actions
					r.Post("/action/backup", incHandler.RequestBackup)
					r.Post("/action/escalate", incHandler.Escalate)
					r.Post("/action/escalate/abort", incHandler.EscalateAbort)

					// Resolution handshake
					r.Post("/action/resolve/propose", incHandler.ProposeResolution)
					r.Post("/action/resolve/confirm", incHandler.ConfirmResolution)
				})

				// WebSocket live stream
				r.Get("/ws/{incident_id}/live", wsHandler.HandleWSUpgrade)
			})
		})
	})

	return r
}
