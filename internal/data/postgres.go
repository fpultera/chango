package data

import (
	"context"
	"time"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Message struct {
	ID        int       `json:"id"`
	Content   string    `json:"content"`
	ChannelID string    `json:"channel_id"`
	CreatedAt time.Time `json:"created_at"`
}

type PostgresStorage struct {
	Pool *pgxpool.Pool
}

func NewPostgresPool(ctx context.Context, dsn string) (*PostgresStorage, error) {
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		return nil, err
	}
	return &PostgresStorage{Pool: pool}, nil
}

func (s *PostgresStorage) SaveMessage(ctx context.Context, content string, channelID string) error {
	query := `INSERT INTO messages (content, channel_id) VALUES ($1, $2)`
	_, err := s.Pool.Exec(ctx, query, content, channelID)
	return err
}

func (s *PostgresStorage) GetHistory(ctx context.Context, channelID string) ([]Message, error) {
	// Traemos los últimos 50 mensajes ordenados por fecha
	query := `SELECT id, content, channel_id, created_at FROM messages 
              WHERE channel_id = $1 ORDER BY created_at ASC LIMIT 50`
	
	rows, err := s.Pool.Query(ctx, query, channelID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var messages []Message = []Message{} // Inicializamos vacío para que el JSON no sea null
	for rows.Next() {
		var m Message
		if err := rows.Scan(&m.ID, &m.Content, &m.ChannelID, &m.CreatedAt); err != nil {
			return nil, err
		}
		messages = append(messages, m)
	}
	return messages, nil
}