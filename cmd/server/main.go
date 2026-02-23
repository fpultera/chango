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

// Regex para validar que el usuario no tenga caracteres extraños
var usernameRegex = regexp.MustCompile(`^[a-zA-Z0-9_]+$`)

func main() {
	ctx := context.Background()

	// 1. Conexión a Base de Datos
	store, err := data.NewPostgresPool(ctx, "postgres://chango_user:chango_password@postgres:5432/chango_app")
	if err != nil {
		log.Fatal("No se pudo conectar a Postgres:", err)
	}

	// 2. Conexión a Redis
	rdb := redis.NewClient(&redis.Options{Addr: "redis:6379"})
	hub := &chat.Hub{RedisClient: rdb}

	// 3. Asegurar carpetas para Avatares persistentes
	avatarPath := filepath.Join("ui", "static", "avatars")
	if _, err := os.Stat(avatarPath); os.IsNotExist(err) {
		os.MkdirAll(avatarPath, os.ModePerm)
	}

	// 4. Servidor de Archivos Estáticos (Avatares y CSS/JS)
	fs := http.FileServer(http.Dir("./ui/static"))
	http.Handle("/static/", http.StripPrefix("/static/", fs))

	// 5. Ruta Principal (Frontend)
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "./ui/index.html")
	})

	// --- API AUTH ---

	http.HandleFunc("/api/register", func(w http.ResponseWriter, r *http.Request) {
		var creds struct{ Username, Password string }
		if err := json.NewDecoder(r.Body).Decode(&creds); err != nil {
			http.Error(w, "Datos inválidos", http.StatusBadRequest)
			return
		}

		u := strings.TrimSpace(creds.Username)
		p := strings.TrimSpace(creds.Password)

		if u == "" || p == "" {
			http.Error(w, "Campos obligatorios", http.StatusBadRequest)
			return
		}

		if !usernameRegex.MatchString(u) {
			http.Error(w, "El usuario solo permite letras, números y guiones bajos", http.StatusBadRequest)
			return
		}

		err := store.CreateUser(context.Background(), u, p)
		if err != nil {
			http.Error(w, "El usuario ya existe", http.StatusConflict)
			return
		}
		w.WriteHeader(http.StatusCreated)
	})

	http.HandleFunc("/api/login", func(w http.ResponseWriter, r *http.Request) {
		var creds struct{ Username, Password string }
		json.NewDecoder(r.Body).Decode(&creds)

		u := strings.TrimSpace(creds.Username)
		if u == "" {
			http.Error(w, "Usuario requerido", http.StatusBadRequest)
			return
		}

		dbUser, _ := store.GetUserByUsername(context.Background(), u)
		if dbUser == nil || bcrypt.CompareHashAndPassword([]byte(dbUser.Password), []byte(creds.Password)) != nil {
			http.Error(w, "Credenciales incorrectas", http.StatusUnauthorized)
			return
		}

		token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
			"username": dbUser.Username,
			"exp":      time.Now().Add(time.Hour * 24).Unix(),
		})
		tokenString, _ := token.SignedString(jwtSecret)

		json.NewEncoder(w).Encode(map[string]string{
			"token":      tokenString,
			"username":   dbUser.Username,
			"avatar_url": dbUser.AvatarURL,
		})
	})

	// --- API CANALES ---

	http.HandleFunc("/api/channels", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case "GET":
			channels, _ := store.GetChannels(context.Background())
			json.NewEncoder(w).Encode(channels)
		case "POST":
			var body struct{ Name, Owner string }
			json.NewDecoder(r.Body).Decode(&body)
			name := strings.TrimSpace(body.Name)
			if name != "" && body.Owner != "" {
				store.CreateChannel(context.Background(), name, body.Owner)
				rdb.Publish(context.Background(), "chango_chat", `{"type":"channels_update"}`)
			}
		case "DELETE":
			name := r.URL.Query().Get("name")
			owner := r.URL.Query().Get("owner")
			if name != "" && owner != "" {
				store.DeleteChannel(context.Background(), name, owner)
				rdb.Publish(context.Background(), "chango_chat", `{"type":"channels_update"}`)
				w.WriteHeader(http.StatusOK)
			}
		}
	})

	// --- API HISTORIAL ---

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

	// --- API PERFIL (AVATAR) ---

	http.HandleFunc("/api/upload-avatar", func(w http.ResponseWriter, r *http.Request) {
		file, header, err := r.FormFile("avatar")
		if err != nil {
			http.Error(w, "Error al recibir archivo", 400)
			return
		}
		defer file.Close()

		username := r.FormValue("username")
		// Creamos un nombre de archivo único para evitar cache del navegador
		filename := username + "_" + time.Now().Format("150405") + filepath.Ext(header.Filename)
		path := filepath.Join(avatarPath, filename)

		dst, err := os.Create(path)
		if err != nil {
			http.Error(w, "Error al guardar archivo", 500)
			return
		}
		defer dst.Close()
		io.Copy(dst, file)

		avatarURL := "/static/avatars/" + filename
		store.UpdateUserAvatar(context.Background(), username, avatarURL)
		json.NewEncoder(w).Encode(map[string]string{"url": avatarURL})
	})

	// --- WEBSOCKET ---
	http.HandleFunc("/ws", chat.HandleWS(hub, store))

	log.Println("🚀 Chango Pro corriendo en :8080 con persistencia de volumen")
	log.Fatal(http.ListenAndServe(":8080", nil))
}