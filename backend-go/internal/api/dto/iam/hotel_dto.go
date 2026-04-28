package iam

import "github.com/google/uuid"

// --- Hotel DTOs ---

type Coordinates struct {
	Lat float64 `json:"lat"`
	Lon float64 `json:"lon"`
}

type CreateHotelRequest struct {
	GroupID     uuid.UUID   `json:"group_id"`
	Name       string      `json:"name"`
	Address    string      `json:"address"`
	Timezone   string      `json:"timezone"`
	Coordinates Coordinates `json:"coordinates"`
}

type CreateHotelResponse struct {
	Status string    `json:"status"`
	Data   HotelData `json:"data"`
}

type HotelData struct {
	HotelID uuid.UUID `json:"hotel_id"`
	Message string    `json:"message"`
}
