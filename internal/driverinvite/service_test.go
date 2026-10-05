package driverinvite

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/kishert-lab/taxi-platform/internal/domain"
)

type pushTokenRepository struct {
	savedUser     uuid.UUID
	savedToken    string
	savedPlatform string
	deletedUser   uuid.UUID
	deletedToken  string
	saveError     error
	deleteError   error
}

func (repository *pushTokenRepository) Transact(context.Context, func(Transaction) error) error {
	panic("unexpected transaction")
}
func (repository *pushTokenRepository) List(context.Context, uuid.UUID, bool) ([]domain.DriverParkInvite, error) {
	panic("unexpected list")
}
func (repository *pushTokenRepository) SavePushToken(_ context.Context, user uuid.UUID, token, platform string) error {
	repository.savedUser = user
	repository.savedToken = token
	repository.savedPlatform = platform
	return repository.saveError
}
func (repository *pushTokenRepository) DeletePushToken(_ context.Context, user uuid.UUID, token string) error {
	repository.deletedUser = user
	repository.deletedToken = token
	return repository.deleteError
}

func TestServicePushTokenValidation(t *testing.T) {
	repository := &pushTokenRepository{}
	service := NewService(repository, nil, 0, nil)
	user := uuid.New()

	if err := service.SavePushToken(context.Background(), user, "token", " ANDROID "); err != nil {
		t.Fatal(err)
	}
	if repository.savedUser != user || repository.savedToken != "token" || repository.savedPlatform != "android" {
		t.Fatalf("unexpected saved token: %+v", repository)
	}
	if err := service.DeletePushToken(context.Background(), user, "token"); err != nil {
		t.Fatal(err)
	}
	if repository.deletedUser != user || repository.deletedToken != "token" {
		t.Fatalf("unexpected deleted token: %+v", repository)
	}

	for _, test := range []struct {
		name     string
		token    string
		platform string
		want     error
	}{
		{name: "empty token", platform: "android", want: domain.ErrInvalidPushToken},
		{name: "oversized token", token: strings.Repeat("x", 4097), platform: "android", want: domain.ErrInvalidPushToken},
		{name: "unknown platform", token: "token", platform: "desktop", want: domain.ErrInvalidPushPlatform},
	} {
		t.Run(test.name, func(t *testing.T) {
			err := service.SavePushToken(context.Background(), user, test.token, test.platform)
			if !errors.Is(err, test.want) {
				t.Fatalf("SavePushToken() error = %v, want %v", err, test.want)
			}
		})
	}
	if err := service.DeletePushToken(context.Background(), user, ""); !errors.Is(err, domain.ErrInvalidPushToken) {
		t.Fatalf("DeletePushToken() error = %v", err)
	}
}

func TestServicePushTokenRepositoryErrors(t *testing.T) {
	saveError := errors.New("save failed")
	deleteError := errors.New("delete failed")
	repository := &pushTokenRepository{saveError: saveError, deleteError: deleteError}
	service := NewService(repository, nil, 0, nil)
	if err := service.SavePushToken(context.Background(), uuid.New(), "token", "ios"); !errors.Is(err, saveError) {
		t.Fatalf("SavePushToken() error = %v", err)
	}
	if err := service.DeletePushToken(context.Background(), uuid.New(), "token"); !errors.Is(err, deleteError) {
		t.Fatalf("DeletePushToken() error = %v", err)
	}
}
