package iam

import (
	"context"
	"errors"

	iamDto "github.com/google-hackathon/rapid-response/internal/api/dto/iam"
	iamRepo "github.com/google-hackathon/rapid-response/internal/repository/iam"
)

type GroupService struct {
	repo *iamRepo.GroupRepository
}

func NewGroupService(repo *iamRepo.GroupRepository) *GroupService {
	return &GroupService{repo: repo}
}

func (s *GroupService) CreateGroup(ctx context.Context, req iamDto.CreateGroupRequest) (iamDto.CreateGroupResponse, error) {
	// 1. Business Logic / Validation
	if req.Name == "" || req.ContactEmail == "" {
		return iamDto.CreateGroupResponse{}, errors.New("name and contact email are required")
	}

	// 2. Call Repository
	groupID, err := s.repo.CreateGroup(ctx, req.Name, req.ContactEmail)
	if err != nil {
		return iamDto.CreateGroupResponse{}, err
	}

	// 3. Format DTO Response
	return iamDto.CreateGroupResponse{
		Status: "success",
		Data: iamDto.GroupData{
			GroupID: groupID,
			Message: "Hotel group created successfully.",
		},
	}, nil
}
