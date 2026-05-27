package redis

import (
	"context"
	"fmt"
	"time"

	goredis "github.com/redis/go-redis/v9"

	"github.com/kishert-lab/taxi-platform/internal/dispatch"
)

type LockManager struct {
	client *goredis.Client
}

func NewLockManager(client *goredis.Client) *LockManager {
	return &LockManager{client: client}
}

func (manager *LockManager) Acquire(ctx context.Context, key string, ttl time.Duration) (dispatch.Lock, bool, error) {
	token := fmt.Sprintf("%d", time.Now().UnixNano())
	acquired, err := manager.client.SetNX(ctx, key, token, ttl).Result()
	if err != nil {
		return nil, false, fmt.Errorf("acquire redis lock: %w", err)
	}
	if !acquired {
		return nil, false, nil
	}

	return &Lock{client: manager.client, key: key, token: token}, true, nil
}

type Lock struct {
	client *goredis.Client
	key    string
	token  string
}

func (lock *Lock) Release(ctx context.Context) error {
	const script = `
		if redis.call("GET", KEYS[1]) == ARGV[1] then
			return redis.call("DEL", KEYS[1])
		end
		return 0`

	if err := lock.client.Eval(ctx, script, []string{lock.key}, lock.token).Err(); err != nil {
		return fmt.Errorf("release redis lock: %w", err)
	}

	return nil
}
