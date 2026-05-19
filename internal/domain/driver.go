package domain

import (
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
)

type DriverStatus string
type VerificationLifecycleStatus string

const (
	DriverStatusOffline DriverStatus = "offline"
	DriverStatusOnline  DriverStatus = "online"
	DriverStatusBusy    DriverStatus = "busy"
	DriverStatusPaused  DriverStatus = "paused"
	DriverStatusBlocked DriverStatus = "blocked"

	ComplianceStatusDraft               VerificationLifecycleStatus = "draft"
	ComplianceStatusPendingVerification VerificationLifecycleStatus = "pending_verification"
	ComplianceStatusVerified            VerificationLifecycleStatus = "verified"
	ComplianceStatusRejected            VerificationLifecycleStatus = "rejected"
	ComplianceStatusBlocked             VerificationLifecycleStatus = "blocked"
	ComplianceStatusArchived            VerificationLifecycleStatus = "archived"
)

type Driver struct {
	ID                    uuid.UUID
	UserID                uuid.UUID
	CityID                uuid.UUID
	TaxiParkID            *uuid.UUID
	Status                DriverStatus
	VerificationStatus    VerificationLifecycleStatus
	Rating                float64
	RatingsCount          int
	CompletedOrdersCount  int
	BirthDate             *time.Time
	LicenseSeries         string
	LicenseNumber         string
	LicenseCategory       string
	LicenseIssuedAt       *time.Time
	LicenseExpiresAt      *time.Time
	DrivingExperienceFrom *time.Time
	HasNoTaxiWorkRestrictions bool
	FederalLaw580Compliant    bool
	RegionalRequirementsCompliant bool
	MedicalCheckPassed        bool
	PretripControlRequired    bool
	PretripControlPassed      bool
	NoTransportBan            bool
	VerificationCheckedAt     *time.Time
	VerificationCheckedBy     *uuid.UUID
	CommissionBasisPoints *int32
	IsVerified            bool
	BlockedReason         string
	TaxiParkComment       string
	MustChangePassword    bool
	CreatedAt             time.Time
	UpdatedAt             time.Time
	DeletedAt             *time.Time
}

var (
	ErrInvalidDriverStatus           = errors.New("invalid driver status")
	ErrInvalidDriverStatusTransition = errors.New("invalid driver status transition")
	ErrInvalidVerificationStatus     = errors.New("invalid verification status")
)

var allowedDriverStatusTransitions = map[DriverStatus][]DriverStatus{
	DriverStatusOffline: {DriverStatusOnline, DriverStatusBlocked},
	DriverStatusOnline:  {DriverStatusBusy, DriverStatusPaused, DriverStatusOffline, DriverStatusBlocked},
	DriverStatusBusy:    {DriverStatusOnline, DriverStatusBlocked},
	DriverStatusPaused:  {DriverStatusOnline, DriverStatusOffline, DriverStatusBlocked},
	DriverStatusBlocked: {},
}

func (status DriverStatus) Validate() error {
	switch status {
	case DriverStatusOffline, DriverStatusOnline, DriverStatusBusy, DriverStatusPaused, DriverStatusBlocked:
		return nil
	default:
		return ErrInvalidDriverStatus
	}
}

func (status VerificationLifecycleStatus) Validate() error {
	switch status {
	case ComplianceStatusDraft,
		ComplianceStatusPendingVerification,
		ComplianceStatusVerified,
		ComplianceStatusRejected,
		ComplianceStatusBlocked,
		ComplianceStatusArchived:
		return nil
	default:
		return ErrInvalidVerificationStatus
	}
}

func CanTransitionDriverStatus(from DriverStatus, to DriverStatus) bool {
	if from == to {
		return true
	}

	for _, allowedStatus := range allowedDriverStatusTransitions[from] {
		if allowedStatus == to {
			return true
		}
	}

	return false
}

func EnsureDriverStatusTransition(from DriverStatus, to DriverStatus) error {
	if err := from.Validate(); err != nil {
		return fmt.Errorf("validate source driver status: %w", err)
	}
	if err := to.Validate(); err != nil {
		return fmt.Errorf("validate target driver status: %w", err)
	}
	if !CanTransitionDriverStatus(from, to) {
		return fmt.Errorf("%w: %s -> %s", ErrInvalidDriverStatusTransition, from, to)
	}

	return nil
}
