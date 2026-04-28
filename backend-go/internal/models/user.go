package models

import (
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// User represents the dashboard administrators and managers in the database
type User struct {
	ID           uuid.UUID  `json:"id" db:"id"`
	Email        string     `json:"email" db:"email"`
	PasswordHash string     `json:"-" db:"password_hash"` // Never serialize to JSON
	Name         string     `json:"name" db:"name"`
	Role         string     `json:"role" db:"role"`
	GroupID      *uuid.UUID `json:"group_id,omitempty" db:"group_id"`
	HotelID      *uuid.UUID `json:"hotel_id,omitempty" db:"hotel_id"`
}

// UserClaims defines the payload embedded inside the JWT token.
// Extended with GuestID and RoomID for guest session tokens.
type UserClaims struct {
	UserID  string `json:"user_id"`
	Role    string `json:"role"`
	GroupID string `json:"group_id,omitempty"`
	HotelID string `json:"hotel_id,omitempty"`
	GuestID string `json:"guest_id,omitempty"` // Set for GUEST role tokens
	RoomID  string `json:"room_id,omitempty"`  // Set for GUEST role tokens
	jwt.RegisteredClaims
}
