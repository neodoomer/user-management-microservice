# User Management Microservice

A production-ready Go microservice for user management with JWT authentication, built with Clean Architecture principles.

## Tech Stack

- **Go** (Echo framework)
- **PostgreSQL** with [sqlc](https://sqlc.dev/) for type-safe queries
- **golang-migrate** for database migrations
- **JWT** (HS256) authentication with bcrypt password hashing
- **HAProxy** as a reverse proxy
- **Docker Compose** for orchestration

## Architecture

```
┌──────────┐     ┌──────────┐     ┌──────────┐     ┌──────────┐
│  HAProxy │────▶│ Next.js  │     │  Go API  │────▶│ Postgres │
│  :80     │────▶│ :3000    │     │  :8080   │     │  :5432   │
└──────────┘     └──────────┘     └──────────┘     └──────────┘
   /api/v1/* ──────────────────────▶ (direct)
   /* ──────────▶ (frontend)
```

Three-layer Clean Architecture:

```
Handler (HTTP) → Service (Business Logic) → Repository (sqlc Queries)
```

## API Endpoints

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| GET | `/health` | No | Health check |
| POST | `/api/v1/auth/signup` | No | Register a new user |
| POST | `/api/v1/auth/signin` | No | Sign in, returns JWT |
| POST | `/api/v1/auth/forgot-password` | No | Request password reset token |
| POST | `/api/v1/auth/reset-password` | No | Reset password with token |
| POST | `/api/v1/users` | Admin | Create a user with role |
| GET | `/api/v1/users?page=1&limit=10` | Admin | List users (paginated) |

## Quick Start

### Prerequisites

- Docker and Docker Compose
- The [frontend repo](https://github.com/neodoomer/user-management-frontend) cloned as a sibling directory:

```
Go Projects/
├── user-management-microservice/   # this repo
└── user-management-frontend/       # frontend repo
```

### Run the full stack

```bash
docker compose up --build -d
```

This starts 4 containers:

| Service | Port | Description |
|---------|------|-------------|
| postgres | 5432 | PostgreSQL 16 |
| app | 8080 | Go API server |
| frontend | 3000 | Next.js app |
| haproxy | **80** | Reverse proxy (entry point) |

Open **http://localhost** in your browser.

### Run locally (without Docker)

```bash
cp .env.example .env
# Start a local Postgres, then:
make migrate-up
make run
```

## Testing

Unit tests cover all layers (hasher, JWT, services, handlers, middleware):

```bash
go test -v ./...
```

Tests also run automatically during `docker build` — the image won't build if tests fail.

## Project Structure

```
cmd/server/main.go          # Entrypoint, DI, graceful shutdown
internal/
  apperr/errors.go          # Domain error types
  config/config.go           # Env-based configuration
  db/                        # sqlc-generated code
  dto/                       # Request/response structs
  handler/                   # HTTP handlers, routes, validator, error handler
  middleware/                 # JWT auth + role-based authorization
  service/                   # Business logic (auth, user)
pkg/
  hasher/                    # bcrypt password hasher
  token/                     # JWT manager
db/
  migrations/                # SQL migration files
  queries/                   # sqlc query definitions
haproxy/haproxy.cfg          # HAProxy configuration
api.http                     # API documentation / test file
```

## Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `SERVER_PORT` | 8080 | HTTP server port |
| `DB_HOST` | localhost | PostgreSQL host |
| `DB_PORT` | 5432 | PostgreSQL port |
| `DB_USER` | postgres | Database user |
| `DB_PASSWORD` | postgres | Database password |
| `DB_NAME` | usermanagement | Database name |
| `DB_SSLMODE` | disable | PostgreSQL SSL mode |
| `JWT_SECRET` | (required) | HMAC secret for signing JWTs |
| `JWT_EXPIRY` | 15m | JWT token lifetime |

## License

MIT
