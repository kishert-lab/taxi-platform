package passenger

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"go.uber.org/zap"

	"github.com/kishert-lab/taxi-platform/internal/domain"
	"github.com/kishert-lab/taxi-platform/internal/dto"
)

func TestRequestCodeInvalidatesPreviousCode(t *testing.T) {
	repository := &fakeAuthCodeRepository{}
	service := newTestAuthService(&fakePassengerRepository{}, repository, &fakePassengerRefreshTokenRepository{})

	result, err := service.RequestCode(context.Background(), dto.PassengerAuthRequestCodeRequest{Phone: "8 (999) 123-45-67"})
	if err != nil {
		t.Fatalf("RequestCode returned error: %v", err)
	}
	if !result.Success {
		t.Fatal("expected success response")
	}
	if repository.invalidatedPhone != "+79991234567" {
		t.Fatalf("expected invalidated phone to be normalized, got %s", repository.invalidatedPhone)
	}
	if repository.created.Phone != "+79991234567" {
		t.Fatalf("expected created code phone to be normalized, got %s", repository.created.Phone)
	}
	if repository.created.CodeHash == "" {
		t.Fatal("expected hashed code to be stored")
	}
}

func TestConfirmCodeCreatesPassengerAndIssuesTokens(t *testing.T) {
	passengerRepository := &fakePassengerRepository{getByPhoneErr: pgx.ErrNoRows}
	authCodeRepository := &fakeAuthCodeRepository{
		latest: domain.PassengerAuthCode{
			ID:          uuid.New(),
			Phone:       "+79991234567",
			CodeHash:    "hash:1234",
			MaxAttempts: 5,
			ExpiresAt:   time.Now().UTC().Add(time.Minute),
		},
	}
	refreshTokens := &fakePassengerRefreshTokenRepository{}
	service := newTestAuthService(passengerRepository, authCodeRepository, refreshTokens)

	result, err := service.ConfirmCode(context.Background(), dto.PassengerAuthConfirmCodeRequest{
		Phone: "89991234567",
		Code:  "1234",
		Name:  "Сергей",
	})
	if err != nil {
		t.Fatalf("ConfirmCode returned error: %v", err)
	}
	if result.Passenger.ID == uuid.Nil {
		t.Fatal("expected passenger id")
	}
	if result.Passenger.Phone != "+79991234567" {
		t.Fatalf("unexpected passenger phone: %s", result.Passenger.Phone)
	}
	if passengerRepository.created.Name != "Сергей" {
		t.Fatalf("expected passenger name to be stored, got %s", passengerRepository.created.Name)
	}
	if authCodeRepository.markedUsed == uuid.Nil {
		t.Fatal("expected code to be marked used")
	}
	if refreshTokens.storedPassengerID == uuid.Nil {
		t.Fatal("expected refresh token to be stored")
	}
}

func TestConfirmCodeInvalidIncrementsAttempts(t *testing.T) {
	authCodeRepository := &fakeAuthCodeRepository{
		latest: domain.PassengerAuthCode{
			ID:          uuid.New(),
			Phone:       "+79991234567",
			CodeHash:    "hash:1234",
			MaxAttempts: 5,
			ExpiresAt:   time.Now().UTC().Add(time.Minute),
		},
	}
	service := newTestAuthService(&fakePassengerRepository{}, authCodeRepository, &fakePassengerRefreshTokenRepository{})

	_, err := service.ConfirmCode(context.Background(), dto.PassengerAuthConfirmCodeRequest{
		Phone: "+79991234567",
		Code:  "9999",
	})
	if !errors.Is(err, ErrInvalidCode) {
		t.Fatalf("expected ErrInvalidCode, got %v", err)
	}
	if authCodeRepository.incrementedID != authCodeRepository.latest.ID {
		t.Fatal("expected attempts to be incremented")
	}
}

func TestConfirmCodeRejectsBlockedPassenger(t *testing.T) {
	passengerRepository := &fakePassengerRepository{
		getByPhone: domain.Passenger{
			ID:       uuid.New(),
			Phone:    "+79991234567",
			IsActive: false,
		},
	}
	authCodeRepository := &fakeAuthCodeRepository{
		latest: domain.PassengerAuthCode{
			ID:          uuid.New(),
			Phone:       "+79991234567",
			CodeHash:    "hash:1234",
			MaxAttempts: 5,
			ExpiresAt:   time.Now().UTC().Add(time.Minute),
		},
	}
	service := newTestAuthService(passengerRepository, authCodeRepository, &fakePassengerRefreshTokenRepository{})

	_, err := service.ConfirmCode(context.Background(), dto.PassengerAuthConfirmCodeRequest{
		Phone: "+79991234567",
		Code:  "1234",
	})
	if !errors.Is(err, ErrPassengerBlocked) {
		t.Fatalf("expected ErrPassengerBlocked, got %v", err)
	}
}

