package domain

import (
	"time"

	"github.com/google/uuid"
)

type Car struct {
	ID                      uuid.UUID
	TaxiParkID              uuid.UUID
	PrimaryDriverID         *uuid.UUID
	AttachedDriverIDs       []uuid.UUID
	Brand                   string
	Model                   string
	Year                    int
	PlateNumber             string
	VIN                     string
	STS                     string
	PTS                     string
	Color                   string
	CarClass                string
	VerificationStatus      VerificationLifecycleStatus
	OwnerDetails            string
	OSAGOExpiresAt          *time.Time
	DiagnosticCardExpiresAt *time.Time
	TaxiPermitNumber        string
	RegionalRegistryNumber  string
	PermitRegion            string
	PermitIssuedAt          *time.Time
	PermitExpiresAt         *time.Time
	TaxiPermitVerified      bool
	RegionalRegistryVerified bool
	RegionalRequirementsCompliant bool
	HasTaxiColorScheme      bool
	HasOrangeRoofLamp       bool
	HasPassengerInfo        bool
	OSAGOVerified           bool
	DiagnosticCardVerified  bool
	TechnicalStateVerified  bool
	LocalizationCompliant   bool
	LegalUseBasisVerified   bool
	VerificationCheckedAt   *time.Time
	VerificationCheckedBy   *uuid.UUID
	IsActive                bool
	CreatedAt               time.Time
	UpdatedAt               time.Time
	DeletedAt               *time.Time
}
