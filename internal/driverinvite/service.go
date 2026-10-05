package driverinvite

import (
	"context"
	"fmt"
	"github.com/google/uuid"
	"github.com/kishert-lab/taxi-platform/internal/domain"
	"go.uber.org/zap"
	"time"
)

type Driver struct {
	ID     uuid.UUID
	UserID uuid.UUID
	ParkID *uuid.UUID
	Name   string
	Status domain.DriverStatus
}
type Park struct {
	ID     uuid.UUID
	CityID uuid.UUID
	Name   string
	Active bool
}

// Transaction exposes persistence operations; all invite decisions belong to Service.
type Transaction interface {
	ActorPark(context.Context, uuid.UUID) (Park, error)
	Park(context.Context, uuid.UUID) (Park, error)
	LockDriver(context.Context, uuid.UUID, string) (Driver, error)
	LockInviteDriver(context.Context, uuid.UUID) (Driver, error)
	Invite(context.Context, uuid.UUID) (domain.DriverParkInvite, error)
	Pending(context.Context, uuid.UUID, uuid.UUID) (*domain.DriverParkInvite, error)
	Insert(context.Context, domain.DriverParkInvite) error
	Save(context.Context, domain.DriverParkInvite) error
	Expire(context.Context, uuid.UUID, time.Time) error
	HasOrders(context.Context, uuid.UUID) (bool, error)
	Transfer(context.Context, Driver, Park, domain.DriverParkInvite, time.Time) error
	CancelOtherPending(context.Context, uuid.UUID, uuid.UUID, time.Time) error
	Audit(context.Context, uuid.UUID, string, domain.DriverParkInvite) error
}
type Repository interface {
	Transact(context.Context, func(Transaction) error) error
	List(context.Context, uuid.UUID, bool) ([]domain.DriverParkInvite, error)
	SavePushToken(context.Context, uuid.UUID, string, string) error
	DeletePushToken(context.Context, uuid.UUID, string) error
}
type Notifier interface {
	Notify(context.Context, domain.DriverParkInvite, string) error
}
type Service struct {
	repository Repository
	notifier   Notifier
	ttl        time.Duration
	logger     *zap.Logger
}

