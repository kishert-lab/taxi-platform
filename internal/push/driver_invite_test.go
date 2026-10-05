package push

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/kishert-lab/taxi-platform/internal/domain"
)

type inviteRealtimeCall struct {
	recipient uuid.UUID
	event     string
	payload   any
}

type inviteRealtimeRecorder struct {
	driverCalls []inviteRealtimeCall
	userCalls   []inviteRealtimeCall
	driverError error
	userError   error
}

func (recorder *inviteRealtimeRecorder) SendToDriver(_ context.Context, recipient uuid.UUID, event string, payload any) error {
	recorder.driverCalls = append(recorder.driverCalls, inviteRealtimeCall{recipient, event, payload})
	return recorder.driverError
}
func (recorder *inviteRealtimeRecorder) SendToUser(_ context.Context, recipient uuid.UUID, event string, payload any) error {
	recorder.userCalls = append(recorder.userCalls, inviteRealtimeCall{recipient, event, payload})
	return recorder.userError
}

type inviteNotificationRepositoryFake struct {
	tokens         []string
	recipients     map[uuid.UUID][]uuid.UUID
	tokenError     error
	recipientError error
}

func (repository *inviteNotificationRepositoryFake) ListDriverPushTokens(context.Context, uuid.UUID) ([]string, error) {
	return repository.tokens, repository.tokenError
}
func (repository *inviteNotificationRepositoryFake) ListParkInviteRecipients(_ context.Context, park uuid.UUID) ([]uuid.UUID, error) {
	if repository.recipientError != nil {
		return nil, repository.recipientError
	}
	return repository.recipients[park], nil
}

type pushProviderRecorder struct {
	tokens       []string
	notification Notification
	err          error
}

func (provider *pushProviderRecorder) SendToTokens(_ context.Context, tokens []string, notification Notification) error {
	provider.tokens = append([]string(nil), tokens...)
	provider.notification = notification
	return provider.err
}

func testInvite(status domain.DriverParkInviteStatus) domain.DriverParkInvite {
	source := uuid.New()
	return domain.DriverParkInvite{
		ID:               uuid.New(),
		DriverID:         uuid.New(),
		DriverName:       "Иван Иванов",
		FromTaxiParkID:   &source,
		FromTaxiParkName: "Такси А",
		ToTaxiParkID:     uuid.New(),
		ToTaxiParkName:   "Такси Б",
		Status:           status,
	}
}

func TestDriverInviteNotifierCreated(t *testing.T) {
	invite := testInvite(domain.DriverParkInvitePending)
	repository := &inviteNotificationRepositoryFake{tokens: []string{"first", "second"}}
	realtime := &inviteRealtimeRecorder{}
	provider := &pushProviderRecorder{}
	notifier := NewDriverInviteNotifier(repository, realtime, NewService(ServiceParams{Provider: provider, Enabled: true}))

	if err := notifier.Notify(context.Background(), invite, "driver.park_invite.created"); err != nil {
		t.Fatal(err)
	}
	if len(realtime.driverCalls) != 1 || realtime.driverCalls[0].recipient != invite.DriverID {
		t.Fatalf("driver realtime calls = %+v", realtime.driverCalls)
	}
	if len(realtime.userCalls) != 0 {
		t.Fatalf("pending invite unexpectedly notified park: %+v", realtime.userCalls)
	}
	if len(provider.tokens) != 2 || provider.tokens[0] != "first" || provider.tokens[1] != "second" {
		t.Fatalf("push tokens = %v", provider.tokens)
	}
	if provider.notification.Title != "Приглашение в таксопарк" || !strings.Contains(provider.notification.Body, "Такси Б") {
		t.Fatalf("push notification = %+v", provider.notification)
	}
	if provider.notification.Data["type"] != "driver.park_invite.created" || provider.notification.Data["invite_id"] != invite.ID.String() {
		t.Fatalf("push data = %v", provider.notification.Data)
	}
}

