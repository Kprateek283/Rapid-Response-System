package iam

import "github.com/google/uuid"

// --- Room DTOs ---

type RoomEntry struct {
	RoomNumber string `json:"room_number"`
	FloorLevel int    `json:"floor_level"`
	QRIndex    int    `json:"qr_index"`
}

type BulkCreateRoomRequest struct {
	Count int         `json:"count"`
	Rooms []RoomEntry `json:"rooms"`
}

type BulkCreateRoomResponse struct {
	Status string   `json:"status"`
	Data   RoomData `json:"data"`
}

type RoomData struct {
	InsertedCount int       `json:"inserted_count"`
	RoomIDs       []uuid.UUID `json:"room_ids,omitempty"`
	Message       string    `json:"message"`
}
