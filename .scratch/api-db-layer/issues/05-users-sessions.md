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
- On 2026-11-03, `api/internal/auth/service.go` and `repository.go` were added:
  session creation generates a random token, stores only its SHA-256 hash through
  sqlc, and returns the raw token and seven-day expiry for the future cookie
  handler. Build, vet and the existing full Go tests pass, but no auth tests or
  HTTP wiring exist yet. The user paused ticket work after requesting a commit
  and push; resume this claimed ticket with dev login, middleware, logout and tests.
- On 2026-11-14, point-1 service orchestration and its first unit-test slice are
  committed and pushed in `8efd33e`. `DevLogin` upserts the fixed `dev`/`local`
  identity with the configured profile before creating a session. Black-box tests
  verify that call order and UUID propagation, plus `CreateSession` forwarding the
  caller context and UUID, returning a 32-byte URL-base64 token, storing only its
  SHA-256 hash, and using the same approximately seven-day expiry in the return and
  repository input. `go build ./...`, `go vet ./...`, and `go test -short ./...`
  pass. Point 1 is still incomplete: distinct-token and upsert/session-insert
  failure tests remain, as do PostgreSQL repository integration tests. No dev-login
  HTTP handler, runtime wiring, authentication middleware, or logout exists yet.
- On 2026-11-15, `TestCreateSessionCreatesDistinctTokens` was added. It calls
  `CreateSession` twice for the same user and verifies two session inserts, different
  raw tokens, and different stored token hashes. The focused test, full short suite,
  build, and vet pass. The remaining point-1 unit cases are upsert failure (no session
  insert and zero token data) and session-insert failure (propagated error and zero
  token data).
- Later on 2026-11-15, `TestDevLoginStopsWhenUpsertFails` was added. A scripted
  upsert error must remain discoverable with `errors.Is`, `DevLogin` must return zero
  `TokenData`, and the repository must record only the upsert call with no inserted
  session. The focused test, full short suite, build, and vet pass. Session-insert
  failure is the only remaining point-1 unit case.
- `TestDevLoginReturnsNoTokenWhenSessionInsertFails` now covers that final unit case.
  After a successful upsert, a scripted insert error remains discoverable through the
  service's wrapping, the returned `TokenData` is zero, and the fake records exactly
  one insert attempt after the upsert. Build, vet, the full short suite, and an
  independent review pass with no actionable findings. The planned point-1 fake-
  repository unit scenarios are complete; PostgreSQL repository integration tests
  remain before the point-1 checkpoint is complete.
- On 2026-11-17, the point-1 PostgreSQL repository checkpoint was completed. Production
  goose migrations are embedded for tests, `testdb.StartMigrated` applies them once to
  an isolated PostgreSQL 17 container, and repository subtests bind sqlc to rollback
  transactions. The integration suite proves that user upsert inserts then refreshes
  one identity while preserving its UUID, a case-insensitive duplicate email under a
  different identity fails on `uq_users_email`, and session insertion persists the
  exact token hash, owner UUID and expiry. The focused Docker tests, complete uncached
  suite, build, vet, short suite, sqlc compile, formatting/diff checks and independent
  review pass. Point 1 is complete; next within this claimed ticket is the dev-only HTTP
  handler in point 2 of `docs/plans/ticket-05-dev-login.md`.
