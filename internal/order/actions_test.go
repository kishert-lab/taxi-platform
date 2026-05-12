package order

import (
	"reflect"
	"testing"

	"github.com/kishert-lab/taxi-platform/internal/domain"
)

func TestAllowedPassengerActions(t *testing.T) {
	t.Parallel()

	actions := AllowedPassengerActions(domain.OrderStatusDriverArriving)
	expected := []Action{ActionCancel, ActionCallDriver}
	if !reflect.DeepEqual(actions, expected) {
		t.Fatalf("expected %v, got %v", expected, actions)
	}
}

func TestAllowedDriverActions(t *testing.T) {
	t.Parallel()

	actions := AllowedDriverActions(domain.OrderStatusSearching, true)
	expected := []Action{ActionAccept, ActionReject}
	if !reflect.DeepEqual(actions, expected) {
		t.Fatalf("expected %v, got %v", expected, actions)
	}
}
