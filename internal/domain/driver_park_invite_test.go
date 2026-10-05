package domain

import (
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestDriverParkInviteEnsurePending(t *testing.T) {
	now := time.Date(2026, 10, 3, 12, 0, 0, 0, time.UTC)
	tests := []struct {
		name   string
		status DriverParkInviteStatus
		expiry time.Time
		want   error
	}{
		{name: "pending before expiry", status: DriverParkInvitePending, expiry: now.Add(time.Second)},
		{name: "expiry boundary is closed", status: DriverParkInvitePending, expiry: now, want: ErrParkInviteExpired},
		{name: "pending after expiry", status: DriverParkInvitePending, expiry: now.Add(-time.Second), want: ErrParkInviteExpired},
		{name: "materialized expiry", status: DriverParkInviteExpired, expiry: now.Add(time.Hour), want: ErrParkInviteExpired},
		{name: "accepted is closed", status: DriverParkInviteAccepted, expiry: now.Add(time.Hour), want: ErrParkInviteClosed},
		{name: "declined is closed", status: DriverParkInviteDeclined, expiry: now.Add(time.Hour), want: ErrParkInviteClosed},
		{name: "cancelled is closed", status: DriverParkInviteCancelled, expiry: now.Add(time.Hour), want: ErrParkInviteClosed},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := (DriverParkInvite{Status: test.status, ExpiresAt: test.expiry}).EnsurePending(now)
			if !errors.Is(err, test.want) {
				t.Fatalf("EnsurePending() error = %v, want %v", err, test.want)
			}
		})
	}
}

func TestDriverParkInviteEnsureTransfer(t *testing.T) {
	source := uuid.New()
	target := uuid.New()
	other := uuid.New()
	tests := []struct {
		name        string
		inviteFrom  *uuid.UUID
		currentPark *uuid.UUID
		status      DriverStatus
		hasOrders   bool
		want        error
	}{
		{name: "offline driver can transfer", inviteFrom: &source, currentPark: &source, status: DriverStatusOffline},
		{name: "unaffiliated driver can join", status: DriverStatusOffline},
		{name: "source park changed", inviteFrom: &source, currentPark: &other, status: DriverStatusOffline, want: ErrParkInviteContextChanged},
		{name: "driver became affiliated", currentPark: &other, status: DriverStatusOffline, want: ErrParkInviteContextChanged},
		{name: "driver became unaffiliated", inviteFrom: &source, status: DriverStatusOffline, want: ErrParkInviteContextChanged},
		{name: "already in target park", inviteFrom: &source, currentPark: &target, status: DriverStatusOffline, want: ErrParkInviteContextChanged},
		{name: "same source and target", inviteFrom: &target, currentPark: &target, status: DriverStatusOffline, want: ErrParkInviteSamePark},
		{name: "online driver", inviteFrom: &source, currentPark: &source, status: DriverStatusOnline, want: ErrParkInviteDriverWorking},
		{name: "busy driver", inviteFrom: &source, currentPark: &source, status: DriverStatusBusy, want: ErrParkInviteDriverWorking},
		{name: "unfinished order", inviteFrom: &source, currentPark: &source, status: DriverStatusOffline, hasOrders: true, want: ErrParkInviteDriverWorking},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			invite := DriverParkInvite{FromTaxiParkID: test.inviteFrom, ToTaxiParkID: target}
			err := invite.EnsureTransfer(test.currentPark, test.status, test.hasOrders)
			if !errors.Is(err, test.want) {
				t.Fatalf("EnsureTransfer() error = %v, want %v", err, test.want)
			}
		})
	}
}
