package iam

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type GroupRepository struct {
	db *pgxpool.Pool
}

func NewGroupRepository(db *pgxpool.Pool) *GroupRepository {
	return &GroupRepository{db: db}
}

func (r *GroupRepository) CreateGroup(ctx context.Context, name, email string) (uuid.UUID, error) {
	var groupID uuid.UUID

	query := `
		INSERT INTO hotel_groups (name, contact_email) 
		VALUES ($1, $2) 
		RETURNING id
	`

	err := r.db.QueryRow(ctx, query, name, email).Scan(&groupID)
	if err != nil {
		return uuid.Nil, err
	}

	return groupID, nil
}
