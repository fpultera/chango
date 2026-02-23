package main

import (
	"context"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"chango/internal/chat"
	"chango/internal/data"
	"github.com/golang-jwt/jwt/v5"
	"github.com/redis/go-redis/v9"
	"golang.org/x/crypto/bcrypt"
)

var jwtSecret = []byte("secreto_chango_2026")
var usernameRegex = regexp.MustCompile(`^[a-zA-Z0-9_]+$`)

func main() {
	ctx := context.Background()
	store, _ := data.NewPostgresPool(ctx, "postgres://chango_user:chango_password@postgres:5432/chango_app")
	rdb := redis.NewClient(&redis.Options{Addr: "redis:6379"})
	hub := &chat.Hub{RedisClient: rdb}

	os.MkdirAll("./ui/static/avatars", os.ModePerm)
	fs := http.FileServer(http.Dir("./ui/static"))
	http.Handle("/static/", http.StripPrefix("/static/", fs))
	
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "./ui/index.html")
	})

	http.HandleFunc("/api/register", func(w http.ResponseWriter, r *http.Request) {
		var creds struct{ Username, Password string }
		json.NewDecoder(r.Body).Decode(&creds)
		u := strings.TrimSpace(creds.Username)
		if u == "" || creds.Password == "" || !usernameRegex.MatchString(u) {
			http.Error(w, "Datos inválidos", 400); return
		}
		store.CreateUser(context.Background(), u, creds.Password)
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

	// ENDPOINT CANALES CORREGIDO
	http.HandleFunc("/api/channels", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case "GET":
			channels, _ := store.GetChannels(context.Background())
			json.NewEncoder(w).Encode(channels)
		case "POST":
			var body struct{ Name, Owner string }
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				http.Error(w, "Bad Request", 400); return
			}
			name := strings.TrimSpace(body.Name)
			if name != "" && body.Owner != "" {
				store.CreateChannel(context.Background(), name, body.Owner)
				rdb.Publish(context.Background(), "chango_chat", `{"type":"channels_update"}`)
				w.WriteHeader(201)
			}
		case "DELETE":
			name, owner := r.URL.Query().Get("name"), r.URL.Query().Get("owner")
			if name != "" && owner != "" {
				store.DeleteChannel(context.Background(), name, owner)
				rdb.Publish(context.Background(), "chango_chat", `{"type":"channels_update"}`)
				w.WriteHeader(200)
			}
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

	http.HandleFunc("/api/upload-avatar", func(w http.ResponseWriter, r *http.Request) {
		file, header, _ := r.FormFile("avatar")
		defer file.Close()
		username := r.FormValue("username")
		filename := username + "_" + time.Now().Format("150405") + filepath.Ext(header.Filename)
		path := filepath.Join("./ui/static/avatars", filename)
		dst, _ := os.Create(path)
		defer dst.Close()
		io.Copy(dst, file)
		avatarURL := "/static/avatars/" + filename
		store.UpdateUserAvatar(context.Background(), username, avatarURL)
		json.NewEncoder(w).Encode(map[string]string{"url": avatarURL})
	})

	http.HandleFunc("/ws", chat.HandleWS(hub, store))

	log.Println("🚀 Chango Pro corriendo en :8080")
	http.ListenAndServe(":8080", nil)
}