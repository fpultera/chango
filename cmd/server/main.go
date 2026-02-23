package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"

	"chango/internal/chat"
	"chango/internal/data"
	"github.com/redis/go-redis/v9"
)

func main() {
	ctx := context.Background()

	// Conexiones (Usando nombres de servicio de Docker)
	store, err := data.NewPostgresPool(ctx, "postgres://chango_user:chango_password@postgres:5432/chango_app")
	if err != nil {
		log.Fatal(err)
	}
	
	rdb := redis.NewClient(&redis.Options{Addr: "redis:6379"})
	hub := &chat.Hub{RedisClient: rdb}

	// Rutas
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "./ui/index.html")
	})

	// NUEVO: Endpoint de historial
	http.HandleFunc("/api/history", func(w http.ResponseWriter, r *http.Request) {
		channel := r.URL.Query().Get("channel")
		if channel == "" { channel = "general" }
		
		messages, err := store.GetHistory(context.Background(), channel)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(messages)
	})

	http.HandleFunc("/ws", chat.HandleWS(hub, store))

	log.Println("🚀 Chango con Historial en :8080")
	http.ListenAndServe(":8080", nil)
}