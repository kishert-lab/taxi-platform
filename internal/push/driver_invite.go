package push

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/kishert-lab/taxi-platform/internal/domain"
)

type InviteRealtime interface {
	SendToDriver(context.Context, uuid.UUID, string, any) error
	SendToUser(context.Context, uuid.UUID, string, any) error
}
type InviteNotificationRepository interface {
	ListDriverPushTokens(context.Context, uuid.UUID) ([]string, error)
	ListParkInviteRecipients(context.Context, uuid.UUID) ([]uuid.UUID, error)
}
type DriverInviteNotifier struct {
	repository InviteNotificationRepository
	realtime   InviteRealtime
	push       *Service
}

func NewDriverInviteNotifier(repository InviteNotificationRepository, realtime InviteRealtime, push *Service) *DriverInviteNotifier {
	return &DriverInviteNotifier{repository, realtime, push}
}
func (notifier *DriverInviteNotifier) Notify(parent context.Context, invite domain.DriverParkInvite, event string) error {
	ctx, cancel := context.WithTimeout(parent, 8*time.Second)
	defer cancel()
	var failures []error
	if err := notifier.realtime.SendToDriver(ctx, invite.DriverID, event, invite); err != nil {
		failures = append(failures, err)
	}
	if event == "driver.park_invite.created" && notifier.push != nil && notifier.push.enabled && notifier.push.provider != nil {
		tokens, err := notifier.repository.ListDriverPushTokens(ctx, invite.DriverID)
		if err != nil {
			failures = append(failures, err)
		} else if len(tokens) > 0 {
			err = notifier.push.provider.SendToTokens(ctx, tokens, Notification{Title: "Приглашение в таксопарк", Body: fmt.Sprintf("Таксопарк «%s» приглашает вас присоединиться. Откройте приглашение, чтобы принять или отклонить его.", invite.ToTaxiParkName), Data: map[string]string{"type": event, "invite_id": invite.ID.String()}})
			if err != nil {
				failures = append(failures, fmt.Errorf("push driver invite: %w", err))
			}
		}
	}
	if invite.Status != domain.DriverParkInvitePending {
		// The recipient sees only the destination park, not another park's private context.
		payload := struct {
			InviteID   uuid.UUID                     `json:"invite_id"`
			DriverID   uuid.UUID                     `json:"driver_id"`
			DriverName string                        `json:"driver_name"`
			Status     domain.DriverParkInviteStatus `json:"status"`
			Message    string                        `json:"message"`
		}{invite.ID, invite.DriverID, invite.DriverName, invite.Status, "Статус приглашения водителя изменён"}
		parkMessages := map[uuid.UUID]string{
			invite.ToTaxiParkID: fmt.Sprintf("Статус приглашения водителя %s изменён", invite.DriverName),
		}
		if invite.Status == domain.DriverParkInviteAccepted {
			parkMessages[invite.ToTaxiParkID] = fmt.Sprintf("Водитель %s принял приглашение и присоединился к вашему таксопарку", invite.DriverName)
			if invite.FromTaxiParkID != nil {
				parkMessages[*invite.FromTaxiParkID] = fmt.Sprintf("Водитель %s перешёл в таксопарк «%s»", invite.DriverName, invite.ToTaxiParkName)
			}
		} else if invite.Status == domain.DriverParkInviteDeclined {
			parkMessages[invite.ToTaxiParkID] = fmt.Sprintf("Водитель %s отклонил приглашение", invite.DriverName)
		} else if invite.Status == domain.DriverParkInviteCancelled {
			parkMessages[invite.ToTaxiParkID] = fmt.Sprintf("Приглашение водителя %s отменено", invite.DriverName)
		}
		for park, message := range parkMessages {
			payload.Message = message
			recipients, err := notifier.repository.ListParkInviteRecipients(ctx, park)
			if err != nil {
				failures = append(failures, err)
				continue
			}
			for _, recipient := range recipients {
				if err := notifier.realtime.SendToUser(ctx, recipient, event, payload); err != nil {
					failures = append(failures, err)
				}
			}
		}
	}
	return errors.Join(failures...)
}
