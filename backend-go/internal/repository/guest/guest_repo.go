package guest

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type GuestRepository struct {
	db *pgxpool.Pool
}

func NewGuestRepository(db *pgxpool.Pool) *GuestRepository {
	return &GuestRepository{db: db}
}

// generateGuestID creates an ID in the format "g_{random_hex(6)}"
func generateGuestID() (string, error) {
	bytes := make([]byte, 3) // 3 bytes = 6 hex chars
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return "g_" + hex.EncodeToString(bytes), nil
}

// CreateGuest inserts a new guest session row.
func (r *GuestRepository) CreateGuest(ctx context.Context, roomID, hotelID uuid.UUID, guestName string, expectedCheckout time.Time) (string, error) {
	guestID, err := generateGuestID()
	if err != nil {
		return "", fmt.Errorf("failed to generate guest ID: %w", err)
	}

	query := `
		INSERT INTO guests (id, room_id, hotel_id, guest_name, expected_checkout)
		VALUES ($1, $2, $3, $4, $5)
	`
	_, err = r.db.Exec(ctx, query, guestID, roomID, hotelID, guestName, expectedCheckout)
	if err != nil {
		return "", fmt.Errorf("failed to create guest: %w", err)
	}

	return guestID, nil
}

// GetGuestByRoomID retrieves the active guest for a given room.
func (r *GuestRepository) GetGuestByRoomID(ctx context.Context, roomID uuid.UUID) (string, uuid.UUID, time.Time, error) {
	var guestID string
	var hotelID uuid.UUID
	var expectedCheckout time.Time

	query := `SELECT id, hotel_id, expected_checkout FROM guests WHERE room_id = $1 ORDER BY created_at DESC LIMIT 1`
	err := r.db.QueryRow(ctx, query, roomID).Scan(&guestID, &hotelID, &expectedCheckout)
	if err != nil {
		return "", uuid.Nil, time.Time{}, fmt.Errorf("no active guest for room: %w", err)
	}
	return guestID, hotelID, expectedCheckout, nil
}

// DeleteGuestByRoomID removes the guest session for a room (checkout).
func (r *GuestRepository) DeleteGuestByRoomID(ctx context.Context, roomID uuid.UUID) error {
	query := `DELETE FROM guests WHERE room_id = $1`
	_, err := r.db.Exec(ctx, query, roomID)
	return err
}

// UpdateDeviceFingerprint sets the device fingerprint for a guest.
func (r *GuestRepository) UpdateDeviceFingerprint(ctx context.Context, guestID, fingerprint string) error {
	query := `UPDATE guests SET device_fingerprint = $1 WHERE id = $2`
	_, err := r.db.Exec(ctx, query, fingerprint, guestID)
	return err
}

// SetBLEPaired updates the BLE pairing status for a guest.
func (r *GuestRepository) SetBLEPaired(ctx context.Context, guestID string, paired bool) error {
	query := `UPDATE guests SET ble_paired = $1 WHERE id = $2`
	_, err := r.db.Exec(ctx, query, paired, guestID)
	return err
}
