package domain

import (
	"errors"
	"net/mail"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
)

type UserRole string

const (
	UserRolePassenger  UserRole = "passenger"
	UserRoleDriver     UserRole = "driver"
	UserRoleTaxiPark   UserRole = "taxi_park"
	UserRoleAdmin      UserRole = "admin"
	UserRoleDispatcher UserRole = "dispatcher"
)

type RegistrationType string

const (
	RegistrationTypePassenger RegistrationType = "passenger"
	RegistrationTypeDriver    RegistrationType = "driver"
	RegistrationTypeTaxiPark  RegistrationType = "taxi_park"
)

type VerificationChannel string

const (
	VerificationChannelSMS   VerificationChannel = "sms"
	VerificationChannelEmail VerificationChannel = "email"
)

type VerificationPurpose string

const (
	VerificationPurposeRegistration  VerificationPurpose = "registration"
	VerificationPurposeLogin         VerificationPurpose = "login"
	VerificationPurposeEmailConfirm  VerificationPurpose = "email_confirm"
	VerificationPurposePhoneChange   VerificationPurpose = "phone_change"
	VerificationPurposePasswordReset VerificationPurpose = "password_reset"
)

type VerificationStatus string

const (
	VerificationStatusPending  VerificationStatus = "pending"
	VerificationStatusVerified VerificationStatus = "verified"
	VerificationStatusExpired  VerificationStatus = "expired"
	VerificationStatusBlocked  VerificationStatus = "blocked"
)

type User struct {
	ID                    uuid.UUID
	Phone                 string
	Email                 string
	Role                  UserRole
	RegistrationType      RegistrationType
	FirstName             string
	LastName              string
	ProfilePhotoURL       string
	Rating                float64
	RatingsCount          int
	PasswordHash          string
	MustChangePassword    bool
	IsPhoneConfirmed      bool
	IsEmailConfirmed      bool
	PersonalDataConsent   bool
	PersonalDataConsentAt *time.Time
	PrivacyPolicyVersion  string
	TermsAccepted         bool
	TermsAcceptedAt       *time.Time
	TermsVersion          string
	ConsentIP             string
	ConsentUserAgent      string
	IsActive              bool
	LastLoginAt           *time.Time
	CreatedAt             time.Time
	UpdatedAt             time.Time
	DeletedAt             *time.Time
}

type TaxiPark struct {
	ID                 uuid.UUID
	OwnerUserID        uuid.UUID
	CityID             uuid.UUID
	Name               string
	LegalName          string
	TaxID              string
	ContactPhone       string
	ContactEmail       string
	IsVerified         bool
	VerificationStatus VerificationStatus
	CreatedAt          time.Time
	UpdatedAt          time.Time
	DeletedAt          *time.Time
}

type VerificationCode struct {
	ID          uuid.UUID
	UserID      uuid.UUID
	Target      string
	Channel     VerificationChannel
	Purpose     VerificationPurpose
	CodeHash    string
	Attempts    int
	MaxAttempts int
	ExpiresAt   time.Time
	ConsumedAt  *time.Time
	CreatedAt   time.Time
	LastSentAt  time.Time
}

type ConsentDocumentType string

const (
	ConsentDocumentPersonalData  ConsentDocumentType = "personal_data"
	ConsentDocumentPrivacyPolicy ConsentDocumentType = "privacy_policy"
	ConsentDocumentTerms         ConsentDocumentType = "terms"
)

type ConsentEventType string

const (
	ConsentEventAccepted ConsentEventType = "accepted"
	ConsentEventRevoked  ConsentEventType = "revoked"
	ConsentEventRenewed  ConsentEventType = "renewed"
)

type UserConsentEvent struct {
	ID              uuid.UUID
	UserID          uuid.UUID
	EventType       ConsentEventType
	DocumentType    ConsentDocumentType
	DocumentVersion string
	IP              string
	UserAgent       string
	CreatedAt       time.Time
}

var (
	ErrInvalidPhone            = errors.New("invalid phone")
	ErrInvalidEmail            = errors.New("invalid email")
	ErrInvalidUserRole         = errors.New("invalid user role")
	ErrInvalidRegistrationType = errors.New("invalid registration type")
	ErrConsentRequired         = errors.New("personal data consent required")
)

var phonePattern = regexp.MustCompile(`^\+[1-9]\d{7,14}$`)

func NormalizePhone(phone string) (string, error) {
	normalized := strings.TrimSpace(phone)
	normalized = strings.ReplaceAll(normalized, " ", "")
	normalized = strings.ReplaceAll(normalized, "-", "")
	normalized = strings.ReplaceAll(normalized, "(", "")
	normalized = strings.ReplaceAll(normalized, ")", "")

	if !phonePattern.MatchString(normalized) {
		return "", ErrInvalidPhone
	}

	return normalized, nil
}

func NormalizeEmail(email string) (string, error) {
	normalized := strings.ToLower(strings.TrimSpace(email))
	if normalized == "" {
		return "", ErrInvalidEmail
	}

	parsedEmail, err := mail.ParseAddress(normalized)
	if err != nil || parsedEmail.Address != normalized {
		return "", ErrInvalidEmail
	}

	return normalized, nil
}

func RoleFromRegistrationType(registrationType RegistrationType) (UserRole, error) {
	switch registrationType {
	case RegistrationTypePassenger:
		return UserRolePassenger, nil
	case RegistrationTypeDriver:
		return UserRoleDriver, nil
	case RegistrationTypeTaxiPark:
		return UserRoleTaxiPark, nil
	default:
		return "", ErrInvalidRegistrationType
	}
}

func (role UserRole) Validate() error {
	switch role {
	case UserRolePassenger, UserRoleDriver, UserRoleTaxiPark, UserRoleAdmin, UserRoleDispatcher:
		return nil
	default:
		return ErrInvalidUserRole
	}
}

func (registrationType RegistrationType) Validate() error {
	switch registrationType {
	case RegistrationTypePassenger, RegistrationTypeDriver, RegistrationTypeTaxiPark:
		return nil
	default:
		return ErrInvalidRegistrationType
	}
}

func ValidateRequiredRegistrationConsent(personalDataConsent bool, termsAccepted bool, privacyPolicyVersion string, termsVersion string) error {
	if !personalDataConsent || !termsAccepted {
		return ErrConsentRequired
	}
	if strings.TrimSpace(privacyPolicyVersion) == "" || strings.TrimSpace(termsVersion) == "" {
		return ErrConsentRequired
	}

	return nil
}
