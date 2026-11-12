# Ticket 05: dev login and session wiring

Status snapshot: 2026-11-04. This is an implementation guide, not a claim that the planned behavior exists. [Ticket 05](../../.scratch/api-db-layer/issues/05-users-sessions.md) is already **claimed**; continue it without claiming another ticket. The [execution map](../../.scratch/api-db-layer/map.md) and the [resolved dev-login design](../../.scratch/api-db-layer/issues/03-dev-login.md) remain authoritative if this guide becomes stale.

## Goal and work split

First finish the **dev-login slice** of ticket 05. A local developer sends `POST /api/auth/dev-login` from the exact trusted localhost origin. The server uses one configured identity, creates or updates that user, creates a server-side session, and sets its raw token in a secure `__Host-session` cookie. The response is **204 with no body**, as agreed in the planning discussion. The route must be unavailable in an ordinary production build.

The user is implementing the Go behavior one point at a time. The agent will review point 1 and write its focused tests at the user's explicit request, run them, and only then move with the user to point 2. This plan gives instructions without replacing that workflow. Do not start the HTTP work until point 1 has been reviewed and tested. No Go login or auth tests have been added yet at this snapshot.

## Why this design

- [D4](../../Complete%20PostgreSQL%20Database%20Model%20for%20the%20Life%20Events%20App.md) identifies an account by issuer and subject, not email. The fixed dev identity is issuer `dev`, subject `local`; changing its configured email updates the same user.
- [D28](../../Complete%20PostgreSQL%20Database%20Model%20for%20the%20Life%20Events%20App.md) uses server-side sessions so logout can revoke a session. Only a SHA-256 hash of the random cookie token belongs in PostgreSQL.
- [D30](../../Complete%20PostgreSQL%20Database%20Model%20for%20the%20Life%20Events%20App.md) requires a host-only `__Host-session` cookie with `Secure`, `HttpOnly`, `SameSite=Lax`, `Path=/`, and no `Domain`.
- [D31](../../Complete%20PostgreSQL%20Database%20Model%20for%20the%20Life%20Events%20App.md) requires exact `Origin` matching on mutations. The resolved dev exception accepts exactly `http://localhost:8080` on dev login; it is not a general Origin bypass. Production uses the exact production HTTPS origin.
- Keeping user/session decisions in `internal/auth`, HTTP details in `internal/httpapi`, and construction in `cmd/api` follows the [resolved package layout](../../.scratch/api-db-layer/issues/01-go-project-layout.md). Future Google login can reuse session creation.

## What exists now

| Part | State at this snapshot | Evidence |
| --- | --- | --- |
| Users/sessions migration | Written; prior up/down/up and constraint probes passed against local PostgreSQL. | [migration](../../api/db/migrations/00001_users_sessions.sql), [ticket progress](../../.scratch/api-db-layer/issues/05-users-sessions.md) |
| SQL queries and generated bindings | `UpsertUserByIdentity`, `CreateSession`, `AuthenticateSession`, and `DeleteSession` exist. The authentication query refreshes `last_seen_at` only for a valid session. Earlier live SQL boundary probes passed. | [queries](../../api/db/queries/auth.sql), [generated bindings](../../api/internal/db/auth.sql.go) |
| Session creation in Go | `CreateSession` generates 32 random bytes, URL-base64 encodes the raw token, hashes it with SHA-256 for storage, and returns the raw token and a seven-day expiry. `InsertSession` calls sqlc. Earlier build, vet, and existing tests passed before the new upsert commit. | [service](../../api/internal/auth/service.go), [repository](../../api/internal/auth/repository.go), [ticket progress](../../.scratch/api-db-layer/issues/05-users-sessions.md) |
| User upsert in Go | `SessionRepository.UpsertUser` is committed in `23528dc`; it maps the identity/profile to sqlc and returns the user UUID. It has **not yet been reviewed or tested for this handoff**. | [repository](../../api/internal/auth/repository.go) |
| HTTP/server | `main.go` registers health only. There is no dev-login handler, route gate, cookie setting, or auth middleware. | [main](../../api/cmd/api/main.go) |
| Auth tests | No auth tests yet. `internal/testdb` starts PostgreSQL 17 but does not run migrations automatically. | [testdb](../../api/internal/testdb/testdb.go), [test setup decision](../../.scratch/api-db-layer/issues/02-test-setup.md) |

