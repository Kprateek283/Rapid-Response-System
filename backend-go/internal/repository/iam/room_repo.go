package iam

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type RoomRepository struct {
	db *pgxpool.Pool
}

func NewRoomRepository(db *pgxpool.Pool) *RoomRepository {
	return &RoomRepository{db: db}
}

// BulkCreateRooms inserts multiple rooms in a single transaction.
// Rolls back the entire batch on any conflict (e.g., duplicate qr_index or room_number).
func (r *RoomRepository) BulkCreateRooms(ctx context.Context, hotelID uuid.UUID, rooms []struct {
	RoomNumber string
	FloorLevel int
	QRIndex    int
}) ([]uuid.UUID, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	query := `
		INSERT INTO rooms (hotel_id, room_number, floor_level, qr_index)
		VALUES ($1, $2, $3, $4)
		RETURNING id
	`

	var ids []uuid.UUID
	for _, room := range rooms {
		var id uuid.UUID
		err := tx.QueryRow(ctx, query, hotelID, room.RoomNumber, room.FloorLevel, room.QRIndex).Scan(&id)
		if err != nil {
			return nil, fmt.Errorf("failed to insert room %s: %w", room.RoomNumber, err)
		}
		ids = append(ids, id)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return ids, nil
}

// GetRoomByID retrieves a single room by ID.
func (r *RoomRepository) GetRoomByID(ctx context.Context, roomID uuid.UUID) (uuid.UUID, string, bool, error) {
	var id uuid.UUID
	var roomNumber string
	var isOccupied bool
	query := `SELECT id, room_number, is_occupied FROM rooms WHERE id = $1`
	err := r.db.QueryRow(ctx, query, roomID).Scan(&id, &roomNumber, &isOccupied)
	if err != nil {
		return uuid.Nil, "", false, fmt.Errorf("room not found: %w", err)
	}
	return id, roomNumber, isOccupied, nil
}

// SetRoomOccupied updates the is_occupied flag for a room.
func (r *RoomRepository) SetRoomOccupied(ctx context.Context, roomID uuid.UUID, occupied bool) error {
	query := `UPDATE rooms SET is_occupied = $1 WHERE id = $2`
	_, err := r.db.Exec(ctx, query, occupied, roomID)
	return err
}

// GetRoomByQRIndex finds a room by its unique QR index.
func (r *RoomRepository) GetRoomByQRIndex(ctx context.Context, roomNumber string, qrIndex int) (uuid.UUID, uuid.UUID, bool, error) {
	var roomID, hotelID uuid.UUID
	var isOccupied bool
	query := `SELECT id, hotel_id, is_occupied FROM rooms WHERE room_number = $1 AND qr_index = $2`
	err := r.db.QueryRow(ctx, query, roomNumber, qrIndex).Scan(&roomID, &hotelID, &isOccupied)
	if err != nil {
		return uuid.Nil, uuid.Nil, false, fmt.Errorf("room not found for qr_index: %w", err)
	}
	return roomID, hotelID, isOccupied, nil
}
