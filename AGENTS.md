# AGENTS.md

## Service Overview

**Name:** Instant File Share\
**Stack:** Go (Golang), Echo v5\
**Type:** HTTP API service\
**Purpose:**\
Сервис мгновенного обмена файлами по короткой ссылке. Работает по
принципу URL shortener, но вместо редиректа возвращает загруженный файл.

------------------------------------------------------------------------

## Core Concept

1.  Пользователь загружает файл.
2.  Сервис сохраняет файл во временное хранилище.
3.  Генерируется короткий токен (slug).
4.  Пользователь получает ссылку вида:

https://files.example.com/{token}

5.  По этой ссылке:
    -   файл скачивается
    -   либо отображается (если тип поддерживается браузером)

Файлы могут:
- удаляться после первого скачивания (one-time mode)
- удаляться по TTL
- удаляться вручную

------------------------------------------------------------------------

## Architecture

### Layers

/main.go (entrypoint)
/handlers (HTTP handlers + UI handlers)
/service (business logic)
/storage (file storage abstraction)
/repository (data access abstraction)
/models (DTOs)
/ui (SPA static files embedded in binary)

### Responsibilities

#### HTTP Layer

-   Echo v5 router
-   Middleware (logging, recovery, rate limit, CORS)
-   Input validation
-   Mapping request ↔ DTO
-   OpenTelemetry HTTP instrumentation

#### Service Layer

-   Бизнес-логика
-   Генерация токенов
-   TTL управление
-   One-time download логика

#### Storage Layer

Поддерживаемые реализации:
- Local FS (default)
- Object Storage (S3-compatible, future)

#### Repository Layer

Метаданные файлов:
- PostgreSQL (default)
- DuckDB (future)

------------------------------------------------------------------------

## Dependencies

### Core
- `github.com/labstack/echo/v5` — HTTP фреймворк
- `github.com/google/uuid` — генерация UUID
- `github.com/jackc/pgx/v5` — PostgreSQL драйвер
- `github.com/rs/zerolog` — логирование
- `golang.org/x/time/rate` — rate limiting

### Frontend (Vite + React SPA)
- `react` ^19.2.4 — UI framework
- `react-dom` ^19.2.4 — React DOM
- `react-router-dom` ^7 — routing

### Infrastructure
- `github.com/k3env/koanf-extensions` — конфигурация (YAML + env)
- `github.com/k3env/otlp-init` — OpenTelemetry (logs, metrics, tracing)

------------------------------------------------------------------------

## API Endpoints

### Upload File

POST /api/v1/files\
Content-Type: multipart/form-data

Form fields:
- file (required)
- ttl_seconds (optional)
- one_time (optional, bool)

Response:

{ "token": "abc123",
"expires_at": "2026-03-03T12:00:00Z" }

------------------------------------------------------------------------

### Get File Metadata

GET /api/v1/files/:token

Response:

{
  "token": "abc123",
  "filename": "document.pdf",
  "size": 1048576,
  "content_type": "application/pdf",
  "one_time": false,
  "expires_at": "2026-03-03T12:00:00Z",
  "status": "active"
}

Behavior:
- 404 if not found
- 410 if expired

------------------------------------------------------------------------

### Download File

GET /:token/download

Behavior:
- 404 if not found
- 410 if expired
- 403 if already downloaded in one-time mode

------------------------------------------------------------------------

### Delete File

DELETE /api/v1/files/:token

------------------------------------------------------------------------

## Token Generation

Requirements:
- Cryptographically secure
- URL-safe
- Length: 6--12 chars

Recommended:
- crypto/rand
- Alphanumeric charset (a-z, A-Z, 0-9)

Token collision handling:
- Retry generation up to 5 times

------------------------------------------------------------------------

## Security Requirements

-   Max file size limit (configurable)
-   Rate limiting per IP (configurable enable/disable)
-   MIME-type detection (no trust to client)
-   Optional password protection (future)
-   Virus scanning hook (future)
-   Do not expose internal storage paths

------------------------------------------------------------------------

## Configuration

Configuration is loaded from `config.yaml` using koanf with environment variable override support.

```yaml
app:
  port: 8080
storage:
  max_file_size: 100          # MB
  default_ttl: 24h
  type: local
  local:
    path: ./data
database:
  type: postgres
  dsn: postgres://user:pass@localhost/drop
rate_limit:
  enabled: true
  rpm: 60
  burst: 10
log:
  environment: dev
  otlp:
    host: otlp.collect.example.com
    port: 443
    insecure: false
```

### Config Loading

- Default: load from `config.yaml`
- Environment variables override: prefix with `DROP_` (e.g., `DROP_APP_PORT=8081`)
- Priority: env vars > yaml file > defaults

### Config Structures

```go
type Config struct {
    App       *AppConfig
    Storage   *StorageConfig
    Database  *DatabaseConfig
    RateLimit *RateLimitConfig
    Log       *LogConfig
}

type AppConfig struct {
    Port int
}

type StorageConfig struct {
    MaxFileSizeMB int64
    DefaultTTL    time.Duration
    Type          string  // "local" or "object"
    Local         *LocalStorageConfig
    Object        *ObjectStorageConfig  // future
}

type LocalStorageConfig struct {
    Path string
}

type DatabaseConfig struct {
    Type string  // "postgres" or "duckdb"
    DSN  string
}

type RateLimitConfig struct {
    RPM     int
    Burst   int
    Enabled bool
}

type LogConfig struct {
    Environment string
    OTLP        *OTLPConfig
}

type OTLPConfig struct {
    Host     string
    Port     int
    Insecure bool
}
```