Session policy already agreed: **seven days absolute** from login plus **24 hours idle**. Successful authentication updates `last_seen_at`; it never extends `expires_at`. The fixed expiry is already calculated in `CreateSession`, and the idle check already exists in SQL. The Go code that calls `AuthenticateSession` is still future ticket-05 work.

## Point 1 — user upsert and service orchestration

**Owner:** user implements; agent reviews and writes tests after the user says point 1 is ready.

**Files:** [repository.go](../../api/internal/auth/repository.go), [service.go](../../api/internal/auth/service.go). Keep SQL generated files untouched.

1. Review the committed `UpsertUser` mapping. It must send issuer, subject, email, display name, and time zone from `UserIdentity`, return the UUID from `UpsertUserByIdentity`, and propagate database errors. `display_name` is nullable in SQL, so sqlc uses `pgtype.Text`; `Valid: true` stores the supplied name, while the zero value means SQL `NULL`. The current fixed dev profile requires a nonempty name.
2. Extend the service's repository contract with user upsert. Keep the contract expressed in auth-layer types, leaving generated sqlc types inside the concrete repository.
3. Add a service operation for dev login that accepts the **configured** email, display name, and IANA time zone, constructs issuer `dev` and subject `local`, calls user upsert, then calls the existing `CreateSession` with the returned UUID. Choose a clear operation name, such as `DevLogin`; the handler should call this one operation rather than coordinating database calls itself.
4. Return the raw token and expiry only after session insertion succeeds. If upsert fails, do not attempt session creation. If session insertion fails, return an error and issue no cookie later. The upserted user may remain after a session-insert failure; this slice does not require a transaction for both operations. A repeated login creates another session for the same user and leaves older sessions intact, per ticket 03.
5. Keep fixed identity and configured profile separate: never accept issuer, subject, email, name, or time zone from the HTTP request. The user/profile fields are configuration inputs to the service, while request authentication decisions belong to the handler.

**Agent's focused tests after implementation:** a fake repository checks fixed issuer/subject and profile mapping; call order and returned user UUID; raw token decoding to 32 random bytes; stored hash equals SHA-256 of the returned token and differs from the raw token; expiry is approximately seven days from creation; repeated calls yield distinct tokens; upsert and session-insert failures propagate and expose no usable token. A PostgreSQL repository integration test should prove insert then update preserves the UUID and refreshes profile values, and should exercise a duplicate email under a different identity if included in the acceptance set. Use one migrated testcontainer per suite and rolled-back transactions per repository test, as ticket 02 requires. The exact migration runner and `db.New(tx)` wiring need to be added to the test setup; they are not present yet. Short tests skip Docker cases.

**Checkpoint:** the service can create a dev user's session without HTTP; point-1 tests, build, and vet pass. Record any remaining failures before starting point 2.

## Point 2 — dev-only HTTP handler

**Start only after the point-1 checkpoint. Owner:** user implements; agent reviews and writes focused handler tests when asked under the same work split.

**Files:** new dev-tagged handler in `api/internal/httpapi/` (package location from ticket 01). Keep general HTTP handling separate from auth persistence.

1. Register `POST /api/auth/dev-login` as a dev-only endpoint. Reject a missing or different `Origin` with **403** before creating a user or session. Accept exactly `http://localhost:8080` in the dev build. Method routing should allow only POST.
2. Accept no request-supplied user data. As a concrete handler rule for this plan, reject a nonempty body with **400**. The configured profile is provided by server setup, not decoded from the request.
3. Call the point-1 service operation. On success, set `__Host-session` to the **raw returned token**, with `Secure`, `HttpOnly`, `SameSite=Lax`, `Path=/`, no `Domain`, and an expiry corresponding to the service expiry. Return **204 with an empty body**. Do not put the token in JSON, logs, or an error message.
4. On a service/database failure, return **500** without setting a session cookie. Keep internal error detail out of the response.

