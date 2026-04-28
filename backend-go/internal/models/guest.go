package models

import (
	"time"

	"github.com/google/uuid"
)

// Guest represents an active guest session in a hotel room.
type Guest struct {
	ID                string    `json:"id" db:"id"`                  // "g_{random_hex}"
	RoomID            uuid.UUID `json:"room_id" db:"room_id"`
	HotelID           uuid.UUID `json:"hotel_id" db:"hotel_id"`
	GuestName         string    `json:"guest_name" db:"guest_name"`
	DeviceFingerprint string    `json:"device_fingerprint,omitempty" db:"device_fingerprint"`
	ExpectedCheckout  time.Time `json:"expected_checkout" db:"expected_checkout"`
	BLEPaired         bool      `json:"ble_paired" db:"ble_paired"`
	CreatedAt         time.Time `json:"created_at" db:"created_at"`
}
