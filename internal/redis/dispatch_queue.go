package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	goredis "github.com/redis/go-redis/v9"

	"github.com/develoop/taxi-platform/internal/dispatch"
)

const dispatchQueueKey = "dispatch:queue"
const dispatchTimeoutQueueKey = "dispatch:timeouts"

type DispatchQueue struct {
	client *goredis.Client
}

func NewDispatchQueue(client *goredis.Client) *DispatchQueue {
	return &DispatchQueue{client: client}
}

func (queue *DispatchQueue) Publish(ctx context.Context, task dispatch.DispatchTask) error {
	payload, err := json.Marshal(task)
	if err != nil {
		return fmt.Errorf("marshal dispatch task: %w", err)
	}
	if err := queue.client.LPush(ctx, dispatchQueueKey, payload).Err(); err != nil {
		return fmt.Errorf("publish dispatch task: %w", err)
	}
	return nil
}

func (queue *DispatchQueue) Consume(ctx context.Context, timeout time.Duration) (dispatch.DispatchTask, bool, error) {
	values, err := queue.client.BRPop(ctx, timeout, dispatchQueueKey).Result()
	if err == goredis.Nil {
		return dispatch.DispatchTask{}, false, nil
	}
	if err != nil {
		return dispatch.DispatchTask{}, false, fmt.Errorf("consume dispatch task: %w", err)
	}
	if len(values) != 2 {
		return dispatch.DispatchTask{}, false, fmt.Errorf("unexpected dispatch queue response length: %d", len(values))
	}

	var task dispatch.DispatchTask
	if err := json.Unmarshal([]byte(values[1]), &task); err != nil {
		return dispatch.DispatchTask{}, false, fmt.Errorf("unmarshal dispatch task: %w", err)
	}
	return task, true, nil
}

type DispatchTimeoutQueue struct {
	client *goredis.Client
}

func NewDispatchTimeoutQueue(client *goredis.Client) *DispatchTimeoutQueue {
	return &DispatchTimeoutQueue{client: client}
}

func (queue *DispatchTimeoutQueue) Schedule(ctx context.Context, task dispatch.DispatchTask, runAt time.Time) error {
	payload, err := json.Marshal(task)
	if err != nil {
		return fmt.Errorf("marshal dispatch timeout task: %w", err)
	}
	if err := queue.client.ZAdd(ctx, dispatchTimeoutQueueKey, goredis.Z{
		Score:  float64(runAt.UnixMilli()),
		Member: payload,
	}).Err(); err != nil {
		return fmt.Errorf("schedule dispatch timeout task: %w", err)
	}
	return nil
}

func (queue *DispatchTimeoutQueue) Due(ctx context.Context, now time.Time, limit int) ([]dispatch.DispatchTask, error) {
	values, err := queue.client.ZRangeByScore(ctx, dispatchTimeoutQueueKey, &goredis.ZRangeBy{
		Min:    "-inf",
		Max:    fmt.Sprintf("%d", now.UnixMilli()),
		Offset: 0,
		Count:  int64(limit),
	}).Result()
	if err != nil {
		return nil, fmt.Errorf("load due dispatch timeout tasks: %w", err)
	}

	tasks := make([]dispatch.DispatchTask, 0, len(values))
	for _, value := range values {
		var task dispatch.DispatchTask
		if err := json.Unmarshal([]byte(value), &task); err != nil {
			return nil, fmt.Errorf("unmarshal due dispatch timeout task: %w", err)
		}
		tasks = append(tasks, task)
	}
	return tasks, nil
}

func (queue *DispatchTimeoutQueue) Remove(ctx context.Context, task dispatch.DispatchTask) error {
	payload, err := json.Marshal(task)
	if err != nil {
		return fmt.Errorf("marshal dispatch timeout task for removal: %w", err)
	}
	if err := queue.client.ZRem(ctx, dispatchTimeoutQueueKey, payload).Err(); err != nil {
		return fmt.Errorf("remove dispatch timeout task: %w", err)
	}
	return nil
}
