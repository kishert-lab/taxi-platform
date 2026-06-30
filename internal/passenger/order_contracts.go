package passenger

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/kishert-lab/taxi-platform/internal/domain"
	"github.com/kishert-lab/taxi-platform/internal/dto"
	geodomain "github.com/kishert-lab/taxi-platform/internal/geocoder/domain"
	geoservice "github.com/kishert-lab/taxi-platform/internal/geocoder/service"
)

type OrderRepository interface {
	ListActiveCarClasses(ctx context.Context) ([]domain.CarClass, error)
	GetActiveCarClassByID(ctx context.Context, carClassID uuid.UUID) (domain.CarClass, error)
	EstimateRoute(ctx context.Context, pickup geodomain.Coordinates, destination geodomain.Coordinates) (float64, error)
	CreatePassengerOrder(ctx context.Context, record CreateOrderRecord) (PassengerOrderDetails, error)
	GetCurrentPassengerOrder(ctx context.Context, passengerID uuid.UUID) (PassengerOrderDetails, error)
	ListPassengerOrderHistory(ctx context.Context, passengerID uuid.UUID, limit int) ([]PassengerOrderDetails, error)
	GetPassengerOrder(ctx context.Context, passengerID uuid.UUID, orderID uuid.UUID) (PassengerOrderDetails, error)
	CancelPassengerOrder(ctx context.Context, passengerID uuid.UUID, orderID uuid.UUID, reason string, cancelledAt time.Time) (PassengerOrderDetails, error)
}

type DispatchQueue interface {
	EnqueueOrder(ctx context.Context, orderID uuid.UUID) error
}

type CityResolver interface {
	ResolveCityByCoordinates(ctx context.Context, coordinates geodomain.Coordinates) (geoservice.CityContext, bool, error)
}

type OrdersUseCase interface {
	ListPassengerCarClasses(ctx context.Context, passengerID uuid.UUID) (dto.PassengerCarClassesResponse, error)
	EstimatePassengerOrder(ctx context.Context, passengerID uuid.UUID, request dto.OrderEstimateRequest) (dto.OrderEstimateResponse, error)
	CreatePassengerOrder(ctx context.Context, passengerID uuid.UUID, request dto.PassengerCreateOrderRequest) (dto.PassengerOrderResponse, error)
	GetCurrentPassengerOrder(ctx context.Context, passengerID uuid.UUID) (dto.PassengerOrderResponse, error)
	ListPassengerOrderHistory(ctx context.Context, passengerID uuid.UUID) (dto.OrderHistoryResponse, error)
	GetPassengerOrder(ctx context.Context, passengerID uuid.UUID, orderID uuid.UUID) (dto.PassengerOrderResponse, error)
	CancelPassengerOrder(ctx context.Context, passengerID uuid.UUID, orderID uuid.UUID, request dto.CancelOrderRequest) (dto.PassengerOrderResponse, error)
	RatePassengerOrder(ctx context.Context, passengerID uuid.UUID, orderID uuid.UUID, request dto.RateOrderRequest) (dto.PassengerOrderResponse, error)
}

type CreateOrderRecord struct {
	PassengerID                     uuid.UUID
	CityID                          uuid.UUID
	CarClassID                      uuid.UUID
	PickupAddress                   string
	PickupEntrance                  string
	PickupComment                   string
	PickupLocation                  geodomain.Coordinates
	DestinationAddress              string
	DestinationLocation             geodomain.Coordinates
	EstimatedPrice                  domain.Money
	PaymentMethod                   domain.PaymentMethod
	PassengerComment                string
	PassengerLocationSharingEnabled bool
}

type PassengerOrderDetails struct {
	Order    domain.Order
	CarClass *domain.CarClass
	Driver   *PassengerAssignedDriver
	Car      *PassengerAssignedCar
}

type PassengerAssignedDriver struct {
	ID           uuid.UUID
	Name         string
	Phone        string
	AvatarURL    string
	Rating       float64
	RatingsCount int
}

type PassengerAssignedCar struct {
	ID          uuid.UUID
	Brand       string
	Model       string
	Color       string
	PlateNumber string
	CarClass    string
}
