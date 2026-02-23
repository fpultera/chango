package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"

	"github.com/fpultera/chango/internal/chat"
	redisbus "github.com/fpultera/chango/internal/redis"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true }, // dev only
}

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// --- Config from ENV (works for docker/k8s/local) ---
	redisAddr := getenv("REDIS_ADDR", "localhost:6379")
	httpAddr := getenv("HTTP_ADDR", ":8080")

	log.Println("starting chango gateway")
	log.Println("redis:", redisAddr)
	log.Println("http:", httpAddr)

	// --- Core components ---
	hub := chat.NewHub()
	bus := redisbus.New(redisAddr)

	// --- HTTP WebSocket endpoint ---
	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		room := r.URL.Query().Get("room")
		user := r.URL.Query().Get("user")

		if room == "" || user == "" {
			http.Error(w, "room and user required", http.StatusBadRequest)
			return
		}

		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Println("upgrade error:", err)
			return
		}

		client := &chat.Client{
			ID:   uuid.NewString(),
			Conn: conn,
			Room: room,
		}

		hub.Add(client)
		log.Printf("client %s joined room %s", client.ID, room)

		// Subscribe THIS pod to the room if not already
		go subscribeRoom(ctx, bus, hub, room)

		// Read loop
		go func() {
			defer func() {
				hub.Remove(client.ID)
				conn.Close()
				log.Printf("client %s disconnected", client.ID)
			}()

			for {
				_, data, err := conn.ReadMessage()
				if err != nil {
					return
				}

				msg := chat.Message{
					ID:        uuid.NewString(),
					Room:      room,
					User:      user,
					Content:   string(data),
					CreatedAt: time.Now().UTC(),
				}

				encoded, err := json.Marshal(msg)
				if err != nil {
					continue
				}

				// Publish to cluster
				if err := bus.Publish(ctx, room, encoded); err != nil {
					log.Println("redis publish error:", err)
				}

				// Broadcast locally (fast-path)
				hub.BroadcastLocal(encoded, room)
			}
		}()
	})

	server := &http.Server{
		Addr: httpAddr,
	}

	go func() {
		log.Println("gateway listening on", httpAddr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal(err)
		}
	}()

	// --- Graceful shutdown ---
	waitForShutdown()
	log.Println("shutting down...")

	cancel()

	ctxShutdown, _ := context.WithTimeout(context.Background(), 5*time.Second)
	server.Shutdown(ctxShutdown)
}

// subscribeRoom listens Redis and rebroadcasts to local clients.
func subscribeRoom(ctx context.Context, bus *redisbus.Bus, hub *chat.Hub, room string) {
	sub := bus.Subscribe(ctx, room)
	ch := sub.Channel()

	log.Println("subscribed to room:", room)

	for msg := range ch {
		hub.BroadcastLocal([]byte(msg.Payload), room)
	}
}

// getenv with fallback
func getenv(key, fallback string) string {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	return v
}

// graceful shutdown helper
func waitForShutdown() {
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	<-sig
}