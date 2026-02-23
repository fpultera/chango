package chat

import "time"

type Message struct {
	ID        string    `json:"id"`
	Room      string    `json:"room"`
	User      string    `json:"user"`
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"created_at"`
}