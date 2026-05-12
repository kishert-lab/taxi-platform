package domain

import (
	"errors"
	"testing"
)

func TestEnsureDriverStatusTransitionAllowsMainScenario(t *testing.T) {
	t.Parallel()

	transitions := []struct {
		from DriverStatus
		to   DriverStatus
	}{
		{from: DriverStatusOffline, to: DriverStatusOnline},
		{from: DriverStatusOnline, to: DriverStatusBusy},
		{from: DriverStatusBusy, to: DriverStatusOnline},
		{from: DriverStatusOnline, to: DriverStatusPaused},
		{from: DriverStatusPaused, to: DriverStatusOffline},
	}

	for _, transition := range transitions {
		transition := transition
		t.Run(string(transition.from)+" to "+string(transition.to), func(t *testing.T) {
			t.Parallel()

			if err := EnsureDriverStatusTransition(transition.from, transition.to); err != nil {
				t.Fatalf("expected transition to be allowed: %v", err)
			}
		})
	}
}

func TestEnsureDriverStatusTransitionRejectsBlockedToOnline(t *testing.T) {
	t.Parallel()

	err := EnsureDriverStatusTransition(DriverStatusBlocked, DriverStatusOnline)
	if !errors.Is(err, ErrInvalidDriverStatusTransition) {
		t.Fatalf("expected invalid driver transition error, got %v", err)
	}
}
