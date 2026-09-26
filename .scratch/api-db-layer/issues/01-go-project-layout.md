# Go project layout

Type: grilling
Status: resolved
Blocked by:

## Question

How is the Go code organised? Packages (e.g. `cmd/api`, `internal/db` for sqlc output,
per-feature service packages), where multi-table transactions live (D11), where
migrations and sqlc query files sit, and what `sqlc.yaml`/Makefile targets exist.

## Answer

- Q1: Go module lives in `api/` (user moved `go.mod`). User also ran `git init` (a Chunk 0 item, done early).
- Q2: module path `github.com/subhayan-bhattacharya/events/api` (user edits go.mod).
- Q3: group by feature: tracker/, event/, tag/, reminder/, telegram/, auth/, plus shared httpapi/ and internal db/ (sqlc). User wants to build reminders first (build-order change pending).
- Build order change agreed: Skeleton → users/sessions → basic events → reminders → scheduler → Telegram → trackers. Apply to map at ticket close.
- Q4: repository pattern per feature: handler (httpapi) → Service (business rules, decides when a tx is needed) → Repository interface (data access only; implemented with sqlc; offers `InTx(func(Repository) error)`). Services unit-testable with a fake repository.
- Q5: sqlc structs never serialised; httpapi has its own request/response types. sqlc config: engine postgresql, sql_package pgx/v5, uuid overridden to github.com/google/uuid, emit_json_tags off.
- Q6: sqlc and goose installed as Go tools (`go get -tool`, run via `go tool sqlc` / `go tool goose`); sequential migration names (`00001_users_sessions.sql`); Makefile targets `db-up`, `migrate-up`, `migrate-down`, `migrate-new name=...`, `sqlc`, `test`, `run`.

Resulting layout:

```
api/
  go.mod                 module github.com/subhayan-bhattacharya/events/api
  cmd/api/main.go        config, DB pool, router, listen
  internal/
    db/                  sqlc output (never hand-edited)
    auth/  event/  reminder/  telegram/  tracker/  tag/
                         each: Service + Repository interface + sqlc-backed repo
    httpapi/             routes, handlers, request/response types, origin check (D31)
  db/migrations/         goose files
  db/queries/            sqlc query files per feature
  sqlc.yaml
  Makefile
```
