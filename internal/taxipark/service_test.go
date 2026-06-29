package taxipark

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/kishert-lab/taxi-platform/internal/domain"
	"github.com/kishert-lab/taxi-platform/internal/dto"
)

func TestCreateDriverNormalizesPhoneAndGeneratesPassword(t *testing.T) {
	repository := &fakeRepository{}
	service := NewService(repository, fakePasswordHasher{})

	result, err := service.CreateDriver(context.Background(), uuid.New(), dto.TaxiParkCreateDriverRequest{
		Phone:     "+7 (999) 000-00-01",
		FirstName: " Ivan ",
		LastName:  " Petrov ",
	})
	if err != nil {
		t.Fatalf("create driver: %v", err)
	}

	if repository.createdRecord.Phone != "+79990000001" {
		t.Fatalf("expected normalized phone, got %q", repository.createdRecord.Phone)
	}
	if repository.createdRecord.FirstName != "Ivan" || repository.createdRecord.LastName != "Petrov" {
		t.Fatalf("expected trimmed names, got %q %q", repository.createdRecord.FirstName, repository.createdRecord.LastName)
	}
	if repository.createdRecord.PasswordHash == "" {
		t.Fatal("expected password hash")
	}
	if !result.PasswordGenerated || len(result.GeneratedPassword) != defaultDriverPasswordLength {
		t.Fatalf("expected generated password length %d, got generated=%t password=%q", defaultDriverPasswordLength, result.PasswordGenerated, result.GeneratedPassword)
	}
}

func TestCreateDriverUsesProvidedPassword(t *testing.T) {
	repository := &fakeRepository{}
	service := NewService(repository, fakePasswordHasher{})

	result, err := service.CreateDriver(context.Background(), uuid.New(), dto.TaxiParkCreateDriverRequest{
		Phone:    "+79990000001",
		Password: "manual-password",
	})
	if err != nil {
		t.Fatalf("create driver: %v", err)
	}

	if repository.createdRecord.PasswordHash != "hash:manual-password" {
		t.Fatalf("expected provided password hash, got %q", repository.createdRecord.PasswordHash)
	}
	if result.PasswordGenerated || result.GeneratedPassword != "" {
		t.Fatalf("expected no generated password, got generated=%t password=%q", result.PasswordGenerated, result.GeneratedPassword)
	}
}

func TestCreateDriverRejectsInvalidPhone(t *testing.T) {
	service := NewService(&fakeRepository{}, fakePasswordHasher{})

	_, err := service.CreateDriver(context.Background(), uuid.New(), dto.TaxiParkCreateDriverRequest{Phone: "999"})
	if err == nil || !strings.Contains(err.Error(), domain.ErrInvalidPhone.Error()) {
		t.Fatalf("expected invalid phone error, got %v", err)
	}
}

func TestCreateDispatcherNormalizesPhoneAndTrimsFields(t *testing.T) {
	repository := &fakeRepository{}
	service := NewService(repository, fakePasswordHasher{})

	result, err := service.CreateDispatcher(context.Background(), uuid.New(), dto.TaxiParkCreateDispatcherRequest{
		Phone:     "+7 (999) 000-00-02",
		Email:     " Dispatcher@Example.com ",
		Password:  "strong-password",
		FirstName: " Ivan ",
		LastName:  " Petrov ",
	})
	if err != nil {
		t.Fatalf("create dispatcher: %v", err)
	}

	if repository.createdDispatcherRecord.Phone != "+79990000002" {
		t.Fatalf("expected normalized dispatcher phone, got %q", repository.createdDispatcherRecord.Phone)
	}
	if repository.createdDispatcherRecord.Email != "dispatcher@example.com" {
		t.Fatalf("expected normalized dispatcher email, got %q", repository.createdDispatcherRecord.Email)
	}
	if repository.createdDispatcherRecord.FirstName != "Ivan" || repository.createdDispatcherRecord.LastName != "Petrov" {
		t.Fatalf("expected trimmed dispatcher names, got %q %q", repository.createdDispatcherRecord.FirstName, repository.createdDispatcherRecord.LastName)
	}
	if repository.createdDispatcherRecord.PasswordHash != "hash:strong-password" {
		t.Fatalf("expected dispatcher password hash, got %q", repository.createdDispatcherRecord.PasswordHash)
	}
	if result.Role != domain.UserRoleDispatcher {
		t.Fatalf("expected dispatcher role, got %s", result.Role)
	}
}

