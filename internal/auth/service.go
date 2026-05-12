package auth

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/develoop/taxi-platform/internal/domain"
)

type RegistrationService struct {
	userRepository             UserRepository
	taxiParkRepository         TaxiParkRepository
	verificationCodeRepository VerificationCodeRepository
	userConsentEventRepository UserConsentEventRepository
	smsProvider                SMSProvider
	emailProvider              EmailProvider
	passwordHasher             PasswordHasher
	codeHasher                 CodeHasher
	codeGenerator              CodeGenerator
	logger                     *zap.Logger
	phoneCodeTTL               time.Duration
	emailCodeTTL               time.Duration
	maxCodeAttempts            int
}

type RegistrationServiceConfig struct {
	PhoneCodeTTL    time.Duration
	EmailCodeTTL    time.Duration
	MaxCodeAttempts int
}

type NewRegistrationServiceParams struct {
	UserRepository             UserRepository
	TaxiParkRepository         TaxiParkRepository
	VerificationCodeRepository VerificationCodeRepository
	UserConsentEventRepository UserConsentEventRepository
	SMSProvider                SMSProvider
	EmailProvider              EmailProvider
	PasswordHasher             PasswordHasher
	CodeHasher                 CodeHasher
	CodeGenerator              CodeGenerator
	Logger                     *zap.Logger
	Config                     RegistrationServiceConfig
}

func NewRegistrationService(params NewRegistrationServiceParams) *RegistrationService {
	return &RegistrationService{
		userRepository:             params.UserRepository,
		taxiParkRepository:         params.TaxiParkRepository,
		verificationCodeRepository: params.VerificationCodeRepository,
		userConsentEventRepository: params.UserConsentEventRepository,
		smsProvider:                params.SMSProvider,
		emailProvider:              params.EmailProvider,
		passwordHasher:             params.PasswordHasher,
		codeHasher:                 params.CodeHasher,
		codeGenerator:              params.CodeGenerator,
		logger:                     params.Logger,
		phoneCodeTTL:               params.Config.PhoneCodeTTL,
		emailCodeTTL:               params.Config.EmailCodeTTL,
		maxCodeAttempts:            params.Config.MaxCodeAttempts,
	}
}

type StartRegistrationCommand struct {
	Phone                string
	Email                string
	Password             string
	FirstName            string
	LastName             string
	RegistrationType     domain.RegistrationType
	CityID               uuid.UUID
	TaxiParkName         string
	TaxiParkLegalName    string
	TaxiParkTaxID        string
	PersonalDataConsent  bool
	TermsAccepted        bool
	PrivacyPolicyVersion string
	TermsVersion         string
	ConsentIP            string
	ConsentUserAgent     string
}

type StartRegistrationResult struct {
	UserID           uuid.UUID
	Role             domain.UserRole
	RegistrationType domain.RegistrationType
	PhoneMasked      string
	EmailMasked      string
}

type ConfirmPhoneCommand struct {
	Phone            string
	RegistrationType domain.RegistrationType
	Code             string
}

type ConfirmEmailCommand struct {
	Email string
	Code  string
}

