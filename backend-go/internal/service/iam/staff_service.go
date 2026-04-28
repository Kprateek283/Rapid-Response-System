package iam

import (
	"context"
	"errors"

	iamDto "github.com/google-hackathon/rapid-response/internal/api/dto/iam"
	iamRepo "github.com/google-hackathon/rapid-response/internal/repository/iam"
	"github.com/google/uuid"
)

type StaffService struct {
	repo *iamRepo.StaffRepository
}

func NewStaffService(repo *iamRepo.StaffRepository) *StaffService {
	return &StaffService{repo: repo}
}

func (s *StaffService) BulkCreateStaff(ctx context.Context, hotelID uuid.UUID, req iamDto.BulkCreateStaffRequest) (iamDto.BulkCreateStaffResponse, error) {
	if len(req.Staff) == 0 {
		return iamDto.BulkCreateStaffResponse{}, errors.New("staff array cannot be empty")
	}
	if req.Count != len(req.Staff) {
		return iamDto.BulkCreateStaffResponse{}, errors.New("count does not match staff array length")
	}

	// Convert DTO entries to repo format
	entries := make([]struct {
		Name         string
		PhoneNumber  string
		Role         string
		SkillProfile interface{}
	}, len(req.Staff))

	for i, st := range req.Staff {
		entries[i] = struct {
			Name         string
			PhoneNumber  string
			Role         string
			SkillProfile interface{}
		}{
			Name:         st.Name,
			PhoneNumber:  st.PhoneNumber,
			Role:         st.Role,
			SkillProfile: st.SkillProfile,
		}
	}

	ids, err := s.repo.BulkCreateStaff(ctx, hotelID, entries)
	if err != nil {
		return iamDto.BulkCreateStaffResponse{}, err
	}

	return iamDto.BulkCreateStaffResponse{
		Status: "success",
		Data: iamDto.StaffData{
			InsertedCount: len(ids),
			StaffIDs:      ids,
			Message:       "Staff created successfully.",
		},
	}, nil
}
