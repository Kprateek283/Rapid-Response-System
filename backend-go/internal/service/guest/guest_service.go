package guest

import (
	"context"
	"errors"

	guestDto "github.com/google-hackathon/rapid-response/internal/api/dto/guest"
	guestRepo "github.com/google-hackathon/rapid-response/internal/repository/guest"
	iamRepo "github.com/google-hackathon/rapid-response/internal/repository/iam"
	authService "github.com/google-hackathon/rapid-response/internal/service/auth"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

type GuestService struct {
	guestRepo *guestRepo.GuestRepository
	roomRepo  *iamRepo.RoomRepository
	authSvc   *authService.AuthService
	redis     *redis.Client
}

func NewGuestService(
	gr *guestRepo.GuestRepository,
	rr *iamRepo.RoomRepository,
	as *authService.AuthService,
	rc *redis.Client,
) *GuestService {
	return &GuestService{
		guestRepo: gr,
		roomRepo:  rr,
		authSvc:   as,
		redis:     rc,
	}
}

func (s *GuestService) CheckIn(ctx context.Context, hotelID, roomID uuid.UUID, req guestDto.CheckInRequest) (guestDto.CheckInResponse, error) {
	if req.GuestName == "" {
		return guestDto.CheckInResponse{}, errors.New("guest_name is required")
	}
	if req.ExpectedCheckout.IsZero() {
		return guestDto.CheckInResponse{}, errors.New("expected_checkout is required")
	}

	// Verify room exists and get current state
	_, _, isOccupied, err := s.roomRepo.GetRoomByID(ctx, roomID)
	if err != nil {
		return guestDto.CheckInResponse{}, errors.New("room not found")
	}
	if isOccupied {
		return guestDto.CheckInResponse{}, errors.New("room is already occupied")
	}

	// Mark room as occupied
	if err := s.roomRepo.SetRoomOccupied(ctx, roomID, true); err != nil {
		return guestDto.CheckInResponse{}, errors.New("failed to update room status")
	}

	// Create guest session
	guestID, err := s.guestRepo.CreateGuest(ctx, roomID, hotelID, req.GuestName, req.ExpectedCheckout)
	if err != nil {
		// Rollback room status on failure
		_ = s.roomRepo.SetRoomOccupied(ctx, roomID, false)
		return guestDto.CheckInResponse{}, errors.New("failed to create guest session")
	}

	return guestDto.CheckInResponse{
		Status: "success",
		Data: guestDto.CheckInData{
			RoomID:  roomID,
			GuestID: guestID,
			Message: "Room marked as occupied.",
		},
	}, nil
}

func (s *GuestService) CheckOut(ctx context.Context, hotelID, roomID uuid.UUID) (guestDto.CheckOutResponse, error) {
	// Verify room exists and is occupied
	_, _, isOccupied, err := s.roomRepo.GetRoomByID(ctx, roomID)
	if err != nil {
		return guestDto.CheckOutResponse{}, errors.New("room not found")
	}
	if !isOccupied {
		return guestDto.CheckOutResponse{}, errors.New("room is not currently occupied")
	}

	// Get guest to find their JWT for blacklisting
	guestID, _, _, err := s.guestRepo.GetGuestByRoomID(ctx, roomID)
	if err == nil {
		// Blacklist guest session in Redis
		sessionKey := "guest:" + guestID + ":session"
		s.redis.Del(ctx, sessionKey)
	}

	// Delete guest row
	if err := s.guestRepo.DeleteGuestByRoomID(ctx, roomID); err != nil {
		return guestDto.CheckOutResponse{}, errors.New("failed to remove guest session")
	}

	// Mark room as vacant
	if err := s.roomRepo.SetRoomOccupied(ctx, roomID, false); err != nil {
		return guestDto.CheckOutResponse{}, errors.New("failed to update room status")
	}

	return guestDto.CheckOutResponse{
		Status:  "success",
		Message: "Room reset to vacant. Previous guest session terminated.",
	}, nil
}