func TestCreateDispatcherRejectsShortPassword(t *testing.T) {
	service := NewService(&fakeRepository{}, fakePasswordHasher{})

	_, err := service.CreateDispatcher(context.Background(), uuid.New(), dto.TaxiParkCreateDispatcherRequest{
		Phone:    "+79990000002",
		Password: "short",
	})
	if err == nil || !strings.Contains(err.Error(), ErrInvalidDispatcherPassword.Error()) {
		t.Fatalf("expected invalid dispatcher password error, got %v", err)
	}
}

func TestUnblockDispatcherDelegatesToRepository(t *testing.T) {
	repository := &fakeRepository{}
	service := NewService(repository, fakePasswordHasher{})
	dispatcherID := uuid.New()

	if err := service.UnblockDispatcher(context.Background(), uuid.New(), dispatcherID); err != nil {
		t.Fatalf("unblock dispatcher: %v", err)
	}
	if repository.unblockDispatcherID != dispatcherID {
		t.Fatalf("expected dispatcher id %s, got %s", dispatcherID, repository.unblockDispatcherID)
	}
}

func TestCreateCarAcceptsRFC3339DatesAndOwnerFallback(t *testing.T) {
	repository := &fakeRepository{}
	service := NewService(repository, fakePasswordHasher{})

	_, err := service.CreateCar(context.Background(), uuid.New(), dto.TaxiParkCarRequest{
		Brand:                         " Lada ",
		Model:                         " Granta ",
		PlateNumber:                   " o777oo77 ",
		Color:                         " Белый ",
		OwnerOrLegalBasis:             " Lease agreement ",
		OSAGOExpiresAt:                "2026-10-16T00:00:00Z",
		DiagnosticCardExpiresAt:       "2026-10-16",
		RegionalRequirementsCompliant: true,
	})
	if err != nil {
		t.Fatalf("create car: %v", err)
	}
	if repository.createdCarRecord.OwnerDetails != "Lease agreement" {
		t.Fatalf("expected owner fallback, got %q", repository.createdCarRecord.OwnerDetails)
	}
	if repository.createdCarRecord.OSAGOExpiresAt == nil || repository.createdCarRecord.DiagnosticCardExpiresAt == nil {
		t.Fatal("expected parsed date fields")
	}
}

func TestCreateOrderAcceptsTaxiParkFrontendPayload(t *testing.T) {
	repository := &fakeRepository{}
	service := NewService(repository, fakePasswordHasher{})
	tariffID := uuid.MustParse("76f71789-9fb1-4945-8b5f-0f358b70aae3")

	_, err := service.CreateOrder(context.Background(), uuid.New(), dto.TaxiParkCreateOrderRequest{
		TariffID:       tariffID,
		PassengerPhone: "+79124966456",
		PickupAddress:  "ADR 1",
		PickupLocation: &dto.TaxiParkOrderCoordinatesRequest{
			Latitude:  56.80586,
			Longitude: 60.575867,
		},
		DestinationAddress: "ADR 2",
		DestinationLocation: &dto.TaxiParkOrderCoordinatesRequest{
			Latitude:  56.829535,
			Longitude: 60.631485,
		},
		PaymentType: domain.PaymentMethodCash,
	})
	if err != nil {
		t.Fatalf("create order from frontend payload: %v", err)
	}
	if repository.createdOrderRecord.TariffID != tariffID {
		t.Fatalf("expected tariff id %s, got %s", tariffID, repository.createdOrderRecord.TariffID)
	}
	if repository.createdOrderRecord.DestinationLocation == nil {
		t.Fatal("expected destination location")
	}
}

func TestCreateScheduledOrderCalculatesTimingAndDoesNotRejectValidFutureTime(t *testing.T) {
	repository := &fakeRepository{}
	service := NewService(repository, fakePasswordHasher{})
	scheduledAt := time.Now().UTC().Add(40 * time.Minute)

	_, err := service.CreateScheduledOrder(context.Background(), uuid.New(), dto.TaxiParkCreateScheduledOrderRequest{
		TariffID:      uuid.New(),
		PickupAddress: "ADR 1",
		PickupLocation: &dto.TaxiParkOrderCoordinatesRequest{
			Latitude:  56.80586,
			Longitude: 60.575867,
		},
		DestinationAddress: "ADR 2",
		ScheduledAt:        scheduledAt,
		Timezone:           "Asia/Yekaterinburg",
	})
	if err != nil {
		t.Fatalf("create scheduled order: %v", err)
	}
	if repository.createdScheduledOrderRecord.ScheduledAt.IsZero() {
		t.Fatal("expected scheduled_at to be forwarded")
	}
}

type fakeRepository struct {
	createdRecord               CreateDriverRecord
	createdDispatcherRecord     CreateDispatcherRecord
	createdCarRecord            CarRecord
	createdOrderRecord          CreateOrderRecord
	createdScheduledOrderRecord CreateScheduledOrderRecord
	unblockDispatcherID         uuid.UUID
}