**Checkpoint:** handler tests cover valid request/cookie/204, missing or wrong Origin/403 with no service call, nonempty body/400, and service failure/500 with no cookie. A handler unit test does not prove that a browser accepts a Secure cookie over localhost; that is a separate browser check in point 4.

## Point 3 — configuration, construction, and route gates

**Files:** [main.go](../../api/cmd/api/main.go), paired `dev` and `!dev` route-registration files in `api/cmd/api/`, and [`.env.example`](../../.env.example) for nonsecret variable names/placeholders.

1. Construct `db.New(pool)`, the auth repository, and the service at server startup. Pass the service to the dev route-registration path.
2. Compile the dev registration and handler only with the `dev` build tag; provide a `!dev` registration path that does not expose this route. An ordinary build must have no dev-login endpoint in its binary/router.
3. In a dev-tagged build, fail startup unless `APP_ENV=development`, `DEV_USER_EMAIL`, `DEV_USER_NAME`, and `DEV_USER_TIME_ZONE` are nonempty and the time zone resolves to a valid IANA location. Validate before serving requests. Keep the fixed issuer/subject in the auth behavior, not in request input.
4. Document those variable names with safe placeholders in `.env.example`. Do not commit a private `.env` or credential values.

**Checkpoint:** a dev-tagged build with valid config serves the route; invalid/missing config fails startup; a normal build has no dev-login route. Tests should exercise both build variants or an equivalent route-registration seam, alongside `go build` and `go vet` for both variants.

## Point 4 — end-to-end verification of dev login

1. Migrate the isolated PostgreSQL testcontainer, then exercise configured-user upsert, session insertion, and the HTTP response. Check the stored value is the token hash, never the raw cookie token; repeat login keeps one user UUID and creates an additional session. Existing sessions remain until expiry or explicit logout.
2. From `api/`, run `go build ./...`, `go vet ./...`, `go test -short ./...`, and with Docker running `go test ./...`; repeat applicable commands with `-tags dev`. Check sqlc generation only if the SQL/query source changes; generated bindings should not be hand-edited.
3. In a real browser at `localhost:8080`, verify that the `Secure` `__Host-session` cookie is accepted and sent on the next request. If the target browser rejects it over HTTP localhost, use local HTTPS; do not weaken the production cookie flags. Verify missing/wrong Origin is rejected.
4. Update ticket 05 and the [project journal](../project-journal.html) with what actually passed. A passing dev login completes this slice, **not the whole ticket**.

## Remaining ticket-05 work after dev login

1. Implement session authentication using the `__Host-session` cookie: hash the supplied token, call `AuthenticateSession` with the current time, attach the verified user identity to the request, and return 401 for missing, unknown, absolutely expired, or idle-expired sessions. The SQL query already enforces the seven-day absolute and 24-hour idle boundaries and refreshes `last_seen_at` on successful authentication. Keep the raw token out of logs.
2. Apply the exact trusted `Origin` check to mutating authenticated routes per D31. Dev-login's localhost exception must stay confined to the dev-only route/environment; production mutating requests use the exact production HTTPS origin.
3. Implement logout: hash the cookie token, call `DeleteSession`, and clear the host-only cookie with matching name/path/security attributes. Deleting an already missing session should be harmless. Test that the previous token is rejected afterward.
4. Add focused middleware/logout tests and full integration checks for session expiry, inactivity boundary, `last_seen_at` refresh, and revocation. Resolve ticket 05 only when its migration, dev login, middleware, logout, and verification criteria all pass. Ticket 06 API conventions remains separate; ticket 07 basic events is blocked by both 05 and 06.

## Resume checklist

- Re-read the [claimed ticket](../../.scratch/api-db-layer/issues/05-users-sessions.md), the [resolved dev-login design](../../.scratch/api-db-layer/issues/03-dev-login.md), this plan, and `git status` before making claims about progress.
- **Current next action:** user finishes point 1 service orchestration. Agent then reviews it, writes point-1 tests, and verifies them. Move to point 2 only after that checkpoint.
- Keep this file's status statements current when implementation lands. Preserve the distinction between planned behavior, committed code, and tested behavior.
