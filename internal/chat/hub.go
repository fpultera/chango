package chat

import (
	"context"
	"sync"

	"github.com/redis/go-redis/v9"
)

type Hub struct {
	RedisClient *redis.Client
	// Clients guarda: Key(string)=Username, Value(string)=ChannelID
	Clients sync.Map
}

func (h *Hub) Publish(ctx context.Context, payload []byte) {
	h.RedisClient.Publish(ctx, "chango_chat", payload)
}

func (h *Hub) Subscribe(ctx context.Context) *redis.PubSub {
	return h.RedisClient.Subscribe(ctx, "chango_chat")
}

// GetOnlineUsers extrae los nombres del mapa para enviarlos al frontend
func (h *Hub) GetOnlineUsers() []string {
	var users []string
	h.Clients.Range(func(key, value any) bool {
		users = append(users, key.(string))
		return true
	})
	return users
}