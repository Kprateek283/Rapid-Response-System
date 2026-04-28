package iam

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

type StaffRepository struct {
	db    *pgxpool.Pool
	redis *redis.Client
}

func NewStaffRepository(db *pgxpool.Pool, redisClient *redis.Client) *StaffRepository {
	return &StaffRepository{db: db, redis: redisClient}
}

// BulkCreateStaff inserts multiple staff members and initializes their Redis availability status.
func (r *StaffRepository) BulkCreateStaff(ctx context.Context, hotelID uuid.UUID, staffEntries []struct {
	Name         string
	PhoneNumber  string
	Role         string
	SkillProfile interface{}
}) ([]uuid.UUID, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	query := `
		INSERT INTO staff (hotel_id, name, phone_number, role, skill_profile)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id
	`

	var ids []uuid.UUID
	for _, s := range staffEntries {
		skillJSON, err := json.Marshal(s.SkillProfile)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal skill_profile for %s: %w", s.Name, err)
		}

		var id uuid.UUID
		err = tx.QueryRow(ctx, query, hotelID, s.Name, s.PhoneNumber, s.Role, skillJSON).Scan(&id)
		if err != nil {
			return nil, fmt.Errorf("failed to insert staff %s: %w", s.Name, err)
		}
		ids = append(ids, id)

		// Set Redis availability status for the new staff member
		redisKey := fmt.Sprintf("staff:%s:%s:status", hotelID.String(), id.String())
		if err := r.redis.Set(ctx, redisKey, "AVAILABLE", 0).Err(); err != nil {
			return nil, fmt.Errorf("failed to set Redis status for staff %s: %w", s.Name, err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return ids, nil
}

// GetStaffByHotelAndRole retrieves staff members for spatial dispatch queries.
func (r *StaffRepository) GetStaffByHotelAndRole(ctx context.Context, hotelID uuid.UUID, role string) ([]struct {
	ID           uuid.UUID
	Name         string
	Role         string
	SkillProfile json.RawMessage
}, error) {
	query := `
		SELECT id, name, role, skill_profile 
		FROM staff 
		WHERE hotel_id = $1 AND role = $2
	`
	rows, err := r.db.Query(ctx, query, hotelID, role)
	if err != nil {
		return nil, fmt.Errorf("failed to query staff: %w", err)
	}
	defer rows.Close()

	var result []struct {
		ID           uuid.UUID
		Name         string
		Role         string
		SkillProfile json.RawMessage
	}

	for rows.Next() {
		var s struct {
			ID           uuid.UUID
			Name         string
			Role         string
			SkillProfile json.RawMessage
		}
		if err := rows.Scan(&s.ID, &s.Name, &s.Role, &s.SkillProfile); err != nil {
			return nil, err
		}
		result = append(result, s)
	}

	return result, nil
}
