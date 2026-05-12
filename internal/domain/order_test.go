package domain

import (
	"errors"
	"testing"

	"github.com/google/uuid"
)

func TestEnsureOrderStatusTransitionAllowsMainPassengerScenario(t *testing.T) {
	t.Parallel()

	transitions := []struct {
		from OrderStatus
		to   OrderStatus
	}{
		{from: OrderStatusCreated, to: OrderStatusSearching},
		{from: OrderStatusSearching, to: OrderStatusDriverAssigned},
		{from: OrderStatusDriverAssigned, to: OrderStatusDriverArriving},
		{from: OrderStatusDriverArriving, to: OrderStatusDriverWaiting},
		{from: OrderStatusDriverWaiting, to: OrderStatusInProgress},
		{from: OrderStatusInProgress, to: OrderStatusCompleted},
	}

	for _, transition := range transitions {
		transition := transition
		t.Run(string(transition.from)+" to "+string(transition.to), func(t *testing.T) {
			t.Parallel()

			if err := EnsureOrderStatusTransition(transition.from, transition.to); err != nil {
				t.Fatalf("expected transition to be allowed: %v", err)
			}
		})
	}
}

func TestEnsureOrderStatusTransitionRejectsInvalidTransition(t *testing.T) {
	t.Parallel()

	err := EnsureOrderStatusTransition(OrderStatusCompleted, OrderStatusSearching)
	if !errors.Is(err, ErrInvalidOrderStatusTransition) {
		t.Fatalf("expected invalid transition error, got %v", err)
	}
}

func TestEnsureOrderStatusTransitionRejectsForbiddenProductionTransitions(t *testing.T) {
	t.Parallel()

	transitions := []struct {
		from OrderStatus
		to   OrderStatus
	}{
		{from: OrderStatusCompleted, to: OrderStatusSearching},
		{from: OrderStatusCancelled, to: OrderStatusSearching},
		{from: OrderStatusFailed, to: OrderStatusSearching},
		{from: OrderStatusInProgress, to: OrderStatusSearching},
		{from: OrderStatusDriverAssigned, to: OrderStatusCompleted},
		{from: OrderStatusInProgress, to: OrderStatusCancelled},
	}

	for _, transition := range transitions {
		transition := transition
		t.Run(string(transition.from)+" to "+string(transition.to), func(t *testing.T) {
			t.Parallel()

			err := EnsureOrderStatusTransition(transition.from, transition.to)
			if !errors.Is(err, ErrInvalidOrderStatusTransition) {
				t.Fatalf("expected invalid transition error, got %v", err)
			}
		})
	}
}

func TestOrderStatusIsTerminal(t *testing.T) {
	t.Parallel()

	for _, status := range []OrderStatus{OrderStatusCompleted, OrderStatusCancelled, OrderStatusFailed} {
		status := status
		t.Run(string(status), func(t *testing.T) {
			t.Parallel()

			if !status.IsTerminal() {
				t.Fatalf("expected %s to be terminal", status)
			}
		})
	}
}

func TestNewOrderRatingValidatesScore(t *testing.T) {
	t.Parallel()

	_, err := NewOrderRating(uuid.New(), uuid.New(), uuid.New(), 6, "")
	if !errors.Is(err, ErrInvalidRatingScore) {
		t.Fatalf("expected invalid rating score error, got %v", err)
	}
}
