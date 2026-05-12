package redis

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	goredis "github.com/redis/go-redis/v9"
)

type LocationThrottle struct {
	client *goredis.Client
}

func NewLocationThrottle(client *goredis.Client) *LocationThrottle {
	return &LocationThrottle{client: client}
}

func (throttle *LocationThrottle) AllowLocationUpdate(ctx context.Context, driverID uuid.UUID, interval time.Duration) (bool, error) {
	allowed, err := throttle.client.SetNX(ctx, driverLocationThrottleKey(driverID), "1", interval).Result()
	if err != nil {
		return false, fmt.Errorf("set driver location throttle key: %w", err)
	}
	return allowed, nil
}

func driverLocationThrottleKey(driverID uuid.UUID) string {
	return fmt.Sprintf("driver:%s:location_throttle", driverID)
}
