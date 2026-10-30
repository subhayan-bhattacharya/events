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
  `UpsertUserByIdentity`, `CreateSession`, `GetSessionUser` (takes `now` from Go;
  no row = unknown/expired), `DeleteSession`. `last_seen_at` is not updated per
  request (agent recommendation, pending user confirmation). Build/vet pass;
  query SQL probed by hand. Awaiting user review.
- Open: session lifetime (needed before the Go expiry logic).
