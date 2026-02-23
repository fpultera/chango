package chat

import (
	"log"
	"time"

	"github.com/gorilla/websocket"
)

const writeWait = 10 * time.Second

// ESTE es el único lugar donde se escribe al socket.
func (c *Client) WritePump() {
	defer func() {
		c.Conn.Close()
		log.Println("writePump closed for", c.ID)
	}()

	for msg := range c.Send {
		c.Conn.SetWriteDeadline(time.Now().Add(writeWait))

		if err := c.Conn.WriteMessage(websocket.TextMessage, msg); err != nil {
			return
		}
	}
}