func (service *RegistrationService) StartRegistration(ctx context.Context, command StartRegistrationCommand) (StartRegistrationResult, error) {
	if err := domain.ValidateRequiredRegistrationConsent(command.PersonalDataConsent, command.TermsAccepted, command.PrivacyPolicyVersion, command.TermsVersion); err != nil {
		return StartRegistrationResult{}, err
	}

	normalizedPhone, err := domain.NormalizePhone(command.Phone)
	if err != nil {
		return StartRegistrationResult{}, fmt.Errorf("normalize registration phone: %w", err)
	}

	normalizedEmail, err := domain.NormalizeEmail(command.Email)
	if err != nil {
		return StartRegistrationResult{}, fmt.Errorf("normalize registration email: %w", err)
	}

	if err := command.RegistrationType.Validate(); err != nil {
		return StartRegistrationResult{}, fmt.Errorf("validate registration type: %w", err)
	}

	role, err := domain.RoleFromRegistrationType(command.RegistrationType)
	if err != nil {
		return StartRegistrationResult{}, fmt.Errorf("resolve user role: %w", err)
	}

	passwordHash, err := service.passwordHasher.HashPassword(command.Password)
	if err != nil {
		return StartRegistrationResult{}, fmt.Errorf("hash registration password: %w", err)
	}

	now := time.Now().UTC()
	user := domain.User{
		Phone:                 normalizedPhone,
		Email:                 normalizedEmail,
		Role:                  role,
		RegistrationType:      command.RegistrationType,
		FirstName:             command.FirstName,
		LastName:              command.LastName,
		PasswordHash:          passwordHash,
		PersonalDataConsent:   true,
		PersonalDataConsentAt: &now,
		PrivacyPolicyVersion:  command.PrivacyPolicyVersion,
		TermsAccepted:         true,
		TermsAcceptedAt:       &now,
		TermsVersion:          command.TermsVersion,
		ConsentIP:             command.ConsentIP,
		ConsentUserAgent:      command.ConsentUserAgent,
		IsActive:              true,
	}

	createdUser, err := service.userRepository.CreateUser(ctx, user)
	if err != nil {
		return StartRegistrationResult{}, fmt.Errorf("create registration user: %w", err)
	}

	if command.RegistrationType == domain.RegistrationTypeTaxiPark {
		taxiPark := domain.TaxiPark{
			OwnerUserID:        createdUser.ID,
			CityID:             command.CityID,
			Name:               command.TaxiParkName,
			LegalName:          command.TaxiParkLegalName,
			TaxID:              command.TaxiParkTaxID,
			ContactPhone:       normalizedPhone,
			ContactEmail:       normalizedEmail,
			VerificationStatus: domain.VerificationStatusPending,
		}
		if _, err := service.taxiParkRepository.CreateTaxiPark(ctx, taxiPark); err != nil {
			return StartRegistrationResult{}, fmt.Errorf("create taxi park registration profile: %w", err)
		}
	}

	if err := service.createConsentAuditEvents(ctx, createdUser.ID, command, now); err != nil {
		return StartRegistrationResult{}, err
	}

	if err := service.sendPhoneConfirmation(ctx, createdUser.ID, normalizedPhone); err != nil {
		return StartRegistrationResult{}, err
	}
	if err := service.sendEmailConfirmation(ctx, createdUser.ID, normalizedEmail); err != nil {
		return StartRegistrationResult{}, err
	}

	service.logger.Info(
		"registration started",
		zap.String("user_id", createdUser.ID.String()),
		zap.String("role", string(role)),
		zap.String("registration_type", string(command.RegistrationType)),
	)

	return StartRegistrationResult{
		UserID:           createdUser.ID,
		Role:             role,
		RegistrationType: command.RegistrationType,
		PhoneMasked:      maskPhone(normalizedPhone),
		EmailMasked:      maskEmail(normalizedEmail),
	}, nil
}

func (service *RegistrationService) createConsentAuditEvents(ctx context.Context, userID uuid.UUID, command StartRegistrationCommand, acceptedAt time.Time) error {
	if service.userConsentEventRepository == nil {
		return fmt.Errorf("user consent event repository is required")
	}

	events := []domain.UserConsentEvent{
		{
			UserID:          userID,
			EventType:       domain.ConsentEventAccepted,
			DocumentType:    domain.ConsentDocumentPersonalData,
			DocumentVersion: command.PrivacyPolicyVersion,
			IP:              command.ConsentIP,
			UserAgent:       command.ConsentUserAgent,
			CreatedAt:       acceptedAt,
		},
		{
			UserID:          userID,
			EventType:       domain.ConsentEventAccepted,
			DocumentType:    domain.ConsentDocumentPrivacyPolicy,
			DocumentVersion: command.PrivacyPolicyVersion,
			IP:              command.ConsentIP,
			UserAgent:       command.ConsentUserAgent,
			CreatedAt:       acceptedAt,
		},
		{
			UserID:          userID,
			EventType:       domain.ConsentEventAccepted,
			DocumentType:    domain.ConsentDocumentTerms,
			DocumentVersion: command.TermsVersion,
			IP:              command.ConsentIP,
			UserAgent:       command.ConsentUserAgent,
			CreatedAt:       acceptedAt,
		},
	}

	for _, event := range events {
		if err := service.userConsentEventRepository.CreateUserConsentEvent(ctx, event); err != nil {
			return fmt.Errorf("create user consent audit event: %w", err)
		}
	}

	return nil
}

