package chat

import (
	"sync"

	"github.com/gorilla/websocket"
)

type Client struct {
	ID   string
	Conn *websocket.Conn
	Room string

	Send chan []byte // <- cola de escritura (clave para evitar race)
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

	if c, ok := h.clients[id]; ok {
		close(c.Send) // cerramos writePump
		delete(h.clients, id)
	}
}

// ⚠️ YA NO ESCRIBE AL WEBSOCKET.
// Solo encola mensajes.
func (h *Hub) BroadcastLocal(msg []byte, room string) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	for _, c := range h.clients {
		if c.Room != room {
			continue
		}

		select {
		case c.Send <- msg:
		default:
			// cliente lento → lo descartamos
			close(c.Send)
			delete(h.clients, c.ID)
		}
	}
}