------------------------------------------------------------------------

## File Lifecycle

States:
- active
- expired
- deleted
- consumed (for one-time files)

Cleanup strategies:
- Lazy expiration (check on access)

------------------------------------------------------------------------

## Metadata Model

``` go
type File struct {
    ID          uuid.UUID
    Token       string
    Filename    string
    ContentType string
    Size        int64
    StoragePath string
    OneTime     bool
    ExpiresAt   *time.Time
    Downloaded  bool
    CreatedAt   time.Time
}
```

------------------------------------------------------------------------

## Middleware

Required:
- Structured logging (zerolog + OTLP)
- Recovery
- Rate limiting (configurable enable/disable)
- CORS
- Body size limit

OpenTelemetry:
- HTTP metrics
- HTTP tracing

------------------------------------------------------------------------

## Observability

Endpoints:
-   /health
-   /ready

OpenTelemetry:
- Logs: zerolog with OTLP exporter
- Metrics: HTTP request metrics
- Tracing: HTTP spans

Metrics (future):
- uploads_total
- downloads_total
- active_files
- expired_files
- storage_used_bytes

------------------------------------------------------------------------

## Error Handling

Use unified error response:

{ "error": "file_not_found", "message": "File does not exist or has
expired" }

Never expose:
- stack traces
- internal storage paths
- DB errors

------------------------------------------------------------------------

## Performance Considerations

-   Stream upload directly to storage (no full buffering in memory)
-   Stream download
-   Avoid loading full file into RAM
-   Support large files safely

------------------------------------------------------------------------

## Future Extensions

-   Authenticated uploads
-   Temporary authenticated links
-   Chunked uploads
-   Object storage support (S3-compatible)
-   QR code generation
-   File preview
-   Encryption at rest
-   End-to-end encrypted mode
-   DuckDB support

------------------------------------------------------------------------

## Non-Goals

-   Long-term storage
-   Versioning
-   Collaborative editing
-   Public file indexing

------------------------------------------------------------------------

## Development Rules

-   No global mutable state
-   Dependency injection
-   Context propagation everywhere
-   No business logic in handlers
-   Use interfaces for storage & repository
-   80%+ unit test coverage for service layer
-   Configuration via YAML with env override

### Frontend Development Rules

-   Vite + React 19 + React Router v7
-   TypeScript
-   CSS variables for theming
-   Neutral minimalistic styling (preserved from original design)
-   Progress bar for large file uploads
-   Clipboard API for copy functionality
-   Responsive design for mobile and desktop

### Frontend Directory Structure

```
front/
├── src/
│   ├── api/
│   │   └── types.ts        # TypeScript types from backend DTOs
│   ├── pages/
│   │   ├── UploadPage.tsx  # Upload form with TTL/one-time options
│   │   ├── DownloadPage.tsx # File info and download
│   │   └── NotFoundPage.tsx # 404 page
│   ├── utils.ts            # formatBytes, formatDuration, etc.
│   ├── index.css           # Global styles
│   ├── main.tsx            # React Router setup
│   └── App.tsx             # Layout
├── vite.config.ts          # Vite config with API proxy
└── package.json

ui/
├── index.html              # SPA entry (copied from front/dist)
├── assets/                 # JS, CSS bundles
├── favicon.svg
└── icons.svg
```

### SPA Integration

The SPA is embedded in the Go binary using `//go:embed`:

```go
//go:embed ui
var embeddedFS embed.FS

func main() {
    staticFiles, _ := fs.Sub(embeddedFS, "ui")
    e.StaticFS("/", staticFiles)
}
```

### Build Process

1. `cd front && npm install && npm run build`
2. Copy `front/dist/*` to `ui/`
3. `go build` embeds `ui/` directory

CI/CD (drone.yml):
```yaml
- name: frontend-build
  image: node:22-alpine
  commands:
    - cd front
    - npm install
    - npm run build
    - cp -r dist/* ../ui/
```

### Dev Server Proxy

Vite proxies API requests to Go backend:

```typescript
// vite.config.ts
server: {
  proxy: {
    '/api': 'http://localhost:8080',
    '/:token/download': 'http://localhost:8080',
  }
}
```

------------------------------------------------------------------------

## Deployment

-   Single binary
-   Docker-ready
-   Stateless app layer
-   Horizontal scaling supported
-   Shared storage required for multi-instance mode

------------------------------------------------------------------------

## Summary

Это lightweight, stateless HTTP-сервис для мгновенного обмена файлами по
короткой ссылке с поддержкой TTL и one-time загрузки, построенный на
Go + Echo v5, с чётким разделением слоёв, YAML-конфигурацией через koanf,
интегрированным OpenTelemetry (логи, метрики, трассировка) и готовностью к
горизонтальному масштабированию.
