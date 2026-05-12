package domain

import (
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
)

type LegalDocumentType string

const (
	LegalDocumentPrivacyPolicy       LegalDocumentType = "privacy_policy"
	LegalDocumentTermsOfService      LegalDocumentType = "terms_of_service"
	LegalDocumentDriverAgreement     LegalDocumentType = "driver_agreement"
	LegalDocumentTaxiParkAgreement   LegalDocumentType = "taxi_park_agreement"
	LegalDocumentConsentPersonalData LegalDocumentType = "consent_personal_data"
)

type LegalDocument struct {
	ID           uuid.UUID
	DocumentType LegalDocumentType
	Version      string
	Title        string
	Content      string
	Language     string
	IsActive     bool
	CreatedAt    time.Time
}

type UserDocumentAcceptance struct {
	ID              uuid.UUID
	UserID          uuid.UUID
	DocumentID      uuid.UUID
	DocumentVersion string
	AcceptedAt      time.Time
	IP              string
	UserAgent       string
}

var (
	ErrInvalidLegalDocumentType = errors.New("invalid legal document type")
	ErrInvalidLegalDocument     = errors.New("invalid legal document")
)

func (documentType LegalDocumentType) Validate() error {
	switch documentType {
	case LegalDocumentPrivacyPolicy,
		LegalDocumentTermsOfService,
		LegalDocumentDriverAgreement,
		LegalDocumentTaxiParkAgreement,
		LegalDocumentConsentPersonalData:
		return nil
	default:
		return ErrInvalidLegalDocumentType
	}
}

func NewLegalDocument(documentType LegalDocumentType, version string, title string, content string, language string) (LegalDocument, error) {
	if err := documentType.Validate(); err != nil {
		return LegalDocument{}, err
	}
	if strings.TrimSpace(version) == "" || strings.TrimSpace(title) == "" || strings.TrimSpace(content) == "" {
		return LegalDocument{}, ErrInvalidLegalDocument
	}
	if strings.TrimSpace(language) == "" {
		language = "ru"
	}

	return LegalDocument{
		DocumentType: documentType,
		Version:      strings.TrimSpace(version),
		Title:        strings.TrimSpace(title),
		Content:      content,
		Language:     strings.TrimSpace(language),
	}, nil
}
