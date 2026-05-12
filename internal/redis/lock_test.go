package redis

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	goredis "github.com/redis/go-redis/v9"
)

func TestLockManagerOnlyOneOwnerCanAcquireLock(t *testing.T) {
	t.Parallel()

	server := miniredis.RunT(t)
	client := goredis.NewClient(&goredis.Options{Addr: server.Addr()})
	defer func() {
		_ = client.Close()
	}()

	manager := NewLockManager(client)
	ctx := context.Background()

	firstLock, acquired, err := manager.Acquire(ctx, "order:order-id:accept_lock", 30*time.Second)
	if err != nil {
		t.Fatalf("first acquire: %v", err)
	}
	if !acquired {
		t.Fatal("expected first lock acquire to succeed")
	}

	_, acquired, err = manager.Acquire(ctx, "order:order-id:accept_lock", 30*time.Second)
	if err != nil {
		t.Fatalf("second acquire: %v", err)
	}
	if acquired {
		t.Fatal("expected second lock acquire to fail")
	}

	if err := firstLock.Release(ctx); err != nil {
		t.Fatalf("release lock: %v", err)
	}

	_, acquired, err = manager.Acquire(ctx, "order:order-id:accept_lock", 30*time.Second)
	if err != nil {
		t.Fatalf("third acquire: %v", err)
	}
	if !acquired {
		t.Fatal("expected lock acquire after release to succeed")
	}
}