func newTestAuthService(passengerRepository Repository, authCodeRepository AuthCodeRepository, refreshTokenRepository RefreshTokenRepository) *AuthService {
	return NewAuthService(AuthServiceParams{
		Repository:             passengerRepository,
		AuthCodeRepository:     authCodeRepository,
		RefreshTokenRepository: refreshTokenRepository,
		SMSService:             &fakeSMSService{},
		CodeGenerator:          fakeCodeGenerator("1234"),
		CodeHasher:             fakeCodeHasher{},
		TokenManager: NewTokenManager(TokenManagerConfig{
			AccessSecret:  "test-passenger-access-secret-32-characters-min",
			RefreshSecret: "test-passenger-refresh-secret-32-characters-min",
			Issuer:        "taxi-platform-test",
			AccessTTL:     time.Hour,
			RefreshTTL:    time.Hour,
		}),
		Logger:          zap.NewNop(),
		CodeLength:      4,
		CodeTTL:         5 * time.Minute,
		MaxCodeAttempts: 5,
	})
}

type fakePassengerRepository struct {
	getByPhone       domain.Passenger
	getByPhoneErr    error
	getByID          domain.Passenger
	getByIDErr       error
	created          domain.Passenger
	createdResult    domain.Passenger
	markResult       domain.Passenger
	updateProfileRes domain.Passenger
}

func (repository *fakePassengerRepository) Create(_ context.Context, passenger domain.Passenger) (domain.Passenger, error) {
	repository.created = passenger
	if repository.createdResult.ID == uuid.Nil {
		repository.createdResult = passenger
		repository.createdResult.ID = uuid.New()
	}
	return repository.createdResult, nil
}

func (repository *fakePassengerRepository) GetByID(_ context.Context, _ uuid.UUID) (domain.Passenger, error) {
	if repository.getByIDErr != nil {
		return domain.Passenger{}, repository.getByIDErr
	}
	return repository.getByID, nil
}

func (repository *fakePassengerRepository) GetByPhone(_ context.Context, _ string) (domain.Passenger, error) {
	if repository.getByPhoneErr != nil {
		return domain.Passenger{}, repository.getByPhoneErr
	}
	return repository.getByPhone, nil
}

func (repository *fakePassengerRepository) UpdateProfile(_ context.Context, _ uuid.UUID, name string, email string, avatarURL string) (domain.Passenger, error) {
	repository.updateProfileRes.Name = name
	repository.updateProfileRes.Email = email
	repository.updateProfileRes.AvatarURL = avatarURL
	return repository.updateProfileRes, nil
}

func (repository *fakePassengerRepository) MarkAuthenticated(_ context.Context, passengerID uuid.UUID, phoneVerifiedAt *time.Time, lastLoginAt time.Time) (domain.Passenger, error) {
	if repository.markResult.ID == uuid.Nil {
		repository.markResult = repository.createdResult
		if repository.markResult.ID == uuid.Nil {
			repository.markResult = repository.getByPhone
		}
	}
	repository.markResult.ID = passengerID
	repository.markResult.PhoneVerifiedAt = phoneVerifiedAt
	repository.markResult.LastLoginAt = &lastLoginAt
	repository.markResult.IsActive = true
	return repository.markResult, nil
}

type fakeAuthCodeRepository struct {
	latest           domain.PassengerAuthCode
	created          domain.PassengerAuthCode
	invalidatedPhone string
	incrementedID    uuid.UUID
	markedUsed       uuid.UUID
}

func (repository *fakeAuthCodeRepository) Create(_ context.Context, code domain.PassengerAuthCode) (domain.PassengerAuthCode, error) {
	repository.created = code
	code.ID = uuid.New()
	return code, nil
}

func (repository *fakeAuthCodeRepository) InvalidateActiveByPhone(_ context.Context, phone string, _ time.Time) error {
	repository.invalidatedPhone = phone
	return nil
}

func (repository *fakeAuthCodeRepository) GetLatestActiveByPhone(_ context.Context, _ string) (domain.PassengerAuthCode, error) {
	return repository.latest, nil
}

func (repository *fakeAuthCodeRepository) IncrementAttempts(_ context.Context, codeID uuid.UUID) error {
	repository.incrementedID = codeID
	return nil
}

func (repository *fakeAuthCodeRepository) MarkUsed(_ context.Context, codeID uuid.UUID, _ time.Time) error {
	repository.markedUsed = codeID
	return nil
}

type fakePassengerRefreshTokenRepository struct {
	storedPassengerID uuid.UUID
}

func (repository *fakePassengerRefreshTokenRepository) Store(_ context.Context, passengerID uuid.UUID, _ string, _ time.Time) error {
	repository.storedPassengerID = passengerID
	return nil
}

func (repository *fakePassengerRefreshTokenRepository) Rotate(context.Context, string, uuid.UUID, string, time.Time) error {
	return nil
}

func (repository *fakePassengerRefreshTokenRepository) Revoke(context.Context, string) error {
	return nil
}

type fakeSMSService struct{}

func (service *fakeSMSService) SendCode(context.Context, string, string) error {
	return nil
}

type fakeCodeGenerator string

func (generator fakeCodeGenerator) GenerateNumericCode(int) (string, error) {
	return string(generator), nil
}

type fakeCodeHasher struct{}

func (hasher fakeCodeHasher) HashCode(code string) (string, error) {
	return "hash:" + code, nil
}

func (hasher fakeCodeHasher) CompareCodeAndHash(code string, hash string) error {
	if hash != "hash:"+code {
		return errors.New("invalid code")
	}
	return nil
}
