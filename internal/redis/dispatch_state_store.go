package redis

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	goredis "github.com/redis/go-redis/v9"
)

type DispatchStateStore struct {
	client *goredis.Client
}

func NewDispatchStateStore(client *goredis.Client) *DispatchStateStore {
	return &DispatchStateStore{client: client}
}

func (store *DispatchStateStore) BeginDispatch(ctx context.Context, orderID uuid.UUID, ttl time.Duration) (bool, error) {
	started, err := store.client.SetNX(ctx, dispatchingKey(orderID), "1", ttl).Result()
	if err != nil {
		return false, fmt.Errorf("begin dispatch state: %w", err)
	}
	return started, nil
}

func (store *DispatchStateStore) FinishDispatch(ctx context.Context, orderID uuid.UUID) error {
	if err := store.client.Del(ctx, dispatchingKey(orderID), activeOfferKey(orderID)).Err(); err != nil {
		return fmt.Errorf("finish dispatch state: %w", err)
	}
	return nil
}

func (store *DispatchStateStore) MarkActiveOffer(ctx context.Context, orderID uuid.UUID, ttl time.Duration) error {
	if err := store.client.Set(ctx, activeOfferKey(orderID), "1", ttl).Err(); err != nil {
		return fmt.Errorf("mark active offer: %w", err)
	}
	return nil
}

func (store *DispatchStateStore) MarkAcceptedDriver(ctx context.Context, orderID uuid.UUID, driverID uuid.UUID, ttl time.Duration) error {
	if err := store.client.Set(ctx, acceptedDriverKey(orderID), driverID.String(), ttl).Err(); err != nil {
		return fmt.Errorf("mark accepted driver: %w", err)
	}
	return nil
}

func dispatchingKey(orderID uuid.UUID) string {
	return fmt.Sprintf("order:%s:dispatching", orderID)
}

func activeOfferKey(orderID uuid.UUID) string {
	return fmt.Sprintf("order:%s:active_offer", orderID)
}

func acceptedDriverKey(orderID uuid.UUID) string {
	return fmt.Sprintf("order:%s:accepted_driver", orderID)
}
