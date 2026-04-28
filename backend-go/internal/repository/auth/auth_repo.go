package auth

import (
	"context"
	"time"

	"github.com/google-hackathon/rapid-response/internal/models"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

type AuthRepository struct {
	db    *pgxpool.Pool
	redis *redis.Client
}

func NewAuthRepository(db *pgxpool.Pool, redisClient *redis.Client) *AuthRepository {
	return &AuthRepository{db: db, redis: redisClient}
}

// GetUserByEmail fetches the user for login verification
func (r *AuthRepository) GetUserByEmail(ctx context.Context, email string) (*models.User, error) {
	var user models.User
	query := `
		SELECT id, email, password_hash, name, role, group_id, hotel_id 
		FROM users WHERE email = $1
	`
	err := r.db.QueryRow(ctx, query, email).Scan(
		&user.ID, &user.Email, &user.PasswordHash, &user.Name,
		&user.Role, &user.GroupID, &user.HotelID,
	)
	if err != nil {
		return nil, err // Returns pgx.ErrNoRows if not found
	}
	return &user, nil
}

// CreateUser inserts a new manager/admin into the database
func (r *AuthRepository) CreateUser(ctx context.Context, user *models.User) (uuid.UUID, error) {
	var newID uuid.UUID
	query := `
		INSERT INTO users (email, password_hash, name, role, group_id, hotel_id) 
		VALUES ($1, $2, $3, $4, $5, $6) 
		RETURNING id
	`
	err := r.db.QueryRow(ctx, query,
		user.Email, user.PasswordHash, user.Name,
		user.Role, user.GroupID, user.HotelID,
	).Scan(&newID)

	if err != nil {
		return uuid.Nil, err
	}
	return newID, nil
}

// BlacklistToken pushes a logged-out JWT signature to Redis with a strict TTL
func (r *AuthRepository) BlacklistToken(ctx context.Context, tokenSignature string, expiration time.Duration) error {
	// Prefix key to prevent collisions
	key := "blacklist:" + tokenSignature
	return r.redis.Set(ctx, key, "revoked", expiration).Err()
}