func TestDriverInviteNotifierAcceptedNotifiesBothParks(t *testing.T) {
	invite := testInvite(domain.DriverParkInviteAccepted)
	oldOwner, newOwner := uuid.New(), uuid.New()
	repository := &inviteNotificationRepositoryFake{recipients: map[uuid.UUID][]uuid.UUID{
		*invite.FromTaxiParkID: {oldOwner},
		invite.ToTaxiParkID:    {newOwner},
	}}
	realtime := &inviteRealtimeRecorder{}
	notifier := NewDriverInviteNotifier(repository, realtime, nil)

	if err := notifier.Notify(context.Background(), invite, "driver.park_invite.accepted"); err != nil {
		t.Fatal(err)
	}
	if len(realtime.driverCalls) != 1 || len(realtime.userCalls) != 2 {
		t.Fatalf("realtime calls: driver=%d park=%d", len(realtime.driverCalls), len(realtime.userCalls))
	}
	messages := map[uuid.UUID]string{}
	for _, call := range realtime.userCalls {
		payload, ok := call.payload.(struct {
			InviteID   uuid.UUID                     `json:"invite_id"`
			DriverID   uuid.UUID                     `json:"driver_id"`
			DriverName string                        `json:"driver_name"`
			Status     domain.DriverParkInviteStatus `json:"status"`
			Message    string                        `json:"message"`
		})
		if !ok {
			t.Fatalf("unexpected payload type %T", call.payload)
		}
		if payload.InviteID != invite.ID || payload.DriverID != invite.DriverID || payload.Status != domain.DriverParkInviteAccepted {
			t.Fatalf("unexpected payload: %+v", payload)
		}
		messages[call.recipient] = payload.Message
	}
	if !strings.Contains(messages[oldOwner], "Такси Б") || !strings.Contains(messages[oldOwner], invite.DriverName) {
		t.Fatalf("old park message = %q", messages[oldOwner])
	}
	if !strings.Contains(messages[newOwner], "присоединился") || !strings.Contains(messages[newOwner], invite.DriverName) {
		t.Fatalf("new park message = %q", messages[newOwner])
	}
}

func TestDriverInviteNotifierDeclinedOnlyNotifiesDestination(t *testing.T) {
	invite := testInvite(domain.DriverParkInviteDeclined)
	oldOwner, newOwner := uuid.New(), uuid.New()
	repository := &inviteNotificationRepositoryFake{recipients: map[uuid.UUID][]uuid.UUID{
		*invite.FromTaxiParkID: {oldOwner},
		invite.ToTaxiParkID:    {newOwner},
	}}
	realtime := &inviteRealtimeRecorder{}
	if err := NewDriverInviteNotifier(repository, realtime, nil).Notify(context.Background(), invite, "driver.park_invite.declined"); err != nil {
		t.Fatal(err)
	}
	if len(realtime.userCalls) != 1 || realtime.userCalls[0].recipient != newOwner {
		t.Fatalf("park calls = %+v", realtime.userCalls)
	}
}

func TestDriverInviteNotifierJoinsDeliveryErrors(t *testing.T) {
	invite := testInvite(domain.DriverParkInviteAccepted)
	driverError := errors.New("driver realtime failed")
	recipientError := errors.New("recipient lookup failed")
	repository := &inviteNotificationRepositoryFake{recipientError: recipientError}
	realtime := &inviteRealtimeRecorder{driverError: driverError}
	err := NewDriverInviteNotifier(repository, realtime, nil).Notify(context.Background(), invite, "driver.park_invite.accepted")
	if !errors.Is(err, driverError) || !errors.Is(err, recipientError) {
		t.Fatalf("Notify() error = %v", err)
	}
}

func TestDriverInviteNotifierPushErrors(t *testing.T) {
	invite := testInvite(domain.DriverParkInvitePending)
	providerError := errors.New("firebase failed")
	provider := &pushProviderRecorder{err: providerError}
	repository := &inviteNotificationRepositoryFake{tokens: []string{"token"}}
	notifier := NewDriverInviteNotifier(repository, &inviteRealtimeRecorder{}, NewService(ServiceParams{Provider: provider, Enabled: true}))
	err := notifier.Notify(context.Background(), invite, "driver.park_invite.created")
	if !errors.Is(err, providerError) {
		t.Fatalf("Notify() error = %v", err)
	}
}

func TestDriverInviteNotifierRespectsDisabledPushService(t *testing.T) {
	invite := testInvite(domain.DriverParkInvitePending)
	provider := &pushProviderRecorder{}
	repository := &inviteNotificationRepositoryFake{tokens: []string{"token"}}
	notifier := NewDriverInviteNotifier(repository, &inviteRealtimeRecorder{}, NewService(ServiceParams{Provider: provider, Enabled: false}))
	if err := notifier.Notify(context.Background(), invite, "driver.park_invite.created"); err != nil {
		t.Fatal(err)
	}
	if len(provider.tokens) != 0 {
		t.Fatalf("disabled push service sent to %v", provider.tokens)
	}
}
