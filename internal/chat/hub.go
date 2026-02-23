package chat

import (
	"context"
	"github.com/redis/go-redis/v9"
)

type Hub struct {
	RedisClient *redis.Client
}

func (h *Hub) Publish(ctx context.Context, payload []byte) {
	h.RedisClient.Publish(ctx, "chango_chat", payload)
}

func (h *Hub) Subscribe(ctx context.Context) *redis.PubSub {
	return h.RedisClient.Subscribe(ctx, "chango_chat")
}