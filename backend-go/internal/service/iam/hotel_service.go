package iam

import (
	"context"
	"errors"

	iamDto "github.com/google-hackathon/rapid-response/internal/api/dto/iam"
	iamRepo "github.com/google-hackathon/rapid-response/internal/repository/iam"
)

type HotelService struct {
	repo *iamRepo.HotelRepository
}

func NewHotelService(repo *iamRepo.HotelRepository) *HotelService {
	return &HotelService{repo: repo}
}

func (s *HotelService) CreateHotel(ctx context.Context, req iamDto.CreateHotelRequest) (iamDto.CreateHotelResponse, error) {
	if req.Name == "" {
		return iamDto.CreateHotelResponse{}, errors.New("hotel name is required")
	}
	if req.GroupID.String() == "00000000-0000-0000-0000-000000000000" {
		return iamDto.CreateHotelResponse{}, errors.New("group_id is required")
	}

	hotelID, err := s.repo.CreateHotel(ctx, req.GroupID, req.Name, req.Address, req.Timezone, req.Coordinates.Lat, req.Coordinates.Lon)
	if err != nil {
		return iamDto.CreateHotelResponse{}, err
	}

	return iamDto.CreateHotelResponse{
		Status: "success",
		Data: iamDto.HotelData{
			HotelID: hotelID,
			Message: "Hotel created successfully.",
		},
	}, nil
}
