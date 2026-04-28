package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/twpayne/go-geom"
)

type HotelGroup struct {
	ID           uuid.UUID `json:"id" db:"id"`
	Name         string    `json:"name" db:"name"`
	ContactEmail string    `json:"contact_email" db:"contact_email"`
	CreatedAt    time.Time `json:"created_at" db:"created_at"`
}

type Hotel struct {
	ID          uuid.UUID   `json:"id" db:"id"`
	GroupID     uuid.UUID   `json:"group_id" db:"group_id"`
	Name        string      `json:"name" db:"name"`
	Address     string      `json:"address" db:"address"`
	Coordinates *geom.Point `json:"coordinates" db:"coordinates"`
	Timezone    string      `json:"timezone" db:"timezone"`
}

type Room struct {
	ID         uuid.UUID `json:"id" db:"id"`
	HotelID    uuid.UUID `json:"hotel_id" db:"hotel_id"`
	RoomNumber string    `json:"room_number" db:"room_number"`
	FloorLevel int       `json:"floor_level" db:"floor_level"`
	QRIndex    int       `json:"qr_index" db:"qr_index"`
	IsOccupied bool      `json:"is_occupied" db:"is_occupied"`
}
