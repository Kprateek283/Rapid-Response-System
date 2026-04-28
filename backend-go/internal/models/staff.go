package models

import (
	"github.com/google/uuid"
)

type SkillProfile struct {
	FirstAid     float64  `json:"first_aid"`
	Combat       float64  `json:"combat"`
	CrowdControl float64  `json:"crowd_control"`
	Languages    []string `json:"languages"`
}

type Staff struct {
	ID           uuid.UUID    `json:"id" db:"id"`
	HotelID      uuid.UUID    `json:"hotel_id" db:"hotel_id"`
	Name         string       `json:"name" db:"name"`
	PhoneNumber  string       `json:"phone_number" db:"phone_number"`
	Role         string       `json:"role" db:"role"`
	SkillProfile SkillProfile `json:"skill_profile" db:"skill_profile"`
}
