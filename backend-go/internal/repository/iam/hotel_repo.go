package iam

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type HotelRepository struct {
	db *pgxpool.Pool
}

func NewHotelRepository(db *pgxpool.Pool) *HotelRepository {
	return &HotelRepository{db: db}
}

// CreateHotel inserts a hotel with PostGIS coordinates.
func (r *HotelRepository) CreateHotel(ctx context.Context, groupID uuid.UUID, name, address, timezone string, lat, lon float64) (uuid.UUID, error) {
	var hotelID uuid.UUID

	query := `
		INSERT INTO hotels (group_id, name, address, timezone, coordinates) 
		VALUES ($1, $2, $3, $4, ST_SetSRID(ST_MakePoint($5, $6), 4326)) 
		RETURNING id
	`

	err := r.db.QueryRow(ctx, query, groupID, name, address, timezone, lon, lat).Scan(&hotelID)
	if err != nil {
		return uuid.Nil, fmt.Errorf("failed to insert hotel: %w", err)
	}

	return hotelID, nil
}

// GetHotelByID retrieves a hotel by ID (used for validation in other handlers).
func (r *HotelRepository) GetHotelByID(ctx context.Context, hotelID uuid.UUID) (uuid.UUID, string, error) {
	var id uuid.UUID
	var name string
	query := `SELECT id, name FROM hotels WHERE id = $1`
	err := r.db.QueryRow(ctx, query, hotelID).Scan(&id, &name)
	if err != nil {
		return uuid.Nil, "", fmt.Errorf("hotel not found: %w", err)
	}
	return id, name, nil
}
