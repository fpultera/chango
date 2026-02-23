# Chango 🐵 - Chat en Tiempo Real

Chango es una aplicación de chat moderna que soporta canales públicos, mensajes directos (DMs) con ventanas emergentes, perfiles de usuario con avatares persistentes y notificaciones en tiempo real.

## 🚀 Tecnologías
* **Backend:** Go (Golang) con WebSockets.
* **Frontend:** HTML5, CSS3 y JavaScript (Vanilla).
* **Base de Datos:** PostgreSQL.
* **Mensajería/Cache:** Redis.
* **Infraestructura:** Docker & Docker Compose.

---

## 🛠️ Instalación y Configuración

### 1. Clonar el repositorio
```bash
git clone [https://github.com/fpultera/chango.git](https://github.com/fpultera/chango.git)
cd chango
```
2. Preparar el entorno
Asegúrate de tener instalado Docker y Docker Compose.

Crea la carpeta para los avatares para asegurar que los volúmenes funcionen correctamente:

```Bash
mkdir -p ui/static/avatars
```
3. Levantar los servicios
Este comando descargará las imágenes, compilará la app de Go y levantará las bases de datos:

```Bash
docker compose up --build -d
```
🗄️ Configuración de la Base de Datos
Una vez que los contenedores estén corriendo, debes ejecutar las siguientes queries para preparar las tablas. Puedes hacerlo ejecutando este comando en tu terminal:

```Bash
docker exec -it chango_db psql -U chango_user -d chango_app -c "
-- Tabla de Usuarios
CREATE TABLE IF NOT EXISTS users (
    id SERIAL PRIMARY KEY,
    username TEXT UNIQUE NOT NULL,
    password TEXT NOT NULL,
    avatar_url TEXT DEFAULT '/static/avatars/default.png'
);

-- Tabla de Canales
CREATE TABLE IF NOT EXISTS channels (
    id SERIAL PRIMARY KEY,
    name TEXT UNIQUE NOT NULL,
    owner TEXT
);

-- Tabla de Mensajes
CREATE TABLE IF NOT EXISTS messages (
    id SERIAL PRIMARY KEY,
    content TEXT NOT NULL,
    channel_id TEXT NOT NULL,
    is_private BOOLEAN DEFAULT FALSE,
    recipient_id TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
```

-- Canal por defecto
```bash
INSERT INTO channels (name, owner) VALUES ('general', 'system') ON CONFLICT DO NOTHING;
```
📋 Funcionalidades Implementadas
Auth: Registro y Login con validación de caracteres (Regex) y contraseñas encriptadas (bcrypt).

Perfil: Subida de avatares personalizados con persistencia en volumen de Docker.

Canales: Creación y borrado de canales (solo por el dueño).

DMs Inteligentes: Los mensajes directos no interrumpen la navegación; activan un indicador de "no leído" y se abren en ventanas mini al hacer clic.

Real-time: Actualización instantánea de lista de usuarios online y mensajes vía Redis Pub/Sub.

📁 Estructura del Proyecto
/cmd/server: Punto de entrada de la aplicación Go.

/internal/chat: Lógica de WebSockets, Hub y manejo de clientes.

/internal/data: Modelos y persistencia en PostgreSQL.

/ui: Archivos estáticos (HTML, CSS, JS) y carpeta de avatares.

🛑 Detener la aplicación
Para apagar los servicios manteniendo los datos:

```Bash
docker compose stop
```
Para borrar los contenedores (los volúmenes persistirán):

```Bash
docker compose down
```
---

### Notas adicionales para ti:

1.  **El archivo `.gitkeep`:** Te recomiendo poner un archivo vacío llamado `.gitkeep` dentro de `ui/static/avatars/` y subirlo al repo. Esto asegura que la carpeta exista en Git pero el contenido (las fotos) se ignore gracias al `.gitignore`.
2.  **Uso de las Queries:** Las queries que puse en el README consolidan **todo** lo que fuimos agregando (la columna `owner`, el `avatar_url`, etc.). Si alguien corre eso en una DB limpia, el proyecto sale andando de una.