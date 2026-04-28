package guest

import (
	"time"

	"github.com/google/uuid"
)

// --- Check-in DTOs ---

type CheckInRequest struct {
	GuestName        string    `json:"guest_name"`
	ExpectedCheckout time.Time `json:"expected_checkout"`
}

type CheckInResponse struct {
	Status string      `json:"status"`
	Data   CheckInData `json:"data"`
}

type CheckInData struct {
	RoomID  uuid.UUID `json:"room_id"`
	GuestID string    `json:"guest_id"`
	Message string    `json:"message"`
}

// --- Check-out DTOs ---

type CheckOutResponse struct {
	Status  string `json:"status"`
	Message string `json:"message"`
}
