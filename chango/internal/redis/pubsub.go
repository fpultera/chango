package redisbus

import (
	"context"

	"github.com/redis/go-redis/v9"
)

type Bus struct {
	rdb *redis.Client
}

func New(addr string) *Bus {
	rdb := redis.NewClient(&redis.Options{
		Addr: addr,
	})
	return &Bus{rdb: rdb}
}

func (b *Bus) Publish(ctx context.Context, room string, payload []byte) error {
	return b.rdb.Publish(ctx, room, payload).Err()
}

func (b *Bus) Subscribe(ctx context.Context, room string) *redis.PubSub {
	return b.rdb.Subscribe(ctx, room)
}