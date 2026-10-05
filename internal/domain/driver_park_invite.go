package domain

import (
	"errors"
	"github.com/google/uuid"
	"time"
)

type DriverParkInviteStatus string

const (
	DriverParkInvitePending   DriverParkInviteStatus = "pending"
	DriverParkInviteAccepted  DriverParkInviteStatus = "accepted"
	DriverParkInviteDeclined  DriverParkInviteStatus = "declined"
	DriverParkInviteCancelled DriverParkInviteStatus = "cancelled"
	DriverParkInviteExpired   DriverParkInviteStatus = "expired"
)

var (
	ErrParkInviteNotFound       = errors.New("Приглашение или водитель не найдены")
	ErrParkInviteClosed         = errors.New("Приглашение уже закрыто")
	ErrParkInviteExpired        = errors.New("Срок приглашения истёк")
	ErrParkInviteSamePark       = errors.New("Водитель уже состоит в этом таксопарке")
	ErrParkInviteInactivePark   = errors.New("Таксопарк недоступен")
	ErrParkInviteDriverWorking  = errors.New("Сначала завершите текущие заказы и выйдите с линии")
	ErrParkInviteContextChanged = errors.New("Текущий парк водителя изменился")
	ErrParkInviteForbidden      = errors.New("Недостаточно прав для управления приглашениями")
)

type DriverParkInvite struct {
	ID                 uuid.UUID              `json:"id"`
	DriverID           uuid.UUID              `json:"driver_id"`
	FromTaxiParkID     *uuid.UUID             `json:"from_taxi_park_id"`
	ToTaxiParkID       uuid.UUID              `json:"to_taxi_park_id"`
	FromTaxiParkName   string                 `json:"from_taxi_park_name,omitempty"`
	ToTaxiParkName     string                 `json:"to_taxi_park_name"`
	DriverName         string                 `json:"driver_name"`
	Status             DriverParkInviteStatus `json:"status"`
	CreatedByUserID    uuid.UUID              `json:"created_by_user_id"`
	CreatedAt          time.Time              `json:"created_at"`
	ExpiresAt          time.Time              `json:"expires_at"`
	AcceptedAt         *time.Time             `json:"accepted_at,omitempty"`
	DeclinedAt         *time.Time             `json:"declined_at,omitempty"`
	CancelledAt        *time.Time             `json:"cancelled_at,omitempty"`
	CancellationReason string                 `json:"cancellation_reason,omitempty"`
}

func (invite DriverParkInvite) EnsurePending(now time.Time) error {
	if invite.Status == DriverParkInviteExpired || (invite.Status == DriverParkInvitePending && !now.Before(invite.ExpiresAt)) {
		return ErrParkInviteExpired
	}
	if invite.Status != DriverParkInvitePending {
		return ErrParkInviteClosed
	}
	return nil
}

func (invite DriverParkInvite) EnsureTransfer(currentPark *uuid.UUID, status DriverStatus, hasOrders bool) error {
	if (currentPark == nil) != (invite.FromTaxiParkID == nil) || (currentPark != nil && invite.FromTaxiParkID != nil && *currentPark != *invite.FromTaxiParkID) {
		return ErrParkInviteContextChanged
	}
	if currentPark != nil && *currentPark == invite.ToTaxiParkID {
		return ErrParkInviteSamePark
	}
	if status != DriverStatusOffline || hasOrders {
		return ErrParkInviteDriverWorking
	}
	return nil
}
