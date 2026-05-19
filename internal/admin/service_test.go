package admin

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"

	"github.com/kishert-lab/taxi-platform/internal/domain"
)

func TestCreateTaxiParkRequiresDocumentAcceptance(t *testing.T) {
	service := NewService(&fakeRepository{}, fakePasswordHasher{})

	_, err := service.CreateTaxiPark(context.Background(), CreateTaxiParkCommand{
		Phone:  "+79990000000",
		Email:  "park@example.com",
		CityID: uuid.New(),
		Name:   "City Taxi",
	})
	if !errors.Is(err, ErrDocumentAcceptanceRequired) {
		t.Fatalf("expected document acceptance error, got %v", err)
	}
}

func TestCreateTaxiParkGeneratesPasswordAndNormalizesPhone(t *testing.T) {
	repository := &fakeRepository{}
	service := NewService(repository, fakePasswordHasher{})

	result, err := service.CreateTaxiPark(context.Background(), CreateTaxiParkCommand{
		Phone:                "+7 (999) 000-00-00",
		Email:                "PARK@example.com",
		CityID:               uuid.New(),
		Name:                 "City Taxi",
		AcceptDocuments:      true,
		PrivacyPolicyVersion: "1.0",
		TermsVersion:         "1.0",
	})
	if err != nil {
		t.Fatalf("create taxi park: %v", err)
	}
	if !result.PasswordGenerated || result.GeneratedPassword == "" {
		t.Fatal("expected generated password in result")
	}
	if repository.createdRecord.Phone != "+79990000000" {
		t.Fatalf("expected normalized phone, got %s", repository.createdRecord.Phone)
	}
	if repository.createdRecord.Email != "park@example.com" {
		t.Fatalf("expected normalized email, got %s", repository.createdRecord.Email)
	}
	if repository.createdRecord.PasswordHash == "" {
		t.Fatal("expected password hash to be persisted")
	}
}

func TestResetPasswordGeneratesPasswordAndRevokesTokens(t *testing.T) {
	repository := &fakeRepository{}
	service := NewService(repository, fakePasswordHasher{})

	result, err := service.ResetPassword(context.Background(), ResetPasswordCommand{
		Phone: "+79990000000",
		Role:  domain.UserRoleTaxiPark,
	})
	if err != nil {
		t.Fatalf("reset password: %v", err)
	}
	if !result.PasswordGenerated || result.GeneratedPassword == "" {
		t.Fatal("expected generated password in result")
	}
	if repository.resetRecord.Role != domain.UserRoleTaxiPark {
		t.Fatalf("expected taxi_park role, got %s", repository.resetRecord.Role)
	}
	if result.RevokedTokenCount != 2 {
		t.Fatalf("expected revoked token count, got %d", result.RevokedTokenCount)
	}
}

func TestListTaxiParkAccountsNormalizesLimitAndSearch(t *testing.T) {
	repository := &fakeRepository{}
	service := NewService(repository, fakePasswordHasher{})

	accounts, err := service.ListTaxiParkAccounts(context.Background(), ListTaxiParkAccountsCommand{
		Limit:          5000,
		Search:         "  city taxi  ",
		IncludeDeleted: true,
	})
	if err != nil {
		t.Fatalf("list taxi park accounts: %v", err)
	}
	if len(accounts) != 1 {
		t.Fatalf("expected one account, got %d", len(accounts))
	}
	if repository.listFilter.Limit != 1000 {
		t.Fatalf("expected capped limit 1000, got %d", repository.listFilter.Limit)
	}
	if repository.listFilter.Search != "city taxi" {
		t.Fatalf("expected trimmed search, got %q", repository.listFilter.Search)
	}
	if !repository.listFilter.IncludeDeleted {
		t.Fatal("expected include deleted flag to be forwarded")
	}
}

type fakePasswordHasher struct{}

func (fakePasswordHasher) HashPassword(password string) (string, error) {
	return "hash:" + password, nil
}

type fakeRepository struct {
	createdRecord CreateTaxiParkOwnerRecord
	resetRecord   ResetPasswordRecord
	listFilter    ListTaxiParkAccountsFilter
}

func (repository *fakeRepository) CreateTaxiParkOwner(_ context.Context, record CreateTaxiParkOwnerRecord) (CreateTaxiParkOwnerResult, error) {
	repository.createdRecord = record
	return CreateTaxiParkOwnerResult{
		UserID:     uuid.New(),
		TaxiParkID: uuid.New(),
		Phone:      record.Phone,
		Email:      record.Email,
	}, nil
}

func (repository *fakeRepository) ResetPasswordByPhone(_ context.Context, record ResetPasswordRecord) (ResetPasswordResult, error) {
	repository.resetRecord = record
	return ResetPasswordResult{
		UserID:            uuid.New(),
		Phone:             record.Phone,
		Role:              record.Role,
		RevokedTokenCount: 2,
	}, nil
}

func (repository *fakeRepository) ListTaxiParkAccounts(_ context.Context, filter ListTaxiParkAccountsFilter) ([]TaxiParkAccount, error) {
	repository.listFilter = filter
	return []TaxiParkAccount{
		{
			TaxiParkID:         uuid.New(),
			OwnerUserID:        uuid.New(),
			CityID:             uuid.New(),
			CityName:           "City",
			Name:               "City Taxi",
			ContactPhone:       "+79990000000",
			ContactEmail:       "park@example.com",
			OwnerPhone:         "+79990000000",
			VerificationStatus: "verified",
			IsOwnerActive:      true,
		},
	}, nil
}
