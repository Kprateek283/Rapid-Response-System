package iam

import (
	"context"
	"errors"

	iamDto "github.com/google-hackathon/rapid-response/internal/api/dto/iam"
	iamRepo "github.com/google-hackathon/rapid-response/internal/repository/iam"
	"github.com/google/uuid"
)

type RoomService struct {
	repo *iamRepo.RoomRepository
}

func NewRoomService(repo *iamRepo.RoomRepository) *RoomService {
	return &RoomService{repo: repo}
}

func (s *RoomService) BulkCreateRooms(ctx context.Context, hotelID uuid.UUID, req iamDto.BulkCreateRoomRequest) (iamDto.BulkCreateRoomResponse, error) {
	if len(req.Rooms) == 0 {
		return iamDto.BulkCreateRoomResponse{}, errors.New("rooms array cannot be empty")
	}
	if req.Count != len(req.Rooms) {
		return iamDto.BulkCreateRoomResponse{}, errors.New("count does not match rooms array length")
	}

	// Convert DTO entries to repo format
	rooms := make([]struct {
		RoomNumber string
		FloorLevel int
		QRIndex    int
	}, len(req.Rooms))

	for i, r := range req.Rooms {
		rooms[i] = struct {
			RoomNumber string
			FloorLevel int
			QRIndex    int
		}{
			RoomNumber: r.RoomNumber,
			FloorLevel: r.FloorLevel,
			QRIndex:    r.QRIndex,
		}
	}

	ids, err := s.repo.BulkCreateRooms(ctx, hotelID, rooms)
	if err != nil {
		return iamDto.BulkCreateRoomResponse{}, err
	}

	return iamDto.BulkCreateRoomResponse{
		Status: "success",
		Data: iamDto.RoomData{
			InsertedCount: len(ids),
			RoomIDs:       ids,
			Message:       "Rooms created successfully.",
		},
	}, nil
}
