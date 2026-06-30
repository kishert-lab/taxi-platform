package passenger

import (
	"context"
	"testing"

	"github.com/google/uuid"

	"github.com/kishert-lab/taxi-platform/internal/domain"
	"github.com/kishert-lab/taxi-platform/internal/dto"
)

func TestProfileServiceGetMe(t *testing.T) {
	passengerID := uuid.New()
	service := NewProfileService(&fakePassengerRepository{
		getByID: domain.Passenger{
			ID:        passengerID,
			Phone:     "+79991234567",
			Name:      "Сергей",
			Email:     "test@example.com",
			AvatarURL: "https://cdn.example.com/a.jpg",
			IsActive:  true,
		},
	})

	result, err := service.GetMe(context.Background(), passengerID)
	if err != nil {
		t.Fatalf("GetMe returned error: %v", err)
	}
	if result.ID != passengerID || result.Phone != "+79991234567" {
		t.Fatalf("unexpected profile response: %#v", result)
	}
}

func TestProfileServiceUpdateMe(t *testing.T) {
	passengerID := uuid.New()
	repository := &fakePassengerRepository{
		getByID: domain.Passenger{
			ID:        passengerID,
			Phone:     "+79991234567",
			Name:      "Старое имя",
			Email:     "old@example.com",
			AvatarURL: "https://cdn.example.com/old.jpg",
			IsActive:  true,
		},
		updateProfileRes: domain.Passenger{
			ID:        passengerID,
			Phone:     "+79991234567",
			Name:      "Новое имя",
			Email:     "new@example.com",
			AvatarURL: "https://cdn.example.com/new.jpg",
			IsActive:  true,
		},
	}
	service := NewProfileService(repository)
	name := "Новое имя"
	email := "new@example.com"
	avatarURL := "https://cdn.example.com/new.jpg"

	result, err := service.UpdateMe(context.Background(), passengerID, dto.PassengerMePatchRequest{
		Name:      &name,
		Email:     &email,
		AvatarURL: &avatarURL,
	})
	if err != nil {
		t.Fatalf("UpdateMe returned error: %v", err)
	}
	if result.Name != "Новое имя" || result.Email != "new@example.com" {
		t.Fatalf("unexpected updated profile: %#v", result)
	}
}
