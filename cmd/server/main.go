package main

import (
	"context"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"chango/internal/chat"
	"chango/internal/data"
	"github.com/golang-jwt/jwt/v5"
	"github.com/redis/go-redis/v9"
	"golang.org/x/crypto/bcrypt"
)

var jwtSecret = []byte("secreto_chango_2026")

func main() {
	ctx := context.Background()
	store, _ := data.NewPostgresPool(ctx, "postgres://chango_user:chango_password@postgres:5432/chango_app")
	rdb := redis.NewClient(&redis.Options{Addr: "redis:6379"})
	hub := &chat.Hub{RedisClient: rdb}

	// Crear carpeta de avatars si no existe
	os.MkdirAll("./ui/static/avatars", os.ModePerm)

	// Servir UI y estáticos
	fs := http.FileServer(http.Dir("./ui/static"))
	http.Handle("/static/", http.StripPrefix("/static/", fs))
	
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "./ui/index.html")
	})

	// API Auth
	http.HandleFunc("/api/register", func(w http.ResponseWriter, r *http.Request) {
		var creds struct{ Username, Password string }
		json.NewDecoder(r.Body).Decode(&creds)
		store.CreateUser(context.Background(), creds.Username, creds.Password)
		w.WriteHeader(201)
	})

	http.HandleFunc("/api/login", func(w http.ResponseWriter, r *http.Request) {
		var creds struct{ Username, Password string }
		json.NewDecoder(r.Body).Decode(&creds)
		u, _ := store.GetUserByUsername(context.Background(), creds.Username)
		if u == nil || bcrypt.CompareHashAndPassword([]byte(u.Password), []byte(creds.Password)) != nil {
			http.Error(w, "Unauthorized", 401); return
		}
		token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{"username": u.Username, "exp": time.Now().Add(time.Hour * 24).Unix()})
		tokenString, _ := token.SignedString(jwtSecret)
		json.NewEncoder(w).Encode(map[string]string{"token": tokenString, "username": u.Username, "avatar_url": u.AvatarURL})
	})

	// API Canales e Historial
	http.HandleFunc("/api/channels", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" {
			channels, _ := store.GetChannels(context.Background())
			json.NewEncoder(w).Encode(channels)
		} else {
			var body struct{ Name string }
			json.NewDecoder(r.Body).Decode(&body)
			store.CreateChannel(context.Background(), body.Name)
			rdb.Publish(context.Background(), "chango_chat", `{"type":"channels_update"}`)
		}
	})

	http.HandleFunc("/api/history", func(w http.ResponseWriter, r *http.Request) {
		channel := r.URL.Query().Get("channel")
		messages, _ := store.GetHistory(context.Background(), channel)
		json.NewEncoder(w).Encode(messages)
	})

	http.HandleFunc("/api/history/private", func(w http.ResponseWriter, r *http.Request) {
		u1, u2 := r.URL.Query().Get("user1"), r.URL.Query().Get("user2")
		messages, _ := store.GetPrivateHistory(context.Background(), u1, u2)
		json.NewEncoder(w).Encode(messages)
	})

	// Subida de Avatar
	http.HandleFunc("/api/upload-avatar", func(w http.ResponseWriter, r *http.Request) {
		file, header, err := r.FormFile("avatar")
		if err != nil { http.Error(w, "Error", 400); return }
		defer file.Close()

		username := r.FormValue("username")
		ext := filepath.Ext(header.Filename)
		filename := username + "_" + time.Now().Format("150405") + ext
		path := filepath.Join("./ui/static/avatars", filename)

		dst, _ := os.Create(path)
		defer dst.Close()
		io.Copy(dst, file)

		avatarURL := "/static/avatars/" + filename
		store.UpdateUserAvatar(context.Background(), username, avatarURL)
		json.NewEncoder(w).Encode(map[string]string{"url": avatarURL})
	})

	http.HandleFunc("/ws", chat.HandleWS(hub, store))

	log.Println("🚀 Chango Perfiles en :8080")
	http.ListenAndServe(":8080", nil)
}