func (service *RegistrationService) ConfirmPhone(ctx context.Context, command ConfirmPhoneCommand) error {
	normalizedPhone, err := domain.NormalizePhone(command.Phone)
	if err != nil {
		return fmt.Errorf("normalize confirmation phone: %w", err)
	}

	if err := command.RegistrationType.Validate(); err != nil {
		return fmt.Errorf("validate confirmation registration type: %w", err)
	}

	role, err := domain.RoleFromRegistrationType(command.RegistrationType)
	if err != nil {
		return fmt.Errorf("resolve confirmation user role: %w", err)
	}

	user, err := service.userRepository.GetUserByPhoneAndRole(ctx, normalizedPhone, role)
	if err != nil {
		return fmt.Errorf("get user for phone confirmation: %w", err)
	}

	verificationCode, err := service.verificationCodeRepository.GetLatestActiveCode(
		ctx,
		normalizedPhone,
		domain.VerificationChannelSMS,
		domain.VerificationPurposeRegistration,
	)
	if err != nil {
		return fmt.Errorf("get active phone confirmation code: %w", err)
	}

	if time.Now().UTC().After(verificationCode.ExpiresAt) {
		return fmt.Errorf("phone confirmation code expired")
	}

	if err := service.codeHasher.CompareCodeAndHash(command.Code, verificationCode.CodeHash); err != nil {
		if incrementErr := service.verificationCodeRepository.IncrementAttempts(ctx, verificationCode.ID); incrementErr != nil {
			return fmt.Errorf("increment phone confirmation attempts after compare failure: %w", incrementErr)
		}
		return fmt.Errorf("compare phone confirmation code: %w", err)
	}

	now := time.Now().UTC()
	if err := service.verificationCodeRepository.ConsumeCode(ctx, verificationCode.ID, now); err != nil {
		return fmt.Errorf("consume phone confirmation code: %w", err)
	}
	if err := service.userRepository.MarkPhoneConfirmed(ctx, user.ID, now); err != nil {
		return fmt.Errorf("mark phone confirmed: %w", err)
	}

	service.logger.Info("phone confirmed", zap.String("user_id", user.ID.String()), zap.String("role", string(role)))
	return nil
}

