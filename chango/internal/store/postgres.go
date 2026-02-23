package store

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/fpultera/chango/internal/chat"
)

type Store struct {
	db *pgxpool.Pool
}

func New(ctx context.Context, url string) (*Store, error) {
	pool, err := pgxpool.New(ctx, url)
	if err != nil {
		return nil, err
	}
	return &Store{db: pool}, nil
}

func (s *Store) SaveMessage(ctx context.Context, m chat.Message) error {
	_, err := s.db.Exec(ctx, `
		insert into messages (id, room, username, content, created_at)
		values ($1,$2,$3,$4,$5)
	`,
		m.ID, m.Room, m.User, m.Content, m.CreatedAt,
	)
	return err
}