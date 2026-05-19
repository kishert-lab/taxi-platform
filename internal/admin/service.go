package admin

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"math/big"
	"regexp"
	"strings"

	"github.com/kishert-lab/taxi-platform/internal/domain"
)

const (
	defaultPasswordLength   = 18
	defaultDocumentVersion  = "1.0"
	defaultConsentIP        = "console"
	defaultConsentUserAgent = "taxi-admin-cli"
)

var (
	ErrDocumentAcceptanceRequired = errors.New("document acceptance confirmation is required")
	ErrInvalidTaxiParkName        = errors.New("invalid taxi park name")
	ErrInvalidCommissionPercent   = errors.New("invalid commission percent")

	commissionPercentPattern = regexp.MustCompile(`^(100(\.00?)?|[0-9]{1,2}(\.[0-9]{1,2})?)$`)
)

type Service struct {
	repository     Repository
	passwordHasher PasswordHasher
}

func NewService(repository Repository, passwordHasher PasswordHasher) *Service {
	return &Service{repository: repository, passwordHasher: passwordHasher}
}

func (service *Service) CreateTaxiPark(ctx context.Context, command CreateTaxiParkCommand) (CreateTaxiParkResult, error) {
	if !command.AcceptDocuments {
		return CreateTaxiParkResult{}, ErrDocumentAcceptanceRequired
	}
	if strings.TrimSpace(command.Name) == "" {
		return CreateTaxiParkResult{}, ErrInvalidTaxiParkName
	}

	phone, err := domain.NormalizePhone(command.Phone)
	if err != nil {
		return CreateTaxiParkResult{}, fmt.Errorf("normalize taxi park owner phone: %w", err)
	}
	email, err := domain.NormalizeEmail(command.Email)
	if err != nil {
		return CreateTaxiParkResult{}, fmt.Errorf("normalize taxi park owner email: %w", err)
	}

	password := strings.TrimSpace(command.Password)
	passwordGenerated := false
	if password == "" {
		password, err = GeneratePassword(defaultPasswordLength)
		if err != nil {
			return CreateTaxiParkResult{}, err
		}
		passwordGenerated = true
	}

	passwordHash, err := service.passwordHasher.HashPassword(password)
	if err != nil {
		return CreateTaxiParkResult{}, fmt.Errorf("hash taxi park owner password: %w", err)
	}

	record := CreateTaxiParkOwnerRecord{
		Phone:                phone,
		Email:                email,
		PasswordHash:         passwordHash,
		FirstName:            strings.TrimSpace(command.FirstName),
		LastName:             strings.TrimSpace(command.LastName),
		CityID:               command.CityID,
		Name:                 strings.TrimSpace(command.Name),
		LegalName:            strings.TrimSpace(command.LegalName),
		TaxID:                strings.TrimSpace(command.TaxID),
		CommissionPercent:    normalizedCommissionPercent(command.CommissionPercent),
		Verified:             command.Verified,
		PrivacyPolicyVersion: normalizedDocumentVersion(command.PrivacyPolicyVersion),
		TermsVersion:         normalizedDocumentVersion(command.TermsVersion),
		ConsentIP:            normalizedFallback(command.ConsentIP, defaultConsentIP),
		ConsentUserAgent:     normalizedFallback(command.ConsentUserAgent, defaultConsentUserAgent),
	}
	if record.CommissionPercent != nil && !commissionPercentPattern.MatchString(*record.CommissionPercent) {
		return CreateTaxiParkResult{}, ErrInvalidCommissionPercent
	}

	created, err := service.repository.CreateTaxiParkOwner(ctx, record)
	if err != nil {
		return CreateTaxiParkResult{}, fmt.Errorf("create taxi park owner: %w", err)
	}

	result := CreateTaxiParkResult{
		CreateTaxiParkOwnerResult: created,
		PasswordGenerated:         passwordGenerated,
	}
	if passwordGenerated {
		result.GeneratedPassword = password
	}

	return result, nil
}

func (service *Service) ResetPassword(ctx context.Context, command ResetPasswordCommand) (ResetPasswordCommandResult, error) {
	if err := command.Role.Validate(); err != nil {
		return ResetPasswordCommandResult{}, fmt.Errorf("validate user role: %w", err)
	}

	phone, err := domain.NormalizePhone(command.Phone)
	if err != nil {
		return ResetPasswordCommandResult{}, fmt.Errorf("normalize password reset phone: %w", err)
	}

	password := strings.TrimSpace(command.Password)
	passwordGenerated := false
	if password == "" {
		password, err = GeneratePassword(defaultPasswordLength)
		if err != nil {
			return ResetPasswordCommandResult{}, err
		}
		passwordGenerated = true
	}

	passwordHash, err := service.passwordHasher.HashPassword(password)
	if err != nil {
		return ResetPasswordCommandResult{}, fmt.Errorf("hash reset password: %w", err)
	}

	updated, err := service.repository.ResetPasswordByPhone(ctx, ResetPasswordRecord{
		Phone:        phone,
		Role:         command.Role,
		PasswordHash: passwordHash,
	})
	if err != nil {
		return ResetPasswordCommandResult{}, fmt.Errorf("reset password by phone: %w", err)
	}

	result := ResetPasswordCommandResult{
		ResetPasswordResult: updated,
		PasswordGenerated:   passwordGenerated,
	}
	if passwordGenerated {
		result.GeneratedPassword = password
	}

	return result, nil
}

func (service *Service) ListTaxiParkAccounts(ctx context.Context, command ListTaxiParkAccountsCommand) ([]TaxiParkAccount, error) {
	limit := command.Limit
	if limit <= 0 {
		limit = 100
	}
	if limit > 1000 {
		limit = 1000
	}

	accounts, err := service.repository.ListTaxiParkAccounts(ctx, ListTaxiParkAccountsFilter{
		Limit:          limit,
		Search:         strings.TrimSpace(command.Search),
		IncludeDeleted: command.IncludeDeleted,
	})
	if err != nil {
		return nil, fmt.Errorf("list taxi park accounts: %w", err)
	}

	return accounts, nil
}

func GeneratePassword(length int) (string, error) {
	if length < 12 {
		length = 12
	}

	const alphabet = "ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz23456789"
	password := make([]byte, length)
	for index := range password {
		value, err := rand.Int(rand.Reader, big.NewInt(int64(len(alphabet))))
		if err != nil {
			return "", fmt.Errorf("generate secure password: %w", err)
		}
		password[index] = alphabet[value.Int64()]
	}

	return string(password), nil
}

func normalizedCommissionPercent(value string) *string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return nil
	}
	return &trimmed
}

func normalizedDocumentVersion(value string) string {
	return normalizedFallback(value, defaultDocumentVersion)
}

func normalizedFallback(value string, fallback string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return fallback
	}
	return trimmed
}
