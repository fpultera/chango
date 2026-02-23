# Paso 1: Compilación
FROM golang:1.25-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
# Compilamos el binario desde la ruta donde está el main.go
RUN go build -o chango-server ./cmd/server/main.go

# Paso 2: Imagen ligera de ejecución
FROM alpine:latest
WORKDIR /root/
# Copiamos el binario y la carpeta ui (el frontend)
COPY --from=builder /app/chango-server .
COPY --from=builder /app/ui ./ui

EXPOSE 8080
CMD ["./chango-server"]