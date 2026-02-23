package data

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Message representa la estructura de un mensaje en la DB
type Message struct {
	ID        int       `json:"id"`
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"created_at"`
}

// PostgresStorage maneja la conexión a la base de datos
type PostgresStorage struct {
	Pool *pgxpool.Pool
}

// NewPostgresPool crea una nueva instancia de conexión
func NewPostgresPool(ctx context.Context, dsn string) (*PostgresStorage, error) {
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		return nil, fmt.Errorf("no se pudo conectar a postgres: %w", err)
	}

	// Verificar conexión
	if err := pool.Ping(ctx); err != nil {
		return nil, err
	}

	return &PostgresStorage{Pool: pool}, nil
}

// SaveMessage inserta un nuevo mensaje
func (s *PostgresStorage) SaveMessage(ctx context.Context, content string) error {
	query := `INSERT INTO messages (content) VALUES ($1)`
	_, err := s.Pool.Exec(ctx, query, content)
	return err
}

// GetHistory recupera los últimos 50 mensajes
func (s *PostgresStorage) GetHistory(ctx context.Context, limit int) ([]Message, error) {
	query := `SELECT id, content, created_at FROM messages ORDER BY created_at DESC LIMIT $1`
	
	rows, err := s.Pool.Query(ctx, query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var messages []Message
	for rows.Next() {
		var m Message
		if err := rows.Scan(&m.ID, &m.Content, &m.CreatedAt); err != nil {
			return nil, err
		}
		messages = append(messages, m)
	}
	
	return messages, nil
}