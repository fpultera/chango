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
	AvatarURL   string   `json:"avatar_url,omitempty"`
	Sender      string   `json:"sender,omitempty"`
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
		conn, _ := upgrader.Upgrade(w, r, nil)

		client := &Client{Conn: conn, Hub: hub, Store: store, ChannelID: channel}
		hub.Clients.Store(user, channel)
		
		initialUsers := hub.GetOnlineUsers()
		initialMsg, _ := json.Marshal(ChatMessage{Type: "users_update", Users: initialUsers})
		conn.WriteMessage(websocket.TextMessage, initialMsg)
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
		json.Unmarshal(msgBytes, &chatMsg)

		if chatMsg.Type == "chat" || chatMsg.Type == "" {
			u, err := c.Store.GetUserByUsername(context.Background(), userName)
			if err == nil {
				chatMsg.AvatarURL = u.AvatarURL
				chatMsg.Sender = userName
			}

			// GUARDADO CON FORMATO PARA EL JOIN
			fullContent := userName + ": " + chatMsg.Content
			c.Store.SaveMessage(context.Background(), data.Message{
				Content:     fullContent,
				ChannelID:   chatMsg.ChannelID,
				IsPrivate:   chatMsg.IsPrivate,
				RecipientID: chatMsg.RecipientID,
			})
			
			msgBytes, _ = json.Marshal(chatMsg)
		}

		c.Hub.Publish(context.Background(), msgBytes)
	}
}