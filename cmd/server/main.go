package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"chango/internal/chat"
	"chango/internal/data"
)

func main() {
	// 1. Configuración de contexto y tiempos
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// 2. Conexión a PostgreSQL (Capa de Datos)
	
	// dsn := "postgres://chango_user:chango_password@localhost:5432/chango_app"
	rdb, err := data.NewRedisClient("redis:6379")
	store, err := data.NewPostgresPool(ctx, dsn)
	if err != nil {
		log.Fatalf("Error Postgres: %v", err)
	}
	defer store.Pool.Close()

	// 3. Conexión a Redis (Capa de Mensajería)
	rdb, err := data.NewRedisClient("localhost:6379")
	if err != nil {
		log.Fatalf("Error Redis: %v", err)
	}

	// 4. Inicializar el Hub del Chat
	hub := &chat.Hub{RedisClient: rdb}

	// 5. Definición de Rutas
	// Servir el Frontend (Ajusta la ruta según dónde estés ejecutando el binario)
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "./ui/index.html")
	})

	// Endpoint de WebSockets usando el nuevo paquete chat
	http.HandleFunc("/ws", chat.HandleWS(hub, store))

	// 6. Arrancar Servidor
	port := ":8080"
	fmt.Printf("🚀 Chango operativo en http://localhost%s\n", port)
	
	server := &http.Server{
		Addr:         port,
		WriteTimeout: 15 * time.Second,
		ReadTimeout:  15 * time.Second,
	}

	log.Fatal(server.ListenAndServe())
}