package passenger

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/kishert-lab/taxi-platform/internal/domain"
	"github.com/kishert-lab/taxi-platform/internal/dto"
)

type ProfileService struct {
	repository Repository
}

func NewProfileService(repository Repository) *ProfileService {
	return &ProfileService{repository: repository}
}

func (service *ProfileService) GetMe(ctx context.Context, passengerID uuid.UUID) (dto.PassengerMeResponse, error) {
	passengerRecord, err := service.repository.GetByID(ctx, passengerID)
	if err != nil {
		return dto.PassengerMeResponse{}, fmt.Errorf("get passenger profile: %w", err)
	}

	return passengerDTO(passengerRecord), nil
}

func (service *ProfileService) UpdateMe(ctx context.Context, passengerID uuid.UUID, request dto.PassengerMePatchRequest) (dto.PassengerMeResponse, error) {
	passengerRecord, err := service.repository.GetByID(ctx, passengerID)
	if err != nil {
		return dto.PassengerMeResponse{}, fmt.Errorf("get passenger before update: %w", err)
	}

	name := passengerRecord.Name
	if request.Name != nil {
		name = domain.NormalizePassengerName(*request.Name)
	}

	email := passengerRecord.Email
	if request.Email != nil {
		if *request.Email == "" {
			email = ""
		} else {
			normalizedEmail, normalizeErr := domain.NormalizeEmail(*request.Email)
			if normalizeErr != nil {
				return dto.PassengerMeResponse{}, fmt.Errorf("normalize passenger email: %w", normalizeErr)
			}
			email = normalizedEmail
		}
	}

	avatarURL := passengerRecord.AvatarURL
	if request.AvatarURL != nil {
		avatarURL = *request.AvatarURL
	}

	passengerRecord, err = service.repository.UpdateProfile(ctx, passengerID, name, email, avatarURL)
	if err != nil {
		return dto.PassengerMeResponse{}, fmt.Errorf("update passenger profile: %w", err)
	}

	return passengerDTO(passengerRecord), nil
}

func passengerDTO(passengerRecord domain.Passenger) dto.PassengerMeResponse {
	return dto.PassengerMeResponse{
		ID:        passengerRecord.ID,
		Phone:     passengerRecord.Phone,
		Name:      passengerRecord.Name,
		Email:     passengerRecord.Email,
		AvatarURL: passengerRecord.AvatarURL,
	}
}
