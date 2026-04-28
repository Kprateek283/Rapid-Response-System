package auth

import "github.com/google/uuid"

// LoginRequest --- LOGIN ---
type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type LoginResponse struct {
	Token   string `json:"token"`
	Role    string `json:"role"`
	Message string `json:"message"`
}

// CreateManagerRequest --- CREATE MANAGERS ---
type CreateManagerRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"` // In production, this might be auto-generated and emailed
	Name     string `json:"name"`
	// If creating a Group Manager, this is required. If creating a Hotel Manager, the creator's GroupID is inferred.
	GroupID uuid.UUID `json:"group_id,omitempty"`
	// If creating a Hotel Manager, this is required.
	HotelID uuid.UUID `json:"hotel_id,omitempty"`
}

type CreateManagerResponse struct {
	UserID  uuid.UUID `json:"user_id"`
	Role    string    `json:"role"`
	Message string    `json:"message"`
}
