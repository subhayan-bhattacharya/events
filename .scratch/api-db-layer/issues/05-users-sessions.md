# Chunk 1: Users and sessions

Type: task
Status: claimed
Blocked by: 02, 03, 04

## Question

Build the users and sessions layer.

## Done when

- Migration creates `users` and `sessions` per the plan, applies and rolls back cleanly
- Dev login (per "Dev-only login") creates a user and a session and sets the `__Host-session` cookie
- Session middleware authenticates requests via the hashed token; expired or unknown token gives 401
- Logout deletes the session row

## Plan decisions to check

D4, D23, D28, D30

## Progress

- User writes Go code and tests; agent writes SQL and other non-Go files for review.
- `pgcrypto` omitted: PostgreSQL 17 has built-in `gen_random_uuid()`.
- Migration `api/db/migrations/00001_users_sessions.sql` written and verified locally:
  up → down (tables and function removed) → up; probes confirmed defaults, named
  uniqueness/PK/FK errors (`uq_users_oidc_identity`, `uq_users_email`, `pk_sessions`,
  `fk_sessions_user`), session cascade on user delete, and the `updated_at` trigger.
  `go build`, `go vet`, `go test -short` pass. Go tests for it not yet written.
- Queries `api/db/queries/auth.sql` written (agent), generated into `api/internal/db`:
  `UpsertUserByIdentity`, `CreateSession`, `AuthenticateSession`, `DeleteSession`.
  The user reports reviewing the code so far. The new `AuthenticateSession` query
  refreshes `last_seen_at` only when the session is within both limits; no row means
  unknown, absolutely expired, or idle-expired. sqlc generation, build, vet and
  short tests pass. With Docker available on 2026-11-02, the local database was
  confirmed at migration version 1. A rolled-back SQL probe confirmed successful
  `last_seen_at` refresh without changing `expires_at`, rejection at the exact
  24-hour idle boundary, and rejection at the absolute expiry boundary.
  `go test ./...` passed; no Go auth tests exist yet.
- Session policy agreed in the 2026-11-02 conversation: seven days absolute from
  login (`expires_at` set by Go), plus 24 hours of inactivity. Successful
  authentication refreshes `last_seen_at` without extending `expires_at`.
