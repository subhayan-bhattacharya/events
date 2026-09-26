# Chunk 0: Skeleton

Type: task
Status: resolved
Blocked by: 01

## Question

Build the project skeleton.

## Done when

- `git init` done; `.gitignore` present
- `docker-compose.yml` DB name mismatch fixed (`POSTGRES_DB: events` vs `appdb` in `DATABASE_URL`); `api/Dockerfile` exists or the api service is removed for now
- goose and sqlc configured per "Go project layout"
- Makefile targets for DB up, migrate up/down, sqlc generate, test, run all work
- `GET /api/healthz` returns 200 only when Postgres is reachable

## Plan decisions to check

D5, D29

## Answer

Built by the user; verified by the agent (build, `go vet`, live healthz run).

- `api/internal/postgres`: `Connect(ctx, dbURL) (*pgxpool.Pool, error)` parses config, creates the pool, pings on startup.
- `api/cmd/api/main.go`: reads `DB_URL`; `GET /api/healthz` pings via a `Pinger` interface (2s timeout): 200 `{"status":"ok"}`, or 503 `{"status":"unavailable"}` and logs the error. Verified: 200 → stop DB → 503 → start DB → 200.
- Config: `DB_URL` and `POSTGRES_PASSWORD` in the git-ignored root `.env` (template `.env.example`); the Makefile `-include`s and exports it. Compose has only the `db` service (database `events`).
- `make migrate-up` and `make sqlc` fail until the first migration and query file exist (expected; Chunk 1).
- Deferred to deployment: `ReadHeaderTimeout`, graceful shutdown on SIGTERM, `PORT` env var. Deferred to "API conventions": moving the health handler into `internal/httpapi`, `log/slog`.