func (service *RegistrationService) ConfirmEmail(ctx context.Context, command ConfirmEmailCommand) error {
	normalizedEmail, err := domain.NormalizeEmail(command.Email)
	if err != nil {
		return fmt.Errorf("normalize confirmation email: %w", err)
	}

	user, err := service.userRepository.GetUserByEmail(ctx, normalizedEmail)
	if err != nil {
		return fmt.Errorf("get user for email confirmation: %w", err)
	}

	verificationCode, err := service.verificationCodeRepository.GetLatestActiveCode(
		ctx,
		normalizedEmail,
		domain.VerificationChannelEmail,
		domain.VerificationPurposeEmailConfirm,
	)
	if err != nil {
		return fmt.Errorf("get active email confirmation code: %w", err)
	}

	if time.Now().UTC().After(verificationCode.ExpiresAt) {
		return fmt.Errorf("email confirmation code expired")
	}

	if err := service.codeHasher.CompareCodeAndHash(command.Code, verificationCode.CodeHash); err != nil {
		if incrementErr := service.verificationCodeRepository.IncrementAttempts(ctx, verificationCode.ID); incrementErr != nil {
			return fmt.Errorf("increment email confirmation attempts after compare failure: %w", incrementErr)
		}
		return fmt.Errorf("compare email confirmation code: %w", err)
	}

	now := time.Now().UTC()
	if err := service.verificationCodeRepository.ConsumeCode(ctx, verificationCode.ID, now); err != nil {
		return fmt.Errorf("consume email confirmation code: %w", err)
	}
	if err := service.userRepository.MarkEmailConfirmed(ctx, user.ID, now); err != nil {
		return fmt.Errorf("mark email confirmed: %w", err)
	}

	service.logger.Info("email confirmed", zap.String("user_id", user.ID.String()), zap.String("role", string(user.Role)))
	return nil
}

func (service *RegistrationService) sendPhoneConfirmation(ctx context.Context, userID uuid.UUID, phone string) error {
	code, err := service.codeGenerator.GenerateNumericCode(6)
	if err != nil {
		return fmt.Errorf("generate phone confirmation code: %w", err)
	}

	codeHash, err := service.codeHasher.HashCode(code)
	if err != nil {
		return fmt.Errorf("hash phone confirmation code: %w", err)
	}

	verificationCode := domain.VerificationCode{
		UserID:      userID,
		Target:      phone,
		Channel:     domain.VerificationChannelSMS,
		Purpose:     domain.VerificationPurposeRegistration,
		CodeHash:    codeHash,
		MaxAttempts: service.maxCodeAttempts,
		ExpiresAt:   time.Now().UTC().Add(service.phoneCodeTTL),
		LastSentAt:  time.Now().UTC(),
	}

	if _, err := service.verificationCodeRepository.CreateVerificationCode(ctx, verificationCode); err != nil {
		return fmt.Errorf("store phone confirmation code: %w", err)
	}
	if err := service.smsProvider.SendVerificationCode(ctx, phone, code); err != nil {
		return fmt.Errorf("send phone confirmation sms: %w", err)
	}

	return nil
}

func (service *RegistrationService) sendEmailConfirmation(ctx context.Context, userID uuid.UUID, email string) error {
	code, err := service.codeGenerator.GenerateNumericCode(6)
	if err != nil {
		return fmt.Errorf("generate email confirmation code: %w", err)
	}

	codeHash, err := service.codeHasher.HashCode(code)
	if err != nil {
		return fmt.Errorf("hash email confirmation code: %w", err)
	}

	verificationCode := domain.VerificationCode{
		UserID:      userID,
		Target:      email,
		Channel:     domain.VerificationChannelEmail,
		Purpose:     domain.VerificationPurposeEmailConfirm,
		CodeHash:    codeHash,
		MaxAttempts: service.maxCodeAttempts,
		ExpiresAt:   time.Now().UTC().Add(service.emailCodeTTL),
		LastSentAt:  time.Now().UTC(),
	}

	if _, err := service.verificationCodeRepository.CreateVerificationCode(ctx, verificationCode); err != nil {
		return fmt.Errorf("store email confirmation code: %w", err)
	}
	if err := service.emailProvider.SendEmailConfirmationCode(ctx, email, code); err != nil {
		return fmt.Errorf("send email confirmation code: %w", err)
	}

	return nil
}

func maskPhone(phone string) string {
	if len(phone) <= 5 {
		return phone
	}
	return phone[:2] + "*****" + phone[len(phone)-3:]
}

func maskEmail(email string) string {
	atIndex := -1
	for index, value := range email {
		if value == '@' {
			atIndex = index
			break
		}
	}
	if atIndex <= 1 {
		return "***" + email[atIndex:]
	}
	return email[:1] + "***" + email[atIndex:]
}
