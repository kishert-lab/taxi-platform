package order

import "github.com/kishert-lab/taxi-platform/internal/domain"

type Action string

const (
	ActionCancel        Action = "cancel"
	ActionCallDriver    Action = "call_driver"
	ActionRate          Action = "rate"
	ActionAccept        Action = "accept"
	ActionReject        Action = "reject"
	ActionArrived       Action = "arrived"
	ActionStart         Action = "start"
	ActionComplete      Action = "complete"
	ActionCallPassenger Action = "call_passenger"
)

func AllowedPassengerActions(status domain.OrderStatus) []Action {
	switch status {
	case domain.OrderStatusSearching:
		return []Action{ActionCancel}
	case domain.OrderStatusDriverAssigned, domain.OrderStatusDriverArriving, domain.OrderStatusDriverWaiting:
		return []Action{ActionCancel, ActionCallDriver}
	case domain.OrderStatusCompleted:
		return []Action{ActionRate}
	default:
		return []Action{}
	}
}

func AllowedDriverActions(status domain.OrderStatus, hasActiveOffer bool) []Action {
	if hasActiveOffer && status == domain.OrderStatusSearching {
		return []Action{ActionAccept, ActionReject}
	}
	switch status {
	case domain.OrderStatusDriverAssigned:
		return []Action{ActionArrived, ActionCallPassenger}
	case domain.OrderStatusDriverWaiting:
		return []Action{ActionStart, ActionCallPassenger}
	case domain.OrderStatusInProgress:
		return []Action{ActionComplete, ActionCallPassenger}
	default:
		return []Action{}
	}
}