func (repository *fakeRepository) GetSettingsByOwnerUserID(context.Context, uuid.UUID) (domain.TaxiParkSettings, error) {
	return domain.TaxiParkSettings{
		ScheduledOrdersEnabled:           true,
		ScheduledMinBeforeMinutes:        15,
		ScheduledActivationBeforeMinutes: 20,
		CityTimezone:                     "Asia/Yekaterinburg",
	}, nil
}

func (repository *fakeRepository) UpdateSettingsByOwnerUserID(context.Context, uuid.UUID, dto.TaxiParkSettingsPatchRequest) (domain.TaxiParkSettings, error) {
	return domain.TaxiParkSettings{}, nil
}

func (repository *fakeRepository) ListTariffsByOwnerUserID(context.Context, uuid.UUID) ([]domain.TaxiParkTariff, error) {
	return nil, nil
}

func (repository *fakeRepository) CreateTariffByOwnerUserID(context.Context, uuid.UUID, dto.TaxiParkTariffRequest) (domain.TaxiParkTariff, error) {
	return domain.TaxiParkTariff{}, nil
}

func (repository *fakeRepository) UpdateTariffByOwnerUserID(context.Context, uuid.UUID, uuid.UUID, dto.TaxiParkTariffPatchRequest) (domain.TaxiParkTariff, error) {
	return domain.TaxiParkTariff{}, nil
}

func (repository *fakeRepository) CreateOrderByOwnerUserID(_ context.Context, _ uuid.UUID, record CreateOrderRecord) (domain.Order, error) {
	repository.createdOrderRecord = record
	return domain.Order{}, nil
}

func (repository *fakeRepository) GetOrderByActorUserID(context.Context, uuid.UUID, uuid.UUID) (domain.Order, error) {
	return domain.Order{}, nil
}

func (repository *fakeRepository) CreateScheduledOrderByActorUserID(_ context.Context, _ uuid.UUID, record CreateScheduledOrderRecord) (ScheduledOrder, error) {
	repository.createdScheduledOrderRecord = record
	status := domain.ScheduledOrderStatusConfirmed
	return ScheduledOrder{
		Order: domain.Order{
			ID:              uuid.New(),
			OrderType:       domain.OrderTypeScheduled,
			Status:          domain.OrderStatusCreated,
			ScheduledStatus: &status,
			ScheduledAt:     &record.ScheduledAt,
			ActivationAt:    ptrTime(record.ScheduledAt.Add(-20 * time.Minute)),
		},
	}, nil
}

func (repository *fakeRepository) ListScheduledOrdersByActorUserID(context.Context, uuid.UUID) ([]ScheduledOrder, error) {
	return nil, nil
}

func (repository *fakeRepository) GetScheduledOrderByActorUserID(context.Context, uuid.UUID, uuid.UUID) (ScheduledOrder, error) {
	return ScheduledOrder{}, nil
}

func (repository *fakeRepository) UpdateScheduledOrderByActorUserID(context.Context, uuid.UUID, uuid.UUID, UpdateScheduledOrderRecord) (ScheduledOrder, error) {
	return ScheduledOrder{}, nil
}

func (repository *fakeRepository) CancelScheduledOrderByActorUserID(context.Context, uuid.UUID, uuid.UUID, string) (ScheduledOrder, error) {
	return ScheduledOrder{}, nil
}

func (repository *fakeRepository) AssignScheduledOrderDriverByActorUserID(context.Context, uuid.UUID, uuid.UUID, uuid.UUID) (ScheduledOrder, error) {
	return ScheduledOrder{}, nil
}

func (repository *fakeRepository) UpdateOrderByActorUserID(context.Context, uuid.UUID, uuid.UUID, UpdateOrderRecord) (domain.Order, error) {
	return domain.Order{}, nil
}

func (repository *fakeRepository) CancelOrderByActorUserID(context.Context, uuid.UUID, uuid.UUID, string) (domain.Order, error) {
	return domain.Order{}, nil
}

func (repository *fakeRepository) CompleteOrderByActorUserID(context.Context, uuid.UUID, uuid.UUID, int64) (domain.Order, error) {
	return domain.Order{}, nil
}

func (repository *fakeRepository) ListDispatchersByOwnerUserID(context.Context, uuid.UUID) ([]Dispatcher, error) {
	return nil, nil
}

