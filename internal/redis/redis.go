package redis

import (
	"context"
	"fmt"

	goredis "github.com/redis/go-redis/v9"
)

func NewRedis(host string, port int) *goredis.Client {
	addr := fmt.Sprintf("%s:%d", host, port)

	client := goredis.NewClient(&goredis.Options{
		Addr: addr,
	})

	client.Ping(context.Background())

	return client
}