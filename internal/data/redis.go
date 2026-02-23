package data

import (
	"context"
	"github.com/redis/go-redis/v9"
)

func NewRedisClient(addr string) (*redis.Client, error) {
	rdb := redis.NewClient(&redis.Options{
		Addr: addr,
	})

	// Verificar si Redis está arriba
	status := rdb.Ping(context.Background())
	if status.Err() != nil {
		return nil, status.Err()
	}

	return rdb, nil
}