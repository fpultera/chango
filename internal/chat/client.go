package chat

import (
	"context"
	"encoding/json"
	"net/http"
	"chango/internal/data"
	"github.com/gorilla/websocket"
)

// Definimos el upgrader aquí para que no de error de "undefined"
var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

type ChatMessage struct {
	Content   string `json:"content"`
	ChannelID string `json:"channel_id"`
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
		if channel == "" {
			channel = "general"
		}

		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}

		client := &Client{
			Conn:      conn,
			Hub:       hub,
			Store:     store,
			ChannelID: channel,
		}
		go client.readFromRedis()
		go client.readFromWS()
	}
}

func (c *Client) readFromRedis() {
	ctx := context.Background()
	pubsub := c.Hub.Subscribe(ctx)
	defer pubsub.Close()
	defer c.Conn.Close()

	for {
		msg, err := pubsub.ReceiveMessage(ctx)
		if err != nil {
			return
		}
		
		if err := c.Conn.WriteMessage(websocket.TextMessage, []byte(msg.Payload)); err != nil {
			return
		}
	}
}

func (c *Client) readFromWS() {
	defer c.Conn.Close()

	for {
		_, msgBytes, err := c.Conn.ReadMessage()
		if err != nil {
			break
		}

		var chatMsg ChatMessage
		if err := json.Unmarshal(msgBytes, &chatMsg); err != nil {
			continue
		}

		// Si el mensaje viene sin canal (por compatibilidad), usamos el del cliente
		if chatMsg.ChannelID == "" {
			chatMsg.ChannelID = c.ChannelID
		}

		// Guardar en Postgres
		c.Store.SaveMessage(context.Background(), chatMsg.Content, chatMsg.ChannelID)

		// Publicar en Redis para el resto de los usuarios
		c.Hub.Publish(context.Background(), msgBytes)
	}
}