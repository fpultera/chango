FROM golang:1.25-alpine

# Instalar dependencias necesarias
RUN apk add --no-cache gcc musl-dev

WORKDIR /app

# Copiar archivos de dependencias
COPY go.mod go.sum ./
RUN go mod download

# Copiar el resto del código
COPY . .

# Crear la estructura de carpetas para estáticos y asegurar permisos
RUN mkdir -p ui/static/avatars

# Compilar la aplicación
RUN go build -o main ./cmd/server/main.go

EXPOSE 8080

CMD ["./main"]