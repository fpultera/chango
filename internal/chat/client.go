package chat

import (
	"context"
	"encoding/json"
	"net/http"
	"chango/internal/data"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{ CheckOrigin: func(r *http.Request) bool { return true } }

type ChatMessage struct {
	Type        string   `json:"type"`
	Content     string   `json:"content"`
	ChannelID   string   `json:"channel_id"`
	RecipientID string   `json:"recipient_id,omitempty"`
	IsPrivate   bool     `json:"is_private"`
	Users       []string `json:"users,omitempty"`
}

type Client struct {
	Conn      *websocket.Conn
	Hub       *Hub
	Store     *data.PostgresStorage
	ChannelID string
}

func HandleWS(hub *Hub, store *data.PostgresStorage) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		channel := r.URL.Query().Get("channel")
		user := r.URL.Query().Get("user")
		
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil { return }

		client := &Client{Conn: conn, Hub: hub, Store: store, ChannelID: channel}
		
		// Registrar usuario en el mapa global
		hub.Clients.Store(user, channel)
		
		// Enviar lista de usuarios actual inmediatamente al nuevo cliente
		users := hub.GetOnlineUsers()
		msg, _ := json.Marshal(ChatMessage{Type: "users_update", Users: users})
		conn.WriteMessage(websocket.TextMessage, msg)

		// Notificar a todos los demás clientes
		client.broadcastUserUpdate()

		go client.readFromRedis()
		go client.readFromWS(user)
	}
}

func (c *Client) broadcastUserUpdate() {
	users := c.Hub.GetOnlineUsers()
	msg, _ := json.Marshal(ChatMessage{Type: "users_update", Users: users})
	c.Hub.Publish(context.Background(), msg)
}

func (c *Client) readFromRedis() {
	ctx := context.Background()
	pubsub := c.Hub.Subscribe(ctx)
	defer pubsub.Close()
	for {
		msg, err := pubsub.ReceiveMessage(ctx)
		if err != nil { return }
		c.Conn.WriteMessage(websocket.TextMessage, []byte(msg.Payload))
	}
}

func (c *Client) readFromWS(userName string) {
	defer func() {
		c.Hub.Clients.Delete(userName)
		c.broadcastUserUpdate()
		c.Conn.Close()
	}()

	for {
		_, msgBytes, err := c.Conn.ReadMessage()
		if err != nil { break }

		var chatMsg ChatMessage
		if err := json.Unmarshal(msgBytes, &chatMsg); err != nil { continue }

		// Persistir mensajes de chat (públicos y DMs)
		if chatMsg.Type == "chat" || chatMsg.Type == "" {
			c.Store.SaveMessage(context.Background(), data.Message{
				Content:     chatMsg.Content,
				ChannelID:   chatMsg.ChannelID,
				IsPrivate:   chatMsg.IsPrivate,
				RecipientID: chatMsg.RecipientID,
			})
		}
		
		// Retransmisión vía Redis
		c.Hub.Publish(context.Background(), msgBytes)
	}
}