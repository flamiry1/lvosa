# Lvosa

A web app for learning Slovenian: lessons, exercises, a learner dashboard, an admin panel and an AI tutor.

The project is in early development. The backend is being built first; the frontend comes later.

## Stack

- **Backend:** Go, gin, pgx, go-redis, bcrypt
- **Storage:** PostgreSQL 17 (data), Redis 8 (sessions)
- **Migrations:** goose
- **Frontend (planned):** Nuxt, Vue 3, TypeScript
- **Infrastructure:** Docker Compose, Caddy (planned)

## Getting started

Requirements: Go 1.26+, Docker, [goose](https://github.com/pressly/goose).

1. Create the env file and set passwords:

   ```bash
   cp .env.example .env
   ```

   Fill in `POSTGRES_PASSWORD` and `REDIS_PASSWORD`.

2. Start Postgres and Redis:

   ```bash
   docker compose up -d
   ```

3. Load the env file into your shell and apply migrations:

   ```bash
   set -a; source .env; set +a
   goose -dir backend/migrations postgres "$DB_DSN" up
   ```

4. Run the server:

   ```bash
   go run ./backend/cmd/server
   ```

   The API listens on `:8080` by default.

## Tests

Unit tests run without any infrastructure:

```bash
go test ./...
```

Repository tests run against real Postgres and Redis. They use a separate database `lvosa_test` and Redis database `1`, and are skipped when `TEST_DB_DSN` / `TEST_REDIS_DSN` are not set.

One-time setup for the test database:

```bash
docker compose exec postgres psql -U lvosa -d postgres -c "CREATE DATABASE lvosa_test;"
goose -dir backend/migrations postgres "$TEST_DB_DSN" up
```

Then:

```bash
set -a; source .env; set +a
go test -race -count=1 ./...
```

## Configuration

| Variable | Required | Default | Description |
|---|---|---|---|
| `DB_DSN` | yes | — | Postgres connection string |
| `REDIS_DSN` | yes | — | Redis connection string |
| `PORT` | no | `8080` | HTTP port |
| `APP_ENV` | no | `dev` | `dev` or `prod` |
| `LOG_LEVEL` | no | `debug` | `debug`, `info`, `warn`, `error` |
| `READ_TIMEOUT`, `WRITE_TIMEOUT`, `IDLE_TIMEOUT`, `SHUTDOWN_TIMEOUT` | no | `10s` | HTTP server timeouts |
| `TEST_DB_DSN`, `TEST_REDIS_DSN` | for tests | — | test Postgres and Redis |

## Project structure

```
backend/
├── cmd/server/          entry point, wiring, graceful shutdown
├── internal/
│   ├── config/          configuration from environment
│   ├── handlers/        HTTP layer
│   ├── service/         business logic (items, auth)
│   ├── models/          domain types and errors
│   └── store/
│       ├── postgres/    Postgres repositories
│       └── redisstore/  Redis session storage
└── migrations/          SQL migrations
frontend/                Nuxt app (planned)
```

## Roadmap

- [x] Layered HTTP API with tests
- [x] Postgres storage and migrations
- [x] Redis connection
- [x] Authentication: registration, login, sessions, roles
- [ ] Lessons and exercises
- [ ] Frontend
- [ ] AI tutor
- [ ] CI and deployment
