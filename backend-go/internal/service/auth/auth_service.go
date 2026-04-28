package auth

import (
	"context"
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
	authDto "github.com/google-hackathon/rapid-response/internal/api/dto/auth"
	"github.com/google-hackathon/rapid-response/internal/models"
	authRepo "github.com/google-hackathon/rapid-response/internal/repository/auth"
	"golang.org/x/crypto/bcrypt"
)

type AuthService struct {
	repo      *authRepo.AuthRepository
	jwtSecret []byte
}

func NewAuthService(repo *authRepo.AuthRepository, jwtSecret string) *AuthService {
	return &AuthService{repo: repo, jwtSecret: []byte(jwtSecret)}
}

func (s *AuthService) Authenticate(ctx context.Context, req authDto.LoginRequest) (authDto.LoginResponse, error) {
	// 1. Fetch User
	user, err := s.repo.GetUserByEmail(ctx, req.Email)
	if err != nil {
		return authDto.LoginResponse{}, errors.New("invalid credentials")
	}

	// 2. Verify Password
	err = bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password))
	if err != nil {
		return authDto.LoginResponse{}, errors.New("invalid credentials")
	}

	// 3. Generate JWT
	expirationTime := time.Now().Add(24 * time.Hour)
	claims := &models.UserClaims{
		UserID: user.ID.String(),
		Role:   user.Role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expirationTime),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	// Safely map UUID pointers if they exist
	if user.GroupID != nil {
		claims.GroupID = user.GroupID.String()
	}
	if user.HotelID != nil {
		claims.HotelID = user.HotelID.String()
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(s.jwtSecret)
	if err != nil {
		return authDto.LoginResponse{}, errors.New("failed to generate token")
	}

	return authDto.LoginResponse{
		Token:   tokenString,
		Role:    user.Role,
		Message: "Login successful",
	}, nil
}

func (s *AuthService) CreateUser(ctx context.Context, req authDto.CreateManagerRequest, role string) (authDto.CreateManagerResponse, error) {
	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return authDto.CreateManagerResponse{}, errors.New("failed to process password")
	}

	user := &models.User{
		Email:        req.Email,
		PasswordHash: string(hashedPassword),
		Name:         req.Name,
		Role:         role,
	}

	// Map relational UUIDs if provided by the DTO
	if req.GroupID.String() != "00000000-0000-0000-0000-000000000000" {
		user.GroupID = &req.GroupID
	}
	if req.HotelID.String() != "00000000-0000-0000-0000-000000000000" {
		user.HotelID = &req.HotelID
	}

	newID, err := s.repo.CreateUser(ctx, user)
	if err != nil {
		return authDto.CreateManagerResponse{}, errors.New("failed to create user, email might already exist")
	}

	return authDto.CreateManagerResponse{
		UserID:  newID,
		Role:    role,
		Message: "User created successfully",
	}, nil
}

func (s *AuthService) BlacklistToken(ctx context.Context, tokenString string) error {
	// Parse the token purely to extract the expiration time
	token, _ := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		return s.jwtSecret, nil
	})

	// Even if the token is mathematically invalid, if we can parse claims, we get the exp
	if claims, ok := token.Claims.(jwt.MapClaims); ok {
		if exp, ok := claims["exp"].(float64); ok {
			expirationTime := time.Unix(int64(exp), 0)
			timeUntilExpiry := time.Until(expirationTime)

			if timeUntilExpiry > 0 {
				// Use the token signature (last segment) as the Redis key to save space
				signature := token.Signature
				return s.repo.BlacklistToken(ctx, string(signature), timeUntilExpiry)
			}
		}
	}
	return nil // Token already expired, no need to blacklist
}

// GenerateGuestToken creates a JWT for guest sessions with checkout-based expiration.
func (s *AuthService) GenerateGuestToken(guestID, roomID, hotelID string, expiresAt time.Time) (string, error) {
	claims := &models.UserClaims{
		UserID:  guestID,
		Role:    "GUEST",
		HotelID: hotelID,
		GuestID: guestID,
		RoomID:  roomID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(s.jwtSecret)
}
