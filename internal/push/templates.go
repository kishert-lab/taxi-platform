package push

import (
	"strings"

	"github.com/google/uuid"
)

func PassengerOrderAssignedNotification(orderID uuid.UUID, driverID uuid.UUID) Notification {
	return Notification{
		Title: "Водитель найден",
		Body:  "Водитель принял ваш заказ.",
		Data: map[string]string{
			"event":     "order.driver_assigned",
			"order_id":  orderID.String(),
			"driver_id": driverID.String(),
		},
	}
}

func PassengerOrderWaitingNotification(orderID uuid.UUID, driverID uuid.UUID) Notification {
	return Notification{
		Title: "Водитель ожидает",
		Body:  "Водитель прибыл и ожидает вас.",
		Data: map[string]string{
			"event":     "order.driver_waiting",
			"order_id":  orderID.String(),
			"driver_id": driverID.String(),
		},
	}
}

func PassengerOrderArrivingNotification(orderID uuid.UUID, driverID uuid.UUID) Notification {
	return Notification{
		Title: "Водитель в пути",
		Body:  "Водитель едет к месту посадки.",
		Data: map[string]string{
			"event":     "order.driver_arriving",
			"order_id":  orderID.String(),
			"driver_id": driverID.String(),
		},
	}
}

func PassengerOrderCancelledByDriverNotification(orderID uuid.UUID, driverID uuid.UUID, reason string) Notification {
	return Notification{
		Title: "Заказ отменен водителем",
		Body:  "Водитель отменил поездку.",
		Data: map[string]string{
			"event":               "order.cancelled",
			"order_id":            orderID.String(),
			"driver_id":           driverID.String(),
			"cancelled_by":        "driver",
			"cancellation_reason": strings.TrimSpace(reason),
		},
	}
}
