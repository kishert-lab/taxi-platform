package domain

import (
	"time"

	"github.com/google/uuid"
)

// TaxiParkDocument describes a driver or car document owned by a taxi park.
type TaxiParkDocument struct {
	ID           uuid.UUID
	DocumentType string
	Status       VerificationLifecycleStatus
	Number       string
	IssuedAt     *time.Time
	ExpiresAt    *time.Time
	FileURL      string
	Comment      string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}
