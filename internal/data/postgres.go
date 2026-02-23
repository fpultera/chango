package data

import (
	"context"
	"strings"
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
	Sender      string    `json:"sender"`      // Campo para el historial
	AvatarURL   string    `json:"avatar_url"`  // Campo para el historial
	CreatedAt   time.Time `json:"created_at"`
}

type User struct {
	ID        int    `json:"id"`
	Username  string `json:"username"`
	Password  string `json:"-"`
	AvatarURL string `json:"avatar_url"`
}

type Channel struct {
	ID    int    `json:"id"`
	Name  string `json:"name"`
	Owner string `json:"owner"`
}

type PostgresStorage struct {
	Pool *pgxpool.Pool
}

func NewPostgresPool(ctx context.Context, dsn string) (*PostgresStorage, error) {
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil { return nil, err }
	return &PostgresStorage{Pool: pool}, nil
}

// --- Usuarios ---
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

// --- Mensajes e Historial con JOIN ---
func (s *PostgresStorage) SaveMessage(ctx context.Context, m Message) error {
	query := `INSERT INTO messages (content, channel_id, is_private, recipient_id) VALUES ($1, $2, $3, $4)`
	_, err := s.Pool.Exec(ctx, query, m.Content, m.ChannelID, m.IsPrivate, m.RecipientID)
	return err
}

func (s *PostgresStorage) GetHistory(ctx context.Context, channelID string) ([]Message, error) {
	query := `
		SELECT m.content, m.created_at, COALESCE(u.username, 'Sistema'), COALESCE(u.avatar_url, '/static/avatars/default.png')
		FROM messages m
		LEFT JOIN users u ON m.content LIKE u.username || ':%'
		WHERE m.channel_id = $1 AND m.is_private = false
		ORDER BY m.created_at ASC LIMIT 50`
	
	rows, err := s.Pool.Query(ctx, query, channelID)
	if err != nil { return nil, err }
	defer rows.Close()

	var messages []Message
	for rows.Next() {
		var content, sender, avatar string
		var createdAt time.Time
		rows.Scan(&content, &createdAt, &sender, &avatar)
		
		cleanContent := content
		if parts := strings.SplitN(content, ": ", 2); len(parts) > 1 {
			cleanContent = parts[1]
		}

		messages = append(messages, Message{
			Content:   cleanContent,
			Sender:    sender,
			AvatarURL: avatar,
			CreatedAt: createdAt,
		})
	}
	return messages, nil
}

func (s *PostgresStorage) GetPrivateHistory(ctx context.Context, u1, u2 string) ([]Message, error) {
	query := `
		SELECT m.content, m.created_at, COALESCE(u.username, 'Sistema'), COALESCE(u.avatar_url, '/static/avatars/default.png')
		FROM messages m
		LEFT JOIN users u ON m.content LIKE u.username || ':%'
		WHERE m.is_private = true 
		AND ((m.channel_id = $1 AND m.recipient_id = $2) OR (m.channel_id = $2 AND m.recipient_id = $1))
		ORDER BY m.created_at ASC LIMIT 50`

	rows, err := s.Pool.Query(ctx, query, u1, u2)
	if err != nil { return nil, err }
	defer rows.Close()

	var messages []Message
	for rows.Next() {
		var content, sender, avatar string
		var createdAt time.Time
		rows.Scan(&content, &createdAt, &sender, &avatar)

		cleanContent := content
		if parts := strings.SplitN(content, ": ", 2); len(parts) > 1 {
			cleanContent = parts[1]
		}

		messages = append(messages, Message{
			Content:   cleanContent,
			Sender:    sender,
			AvatarURL: avatar,
			CreatedAt: createdAt,
		})
	}
	return messages, nil
}

// --- Canales ---
func (s *PostgresStorage) CreateChannel(ctx context.Context, name, owner string) error {
	_, err := s.Pool.Exec(ctx, "INSERT INTO channels (name, owner) VALUES ($1, $2) ON CONFLICT DO NOTHING", name, owner)
	return err
}

func (s *PostgresStorage) GetChannels(ctx context.Context) ([]Channel, error) {
	rows, err := s.Pool.Query(ctx, "SELECT id, name, COALESCE(owner, '') FROM channels ORDER BY name ASC")
	if err != nil { return nil, err }
	defer rows.Close()
	var channels []Channel
	for rows.Next() {
		var c Channel
		rows.Scan(&c.ID, &c.Name, &c.Owner)
		channels = append(channels, c)
	}
	return channels, nil
}

func (s *PostgresStorage) DeleteChannel(ctx context.Context, name, owner string) error {
	_, err := s.Pool.Exec(ctx, "DELETE FROM channels WHERE name = $1 AND owner = $2", name, owner)
	return err
}