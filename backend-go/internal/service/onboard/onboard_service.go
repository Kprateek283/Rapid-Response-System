package onboard

import (
	"context"
	"errors"

	onboardDto "github.com/google-hackathon/rapid-response/internal/api/dto/onboard"
	guestRepo "github.com/google-hackathon/rapid-response/internal/repository/guest"
	iamRepo "github.com/google-hackathon/rapid-response/internal/repository/iam"
	authService "github.com/google-hackathon/rapid-response/internal/service/auth"
)

type OnboardService struct {
	roomRepo  *iamRepo.RoomRepository
	guestRepo *guestRepo.GuestRepository
	authSvc   *authService.AuthService
}

func NewOnboardService(
	rr *iamRepo.RoomRepository,
	gr *guestRepo.GuestRepository,
	as *authService.AuthService,
) *OnboardService {
	return &OnboardService{
		roomRepo:  rr,
		guestRepo: gr,
		authSvc:   as,
	}
}

// Scan validates QR code data, looks up the guest session, and issues a GUEST JWT.
func (s *OnboardService) Scan(ctx context.Context, req onboardDto.ScanRequest) (onboardDto.ScanResponse, error) {
	if req.RoomNumber == "" || req.QRIndex == 0 {
		return onboardDto.ScanResponse{}, errors.New("room_number and qr_index are required")
	}

	// Find room by QR index + room number
	roomID, hotelID, isOccupied, err := s.roomRepo.GetRoomByQRIndex(ctx, req.RoomNumber, req.QRIndex)
	if err != nil {
		return onboardDto.ScanResponse{}, errors.New("invalid QR code: room not found")
	}
	if !isOccupied {
		return onboardDto.ScanResponse{}, errors.New("room is not currently occupied — check in first")
	}

	// Get guest session for this room
	guestID, _, expectedCheckout, err := s.guestRepo.GetGuestByRoomID(ctx, roomID)
	if err != nil {
		return onboardDto.ScanResponse{}, errors.New("no active guest session for this room")
	}

	// Update device fingerprint
	if req.DeviceFingerprint != "" {
		_ = s.guestRepo.UpdateDeviceFingerprint(ctx, guestID, req.DeviceFingerprint)
	}

	// Generate GUEST JWT with checkout-based expiration
	token, err := s.authSvc.GenerateGuestToken(guestID, roomID.String(), hotelID.String(), expectedCheckout)
	if err != nil {
		return onboardDto.ScanResponse{}, errors.New("failed to generate guest token")
	}

	return onboardDto.ScanResponse{
		Status: "success",
		Data: onboardDto.ScanData{
			Token:   token,
			GuestID: guestID,
			RoomID:  roomID.String(),
			BLEPairingInfo: onboardDto.BLEPairingInfo{
				MACAddress:    "00:1A:2B:3C:4D:5E", // ASSUMPTION: Mocked BLE MAC address
				HardwareColor: "red",
			},
		},
	}, nil
}

// PairStatus updates the BLE pairing status for a guest.
func (s *OnboardService) PairStatus(ctx context.Context, guestID string, req onboardDto.PairStatusRequest) (onboardDto.PairStatusResponse, error) {
	if guestID == "" {
		return onboardDto.PairStatusResponse{}, errors.New("guest_id not found in token")
	}

	paired := req.Status == "connected"
	if err := s.guestRepo.SetBLEPaired(ctx, guestID, paired); err != nil {
		return onboardDto.PairStatusResponse{}, errors.New("failed to update pairing status")
	}

	return onboardDto.PairStatusResponse{
		Status:  "success",
		Message: "System armed and ready for emergencies.",
	}, nil
}
