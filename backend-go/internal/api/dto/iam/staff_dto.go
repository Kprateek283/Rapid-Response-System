package iam

import "github.com/google/uuid"

// --- Staff DTOs ---

type SkillProfileDTO struct {
	FirstAid     float64  `json:"first_aid"`
	Combat       float64  `json:"combat"`
	CrowdControl float64  `json:"crowd_control"`
	Languages    []string `json:"languages"`
}

type StaffEntry struct {
	Name         string          `json:"name"`
	PhoneNumber  string          `json:"phone_number"`
	Role         string          `json:"role"`
	SkillProfile SkillProfileDTO `json:"skill_profile"`
}

type BulkCreateStaffRequest struct {
	Count int          `json:"count"`
	Staff []StaffEntry `json:"staff"`
}

type BulkCreateStaffResponse struct {
	Status string    `json:"status"`
	Data   StaffData `json:"data"`
}

type StaffData struct {
	InsertedCount int       `json:"inserted_count"`
	StaffIDs      []uuid.UUID `json:"staff_ids,omitempty"`
	Message       string    `json:"message"`
}