func NewService(repository Repository, notifier Notifier, ttl time.Duration, logger *zap.Logger) *Service {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &Service{repository: repository, notifier: notifier, ttl: ttl, logger: logger}
}
func (service *Service) Create(ctx context.Context, actor uuid.UUID, rawPhone string) (domain.DriverParkInvite, error) {
	phone, err := domain.NormalizePhone(rawPhone)
	if err != nil {
		return domain.DriverParkInvite{}, fmt.Errorf("normalize invited driver phone: %w", err)
	}
	if service.ttl <= 0 {
		return domain.DriverParkInvite{}, fmt.Errorf("invite lifetime must be positive")
	}
	var result domain.DriverParkInvite
	created := false
	err = service.repository.Transact(ctx, func(transaction Transaction) error {
		park, err := transaction.ActorPark(ctx, actor)
		if err != nil {
			return err
		}
		if !park.Active {
			return domain.ErrParkInviteInactivePark
		}
		driver, err := transaction.LockDriver(ctx, uuid.Nil, phone)
		if err != nil {
			return err
		}
		if driver.ParkID != nil && *driver.ParkID == park.ID {
			return domain.ErrParkInviteSamePark
		}
		now := time.Now().UTC()
		if err := transaction.Expire(ctx, driver.ID, now); err != nil {
			return err
		}
		existing, err := transaction.Pending(ctx, driver.ID, park.ID)
		if err != nil {
			return err
		}
		if existing != nil {
			result = *existing
			return nil
		}
		result = domain.DriverParkInvite{ID: uuid.New(), DriverID: driver.ID, FromTaxiParkID: driver.ParkID, ToTaxiParkID: park.ID, ToTaxiParkName: park.Name, DriverName: driver.Name, Status: domain.DriverParkInvitePending, CreatedByUserID: actor, CreatedAt: now, ExpiresAt: now.Add(service.ttl)}
		if driver.ParkID != nil {
			previous, err := transaction.Park(ctx, *driver.ParkID)
			if err != nil {
				return err
			}
			result.FromTaxiParkName = previous.Name
		}
		if err := transaction.Insert(ctx, result); err != nil {
			return err
		}
		created = true
		return transaction.Audit(ctx, actor, "driver_park_invite.created", result)
	})
	if err != nil {
		return domain.DriverParkInvite{}, fmt.Errorf("create driver park invite: %w", err)
	}
	if created {
		service.notify(ctx, result, "driver.park_invite.created")
	}
	return result, nil
}
func (service *Service) List(ctx context.Context, actor uuid.UUID, park bool) ([]domain.DriverParkInvite, error) {
	invites, err := service.repository.List(ctx, actor, park)
	if err != nil {
		return nil, fmt.Errorf("list driver park invites: %w", err)
	}
	now := time.Now().UTC()
	for index := range invites {
		if invites[index].Status == domain.DriverParkInvitePending && !now.Before(invites[index].ExpiresAt) {
			invites[index].Status = domain.DriverParkInviteExpired
		}
		// A park must not discover a competing park through an invitation response.
		if park {
			invites[index].FromTaxiParkID = nil
			invites[index].FromTaxiParkName = ""
		}
	}
	return invites, nil
}
func (service *Service) Respond(ctx context.Context, actor, id uuid.UUID, action domain.DriverParkInviteStatus) (domain.DriverParkInvite, error) {
	var result domain.DriverParkInvite
	var businessError error
	err := service.repository.Transact(ctx, func(transaction Transaction) error {
		// All mutations take the driver lock first, including cancellation and creation.
		var driver Driver
		var err error
		if action == domain.DriverParkInviteCancelled {
			driver, err = transaction.LockInviteDriver(ctx, id)
		} else {
			driver, err = transaction.LockDriver(ctx, actor, "")
		}
		if err != nil {
			return err
		}
		result, err = transaction.Invite(ctx, id)
		if err != nil {
			return err
		}
		if result.DriverID != driver.ID {
			return domain.ErrParkInviteNotFound
		}
		if action == domain.DriverParkInviteCancelled {
			park, err := transaction.ActorPark(ctx, actor)
			if err != nil {
				return err
			}
			if result.ToTaxiParkID != park.ID {
				return domain.ErrParkInviteNotFound
			}
		} else if driver.UserID != actor {
			return domain.ErrParkInviteNotFound
		}
		now := time.Now().UTC()
		if err := result.EnsurePending(now); err != nil {
			if result.Status == domain.DriverParkInvitePending {
				result.Status = domain.DriverParkInviteExpired
				if saveErr := transaction.Save(ctx, result); saveErr != nil {
					return saveErr
				}
				businessError = err
				return nil
			}
			return err
		}
		switch action {
		case domain.DriverParkInviteAccepted:
			park, err := transaction.Park(ctx, result.ToTaxiParkID)
			if err != nil {
				return err
			}
			if !park.Active {
				return domain.ErrParkInviteInactivePark
			}
			hasOrders, err := transaction.HasOrders(ctx, driver.ID)
			if err != nil {
				return err
			}
			if err := result.EnsureTransfer(driver.ParkID, driver.Status, hasOrders); err != nil {
				return err
			}
			result.AcceptedAt = &now
			if err := transaction.Transfer(ctx, driver, park, result, now); err != nil {
				return err
			}
			if err := transaction.CancelOtherPending(ctx, driver.ID, id, now); err != nil {
				return err
			}
		case domain.DriverParkInviteDeclined:
			result.DeclinedAt = &now
		case domain.DriverParkInviteCancelled:
			result.CancelledAt = &now
			result.CancellationReason = "cancelled_by_park"
		default:
			return domain.ErrParkInviteForbidden
		}
		result.Status = action
		if err := transaction.Save(ctx, result); err != nil {
			return err
		}
		if err := transaction.Audit(ctx, actor, "driver_park_invite."+string(action), result); err != nil {
			return err
		}
		if action == domain.DriverParkInviteAccepted {
			return transaction.Audit(ctx, actor, "driver.transferred_to_park", result)
		}
		return nil
	})
	if err != nil {
		return domain.DriverParkInvite{}, fmt.Errorf("respond to driver park invite: %w", err)
	}
	if businessError != nil {
		return domain.DriverParkInvite{}, businessError
	}
	service.notify(ctx, result, "driver.park_invite."+string(action))
	return result, nil
}
func (service *Service) notify(ctx context.Context, invite domain.DriverParkInvite, event string) {
	service.logger.Info("driver park invite changed", zap.String("operation", event), zap.String("driver_id", invite.DriverID.String()), zap.String("invite_id", invite.ID.String()))
	if service.notifier != nil {
		if err := service.notifier.Notify(ctx, invite, event); err != nil {
			service.logger.Error("deliver driver park invite notification", zap.String("operation", event), zap.String("invite_id", invite.ID.String()), zap.Error(err))
		}
	}
}
func (service *Service) SavePushToken(ctx context.Context, actor uuid.UUID, token, platform string) error {
	if len(token) < 1 || len(token) > 4096 {
		return domain.ErrInvalidPushToken
	}
	platform = domain.NormalizePushPlatform(platform)
	if platform != "android" && platform != "ios" && platform != "web" {
		return domain.ErrInvalidPushPlatform
	}
	return service.repository.SavePushToken(ctx, actor, token, platform)
}
func (service *Service) DeletePushToken(ctx context.Context, actor uuid.UUID, token string) error {
	if len(token) < 1 || len(token) > 4096 {
		return domain.ErrInvalidPushToken
	}
	return service.repository.DeletePushToken(ctx, actor, token)
}
