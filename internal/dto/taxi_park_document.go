package dto

import (
	"time"

	"github.com/google/uuid"

	"github.com/kishert-lab/taxi-platform/internal/domain"
)

// TaxiParkDocumentResponse is returned by taxi park document endpoints.
type TaxiParkDocumentResponse struct {
	ID           uuid.UUID                          `json:"id" example:"66666666-6666-6666-6666-666666666666"`
	DocumentType string                             `json:"document_type" example:"license"`
	Status       domain.VerificationLifecycleStatus `json:"status" example:"pending_verification"`
	Number       string                             `json:"number,omitempty" example:"77AA000000"`
	IssuedAt     *time.Time                         `json:"issued_at,omitempty" example:"2026-01-31T00:00:00Z"`
	ExpiresAt    *time.Time                         `json:"expires_at,omitempty" example:"2031-01-31T00:00:00Z"`
	FileURL      string                             `json:"file_url,omitempty" example:"https://cdn.example/doc.pdf"`
	Comment      string                             `json:"comment,omitempty" example:"Checked by park"`
	CreatedAt    time.Time                          `json:"created_at" example:"2026-05-19T12:00:00Z"`
	UpdatedAt    time.Time                          `json:"updated_at" example:"2026-05-19T12:00:00Z"`
}

// TaxiParkDocumentsResponse wraps taxi park documents.
type TaxiParkDocumentsResponse struct {
	Documents []TaxiParkDocumentResponse `json:"documents"`
}

func TaxiParkDocumentFromDomain(document domain.TaxiParkDocument) TaxiParkDocumentResponse {
	return TaxiParkDocumentResponse{
		ID:           document.ID,
		DocumentType: document.DocumentType,
		Status:       document.Status,
		Number:       document.Number,
		IssuedAt:     document.IssuedAt,
		ExpiresAt:    document.ExpiresAt,
		FileURL:      document.FileURL,
		Comment:      document.Comment,
		CreatedAt:    document.CreatedAt,
		UpdatedAt:    document.UpdatedAt,
	}
}
