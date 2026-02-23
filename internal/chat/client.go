package chat

import (
	"context"
	"log"
	"net/http"

	"chango/internal/data" // Ajusta según tu go.mod
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

// Client representa la conexión de un usuario
type Client struct {
	Conn  *websocket.Conn
	Hub   *Hub
	Store *data.PostgresStorage
}

// HandleWS es el constructor de la conexión
func HandleWS(hub *Hub, store *data.PostgresStorage) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Printf("Error upgrade: %v", err)
			return
		}

		client := &Client{
			Conn:  conn,
			Hub:   hub,
			Store: store,
		}

		// Lanzar procesos de escucha
		go client.readFromRedis()
		go client.readFromWS()
	}
}

// Lee de Redis y envía al navegador del usuario
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

// Lee del navegador del usuario y envía a Redis + Postgres
func (c *Client) readFromWS() {
	defer c.Conn.Close()

	for {
		_, msg, err := c.Conn.ReadMessage()
		if err != nil {
			break
		}

		// Persistir en Postgres
		ctx := context.Background()
		if err := c.Store.SaveMessage(ctx, string(msg)); err != nil {
			log.Printf("Error persistiendo: %v", err)
		}

		// Publicar en Redis
		c.Hub.Publish(ctx, string(msg))
	}
}