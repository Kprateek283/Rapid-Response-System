package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google-hackathon/rapid-response/internal/models"
	"github.com/redis/go-redis/v9"
)

type contextKey string

const ClaimsKey contextKey = "user_claims"

// RequireAuth validates JWT tokens and checks the Redis blacklist.
// The JWT secret is injected via parameter to avoid hardcoded globals.
func RequireAuth(redisClient *redis.Client, jwtSecret string) func(http.Handler) http.Handler {
	secret := []byte(jwtSecret)
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
				http.Error(w, `{"status":"error","message":"Unauthorized"}`, http.StatusUnauthorized)
				return
			}

			tokenString := strings.TrimPrefix(authHeader, "Bearer ")

			// 1. Check if token signature is in Redis Blacklist
			parts := strings.Split(tokenString, ".")
			if len(parts) == 3 {
				signature := parts[2]
				isBlacklisted, _ := redisClient.Exists(r.Context(), "blacklist:"+signature).Result()
				if isBlacklisted > 0 {
					http.Error(w, `{"status":"error","message":"Token has been revoked/logged out"}`, http.StatusUnauthorized)
					return
				}
			}

			// 2. Parse and verify JWT (existing logic)
			claims := &models.UserClaims{}
			token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
				return secret, nil
			})

			if err != nil || !token.Valid {
				http.Error(w, `{"status":"error","message":"Invalid token"}`, http.StatusUnauthorized)
				return
			}

			ctx := context.WithValue(r.Context(), ClaimsKey, claims)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
