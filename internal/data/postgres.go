package data

import (
	"context"
	"time"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"
)

type Message struct {
	ID          int       `json:"id"`
	Content     string    `json:"content"`
	ChannelID   string    `json:"channel_id"`
	IsPrivate   bool      `json:"is_private"`
	RecipientID string    `json:"recipient_id"`
	CreatedAt   time.Time `json:"created_at"`
}

type User struct {
	ID        int    `json:"id"`
	Username  string `json:"username"`
	Password  string `json:"-"`
	AvatarURL string `json:"avatar_url"`
}

type Channel struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

type PostgresStorage struct {
	Pool *pgxpool.Pool
}

func NewPostgresPool(ctx context.Context, dsn string) (*PostgresStorage, error) {
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil { return nil, err }
	return &PostgresStorage{Pool: pool}, nil
}

func (s *PostgresStorage) CreateUser(ctx context.Context, username, password string) error {
	hashed, _ := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	_, err := s.Pool.Exec(ctx, "INSERT INTO users (username, password, avatar_url) VALUES ($1, $2, $3)", 
		username, string(hashed), "/static/avatars/default.png")
	return err
}

func (s *PostgresStorage) GetUserByUsername(ctx context.Context, username string) (*User, error) {
	var u User
	err := s.Pool.QueryRow(ctx, "SELECT id, username, password, avatar_url FROM users WHERE username = $1", username).
		Scan(&u.ID, &u.Username, &u.Password, &u.AvatarURL)
	if err != nil { return nil, err }
	return &u, nil
}

func (s *PostgresStorage) UpdateUserAvatar(ctx context.Context, username, avatarURL string) error {
	_, err := s.Pool.Exec(ctx, "UPDATE users SET avatar_url = $1 WHERE username = $2", avatarURL, username)
	return err
}

func (s *PostgresStorage) SaveMessage(ctx context.Context, m Message) error {
	query := `INSERT INTO messages (content, channel_id, is_private, recipient_id) VALUES ($1, $2, $3, $4)`
	_, err := s.Pool.Exec(ctx, query, m.Content, m.ChannelID, m.IsPrivate, m.RecipientID)
	return err
}

func (s *PostgresStorage) GetHistory(ctx context.Context, channelID string) ([]Message, error) {
	query := `SELECT content, created_at FROM messages WHERE channel_id = $1 AND is_private = false ORDER BY created_at ASC LIMIT 50`
	rows, err := s.Pool.Query(ctx, query, channelID)
	if err != nil { return nil, err }
	defer rows.Close()
	var messages []Message
	for rows.Next() {
		var m Message
		rows.Scan(&m.Content, &m.CreatedAt)
		messages = append(messages, m)
	}
	return messages, nil
}

func (s *PostgresStorage) GetPrivateHistory(ctx context.Context, u1, u2 string) ([]Message, error) {
	query := `SELECT content, created_at FROM messages 
              WHERE is_private = true 
              AND ((channel_id = $1 AND recipient_id = $2) OR (channel_id = $2 AND recipient_id = $1))
              ORDER BY created_at ASC LIMIT 50`
	rows, err := s.Pool.Query(ctx, query, u1, u2)
	if err != nil { return nil, err }
	defer rows.Close()
	var messages []Message
	for rows.Next() {
		var m Message
		rows.Scan(&m.Content, &m.CreatedAt)
		messages = append(messages, m)
	}
	return messages, nil
}

func (s *PostgresStorage) CreateChannel(ctx context.Context, name string) error {
	_, err := s.Pool.Exec(ctx, "INSERT INTO channels (name) VALUES ($1) ON CONFLICT DO NOTHING", name)
	return err
}

func (s *PostgresStorage) GetChannels(ctx context.Context) ([]Channel, error) {
	rows, err := s.Pool.Query(ctx, "SELECT id, name FROM channels ORDER BY name ASC")
	if err != nil { return nil, err }
	defer rows.Close()
	var channels []Channel
	for rows.Next() {
		var c Channel
		rows.Scan(&c.ID, &c.Name)
		channels = append(channels, c)
	}
	return channels, nil
}