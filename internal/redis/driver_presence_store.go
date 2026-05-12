package redis

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	goredis "github.com/redis/go-redis/v9"
)

type DriverPresenceStore struct {
	client *goredis.Client
}

func NewDriverPresenceStore(client *goredis.Client) *DriverPresenceStore {
	return &DriverPresenceStore{client: client}
}

func (store *DriverPresenceStore) MarkOnline(ctx context.Context, cityID uuid.UUID, driverID uuid.UUID, ttl time.Duration) error {
	pipe := store.client.TxPipeline()
	pipe.SAdd(ctx, onlineDriversKey(cityID), driverID.String())
	pipe.Set(ctx, driverOnlineKey(driverID), cityID.String(), ttl)
	pipe.Expire(ctx, onlineDriversKey(cityID), ttl)

	if _, err := pipe.Exec(ctx); err != nil {
		return fmt.Errorf("mark driver online in redis: %w", err)
	}

	return nil
}

func (store *DriverPresenceStore) MarkOffline(ctx context.Context, cityID uuid.UUID, driverID uuid.UUID) error {
	pipe := store.client.TxPipeline()
	pipe.SRem(ctx, onlineDriversKey(cityID), driverID.String())
	pipe.Del(ctx, driverOnlineKey(driverID))

	if _, err := pipe.Exec(ctx); err != nil {
		return fmt.Errorf("mark driver offline in redis: %w", err)
	}

	return nil
}

func (store *DriverPresenceStore) IsOnline(ctx context.Context, driverID uuid.UUID) (bool, error) {
	exists, err := store.client.Exists(ctx, driverOnlineKey(driverID)).Result()
	if err != nil {
		return false, fmt.Errorf("check driver online state in redis: %w", err)
	}

	return exists > 0, nil
}

func onlineDriversKey(cityID uuid.UUID) string {
	return fmt.Sprintf("drivers:city:%s:online", cityID)
}

func driverOnlineKey(driverID uuid.UUID) string {
	return fmt.Sprintf("driver:online:%s", driverID)
}
