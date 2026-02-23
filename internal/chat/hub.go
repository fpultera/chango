package chat

import (
	"sync"

	"github.com/gorilla/websocket"
)

type Client struct {
	ID   string
	Conn *websocket.Conn
	Room string
}

type Hub struct {
	mu      sync.RWMutex
	clients map[string]*Client
}

func NewHub() *Hub {
	return &Hub{
		clients: make(map[string]*Client),
	}
}

func (h *Hub) Add(c *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.clients[c.ID] = c
}

func (h *Hub) Remove(id string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	delete(h.clients, id)
}

func (h *Hub) BroadcastLocal(msg []byte, room string) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	for _, c := range h.clients {
		if c.Room == room {
			c.Conn.WriteMessage(websocket.TextMessage, msg)
		}
	}
}