func (repository *fakeRepository) CreateDispatcherByOwnerUserID(_ context.Context, _ uuid.UUID, record CreateDispatcherRecord) (Dispatcher, error) {
	repository.createdDispatcherRecord = record
	return Dispatcher{
		DispatcherID: uuid.New(),
		UserID:       uuid.New(),
		TaxiParkID:   uuid.New(),
		Phone:        record.Phone,
		Email:        record.Email,
		FirstName:    record.FirstName,
		LastName:     record.LastName,
		Role:         domain.UserRoleDispatcher,
		IsActive:     true,
		CreatedAt:    time.Now().UTC(),
		UpdatedAt:    time.Now().UTC(),
	}, nil
}

func (repository *fakeRepository) UpdateDispatcherByOwnerUserID(context.Context, uuid.UUID, uuid.UUID, UpdateDispatcherRecord) (Dispatcher, error) {
	return Dispatcher{}, nil
}

func (repository *fakeRepository) BlockDispatcherByOwnerUserID(context.Context, uuid.UUID, uuid.UUID) error {
	return nil
}

func (repository *fakeRepository) UnblockDispatcherByOwnerUserID(_ context.Context, _ uuid.UUID, dispatcherID uuid.UUID) error {
	repository.unblockDispatcherID = dispatcherID
	return nil
}

func (repository *fakeRepository) CreateDriverByOwnerUserID(_ context.Context, _ uuid.UUID, record CreateDriverRecord) (CreateDriverResult, error) {
	repository.createdRecord = record
	return CreateDriverResult{
		UserID:     uuid.New(),
		DriverID:   uuid.New(),
		TaxiParkID: uuid.New(),
		Phone:      record.Phone,
		Status:     domain.DriverStatusOffline,
		Rating:     5,
	}, nil
}

func (repository *fakeRepository) ListDriverLocationsByOwnerUserID(context.Context, uuid.UUID, time.Duration) ([]DriverLocation, error) {
	return nil, nil
}

func (repository *fakeRepository) UpdateDriverByOwnerUserID(context.Context, uuid.UUID, uuid.UUID, UpdateDriverRecord) (CreateDriverResult, error) {
	return CreateDriverResult{}, nil
}

func (repository *fakeRepository) UpdateDriverPasswordByOwnerUserID(context.Context, uuid.UUID, uuid.UUID, string) error {
	return nil
}

func (repository *fakeRepository) BlockDriverByOwnerUserID(context.Context, uuid.UUID, uuid.UUID, string) error {
	return nil
}

func (repository *fakeRepository) ArchiveDriverByOwnerUserID(context.Context, uuid.UUID, uuid.UUID) error {
	return nil
}

func (repository *fakeRepository) UnblockDriverByOwnerUserID(context.Context, uuid.UUID, uuid.UUID) error {
	return nil
}

func (repository *fakeRepository) ArchiveCarByOwnerUserID(context.Context, uuid.UUID, uuid.UUID) error {
	return nil
}

func (repository *fakeRepository) AttachCarToDriverByOwnerUserID(context.Context, uuid.UUID, uuid.UUID, uuid.UUID) error {
	return nil
}

func (repository *fakeRepository) AssignCarToDriverByOwnerUserID(context.Context, uuid.UUID, uuid.UUID, uuid.UUID) error {
	return nil
}

func (repository *fakeRepository) DetachCarFromDriverByOwnerUserID(context.Context, uuid.UUID, uuid.UUID, uuid.UUID) error {
	return nil
}

func (repository *fakeRepository) ListDriverDocumentsByOwnerUserID(context.Context, uuid.UUID, uuid.UUID) ([]domain.TaxiParkDocument, error) {
	return nil, nil
}

func (repository *fakeRepository) ListCarDocumentsByOwnerUserID(context.Context, uuid.UUID, uuid.UUID) ([]domain.TaxiParkDocument, error) {
	return nil, nil
}

func (repository *fakeRepository) ListCarsByOwnerUserID(context.Context, uuid.UUID) ([]domain.Car, error) {
	return nil, nil
}

func (repository *fakeRepository) ListCarsByDriverAndOwnerUserID(context.Context, uuid.UUID, uuid.UUID) ([]domain.Car, error) {
	return nil, nil
}

func (repository *fakeRepository) CreateCarByOwnerUserID(_ context.Context, _ uuid.UUID, record CarRecord) (domain.Car, error) {
	repository.createdCarRecord = record
	return domain.Car{ID: uuid.New(), Brand: record.Brand, Model: record.Model}, nil
}

func (repository *fakeRepository) UpdateCarByOwnerUserID(context.Context, uuid.UUID, uuid.UUID, CarPatchRecord) (domain.Car, error) {
	return domain.Car{}, nil
}

type fakePasswordHasher struct{}

func (fakePasswordHasher) HashPassword(password string) (string, error) {
	return "hash:" + password, nil
}

func ptrTime(value time.Time) *time.Time {
	return &value
}
