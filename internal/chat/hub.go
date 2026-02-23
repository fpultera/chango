// internal/chat/hub.go
package chat

import (
	"context"
	"github.com/redis/go-redis/v9"
)

type Hub struct {
	RedisClient *redis.Client
}

func (h *Hub) Publish(ctx context.Context, msg string) {
	h.RedisClient.Publish(ctx, "chango_chat", msg)
}

func (h *Hub) Subscribe(ctx context.Context) *redis.PubSub {
	return h.RedisClient.Subscribe(ctx, "chango_chat")
}