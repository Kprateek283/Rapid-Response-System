package iam

import "github.com/google/uuid"

// CreateGroupRequest Request payload from Flutter
type CreateGroupRequest struct {
	Name         string `json:"name"`
	ContactEmail string `json:"contact_email"`
}

// CreateGroupResponse Response payload to Flutter
type CreateGroupResponse struct {
	Status string    `json:"status"`
	Data   GroupData `json:"data"`
}

type GroupData struct {
	GroupID uuid.UUID `json:"group_id"`
	Message string    `json:"message"`
}
