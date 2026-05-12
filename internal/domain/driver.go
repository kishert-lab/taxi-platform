package domain

import (
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
)

type DriverStatus string

const (
	DriverStatusOffline DriverStatus = "offline"
	DriverStatusOnline  DriverStatus = "online"
	DriverStatusBusy    DriverStatus = "busy"
	DriverStatusPaused  DriverStatus = "paused"
	DriverStatusBlocked DriverStatus = "blocked"
)

type Driver struct {
	ID                    uuid.UUID
	UserID                uuid.UUID
	CityID                uuid.UUID
	TaxiParkID            *uuid.UUID
	Status                DriverStatus
	Rating                float64
	CompletedOrdersCount  int
	LicenseNumber         string
	CommissionBasisPoints *int32
	IsVerified            bool
	BlockedReason         string
	CreatedAt             time.Time
	UpdatedAt             time.Time
	DeletedAt             *time.Time
}

var (
	ErrInvalidDriverStatus           = errors.New("invalid driver status")
	ErrInvalidDriverStatusTransition = errors.New("invalid driver status transition")
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
