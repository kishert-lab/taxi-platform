package chat

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/kishert-lab/taxi-platform/internal/domain"
	"github.com/kishert-lab/taxi-platform/internal/dto"
)

func TestSendDispatcherDriverMessagePublishesToDriverAndTaxiPark(t *testing.T) {
	orderID := uuid.New()
	taxiParkID := uuid.New()
	driverID := uuid.New()
	driverUserID := uuid.New()
	dispatcherUserID := uuid.New()
	passengerID := uuid.New()
	threadID := uuid.New()

	repository := &fakeChatRepository{
		thread: domain.ChatThread{
			ID:          threadID,
			Type:        domain.ChatTypeDispatcherDriver,
			OrderID:     &orderID,
			TaxiParkID:  &taxiParkID,
			PassengerID: &passengerID,
			DriverID:    &driverID,
			Status:      domain.ChatStatusOpen,
			CreatedAt:   time.Now().UTC(),
			UpdatedAt:   time.Now().UTC(),
		},
		orderContext: OrderChatContext{
			OrderID:      orderID,
			Status:       domain.OrderStatusDriverAssigned,
			PassengerID:  passengerID,
			DriverID:     &driverID,
			DriverUserID: &driverUserID,
			TaxiParkID:   &taxiParkID,
		},
		taxiParkActorAllowed: true,
	}
	gateway := &fakeChatRealtimeGateway{}
	service := NewService(repository, gateway, nil)

	message, err := service.SendOrderMessage(context.Background(), dispatcherUserID, domain.UserRoleDispatcher, orderID, domain.ChatTypeDispatcherDriver, testChatRequest("  hello driver  "))
	if err != nil {
		t.Fatalf("send message: %v", err)
	}
	if message.Body != "hello driver" {
		t.Fatalf("expected normalized body, got %q", message.Body)
	}
	if gateway.driverEvents != 1 {
		t.Fatalf("expected driver event, got %d", gateway.driverEvents)
	}
	if gateway.taxiParkEvents != 1 {
		t.Fatalf("expected taxi park event, got %d", gateway.taxiParkEvents)
	}
}

func TestSendDriverPassengerMessageRejectsUnrelatedPassenger(t *testing.T) {
	orderID := uuid.New()
	driverID := uuid.New()
	driverUserID := uuid.New()
	passengerID := uuid.New()
	unrelatedPassengerID := uuid.New()

	service := NewService(&fakeChatRepository{
		orderContext: OrderChatContext{
			OrderID:      orderID,
			Status:       domain.OrderStatusDriverAssigned,
			PassengerID:  passengerID,
			DriverID:     &driverID,
			DriverUserID: &driverUserID,
		},
	}, &fakeChatRealtimeGateway{}, nil)

	_, err := service.SendOrderMessage(context.Background(), unrelatedPassengerID, domain.UserRolePassenger, orderID, domain.ChatTypeDriverPassenger, testChatRequest("hello"))
	if !errors.Is(err, ErrChatForbidden) {
		t.Fatalf("expected ErrChatForbidden, got %v", err)
	}
}

func TestSendDriverPassengerMessageRejectsTerminalOrder(t *testing.T) {
	orderID := uuid.New()
	driverID := uuid.New()
	driverUserID := uuid.New()
	passengerID := uuid.New()

	service := NewService(&fakeChatRepository{
		orderContext: OrderChatContext{
			OrderID:      orderID,
			Status:       domain.OrderStatusCompleted,
			PassengerID:  passengerID,
			DriverID:     &driverID,
			DriverUserID: &driverUserID,
		},
	}, &fakeChatRealtimeGateway{}, nil)

	_, err := service.SendOrderMessage(context.Background(), passengerID, domain.UserRolePassenger, orderID, domain.ChatTypeDriverPassenger, testChatRequest("hello"))
	if !errors.Is(err, ErrChatUnavailable) {
		t.Fatalf("expected ErrChatUnavailable, got %v", err)
	}
}

func testChatRequest(body string) dto.ChatSendMessageRequest {
	return dto.ChatSendMessageRequest{Body: body}
}

type fakeChatRepository struct {
	thread               domain.ChatThread
	orderContext         OrderChatContext
	taxiParkActorAllowed bool
}

func (repository *fakeChatRepository) EnsureOrderThread(context.Context, uuid.UUID, domain.ChatType) (domain.ChatThread, error) {
	if repository.thread.ID == uuid.Nil {
		repository.thread = domain.ChatThread{ID: uuid.New(), Type: domain.ChatTypeDriverPassenger}
	}
	return repository.thread, nil
}

func (repository *fakeChatRepository) EnsurePassengerSupportThread(context.Context, uuid.UUID) (domain.ChatThread, error) {
	return domain.ChatThread{ID: uuid.New(), Type: domain.ChatTypePassengerSupport}, nil
}

func (repository *fakeChatRepository) CreateMessage(_ context.Context, thread domain.ChatThread, senderUserID uuid.UUID, senderRole domain.UserRole, body string) (domain.ChatMessage, error) {
	return domain.ChatMessage{
		ID:           uuid.New(),
		ThreadID:     thread.ID,
		OrderID:      thread.OrderID,
		SenderUserID: senderUserID,
		SenderRole:   senderRole,
		Body:         body,
		CreatedAt:    time.Now().UTC(),
	}, nil
}

func (repository *fakeChatRepository) ListMessages(context.Context, domain.ChatThread, int) ([]domain.ChatMessage, error) {
	return nil, nil
}

func (repository *fakeChatRepository) GetOrderChatContext(context.Context, uuid.UUID) (OrderChatContext, error) {
	return repository.orderContext, nil
}

func (repository *fakeChatRepository) IsTaxiParkActor(context.Context, uuid.UUID, uuid.UUID) (bool, error) {
	return repository.taxiParkActorAllowed, nil
}

type fakeChatRealtimeGateway struct {
	driverEvents    int
	passengerEvents int
	taxiParkEvents  int
}

func (gateway *fakeChatRealtimeGateway) SendToDriver(context.Context, uuid.UUID, string, any) error {
	gateway.driverEvents++
	return nil
}

func (gateway *fakeChatRealtimeGateway) SendToPassenger(context.Context, uuid.UUID, string, any) error {
	gateway.passengerEvents++
	return nil
}

func (gateway *fakeChatRealtimeGateway) SendToTaxiParkByOrder(context.Context, uuid.UUID, string, any) error {
	gateway.taxiParkEvents++
	return nil
}
