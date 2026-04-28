package middleware

import (
	"net/http"

	"github.com/google-hackathon/rapid-response/internal/models"
)

// RequireRole enforces strict RBAC allowing multiple roles (e.g., "SUPER_ADMIN", "GROUP_MANAGER")
func RequireRole(requiredRoles ...string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Extract claims from context (set by RequireAuth)
			claims, ok := r.Context().Value(ClaimsKey).(*models.UserClaims)
			if !ok {
				http.Error(w, `{"error":"Unauthorized: Missing or invalid token context"}`, http.StatusUnauthorized)
				return
			}

			// Check if the user's role matches any of the permitted roles
			hasAccess := false
			for _, role := range requiredRoles {
				if claims.Role == role {
					hasAccess = true
					break
				}
			}

			if !hasAccess {
				http.Error(w, `{"error":"Forbidden: insufficient permissions"}`, http.StatusForbidden)
				return
			}

			// User has the correct role, pass to the handler
			next.ServeHTTP(w, r)
		})
	}
}
