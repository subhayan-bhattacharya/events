# RESUME: ticket-05-dev-login (teaching session notes)

Last updated: 2026-11-18 (session 5). Supersedes the earlier notes.

## Plan
`docs/plans/ticket-05-dev-login.md`. Focus from user: Point 3 next (dev-only HTTP handler in Point 2 is complete and verified).
Map rules: the user writes all Go code and >90% of tests; agent explains, reviews, verifies, proposes tests. Agent replies in the map are meant to be short (the teach-plan skill allows longer teaching text).

## Current position
Point 2 complete and verified (2026-11-17). Next up: **Point 3 — Configuration, construction, and route gates** (`api/cmd/api/main.go`, paired `dev` and `!dev` route-registration files in `api/cmd/api/`, and `.env.example`).
Three minor non-blocking doc-comment nits remain in `api/internal/auth/service.go` and `api/internal/httpapi/`: (a) `auth.DevLoginService` comment still says "declared here, by its consumer" and mentions `*auth.Service`; (b) exported `DevLoginHandler` in `devlogin.go` has no doc comment; (c) package doc comment `// Package httpapi ...` sits in `dev`-tagged `devlogin.go` rather than untagged `origin.go`.

## Done (verified 2026-11-17)
- Point 1 complete: `DevLogin` + `DevUserConfig` in `api/internal/auth/service.go`; fake-repo unit tests in `service_test.go`; Postgres integration tests in `repository_test.go` (migrations embedded via `api/db/migrations/embed.go`, `testdb.StartMigrated`). Commits 8efd33e..edd8c79.
- Point 2 complete:
  - `api/internal/httpapi/origin.go` (untagged, `package httpapi`): `requireOrigin(allowed string) func(http.Handler) http.Handler` enforcing exact `Origin` match (D31) and rejecting empty header/allowed values with 403 before calling `next`.
  - `api/internal/httpapi/origin_test.go` (untagged, `package httpapi`): 12-case table-driven `TestRequireOrigin`.
  - `api/internal/httpapi/devlogin.go` (`//go:build dev`, `package httpapi`): `hasPayload` (rejecting non-empty bodies including chunked/unknown-length with 1-byte probe) and `DevLoginHandler(service auth.DevLoginService, devUserDetails auth.DevUserConfig) http.Handler` (wrapped with `requireOrigin("http://localhost:8080")`, returns 400 on payload/read error, 500 on service failure with no cookie, or sets `__Host-session` cookie with `Secure`, `HttpOnly`, `SameSite=Lax`, `Path="/"`, no `Domain`, `Expires: tokenData.Expiration` and writes 204).
  - `api/internal/httpapi/devlogin_test.go` (`//go:build dev`, `package httpapi_test`): 6 black-box handler unit tests covering 204 + cookie attributes, 403 Origin check before service call, 400 non-empty body (known and unknown length), 204 empty unknown-length body, 400 body read error, and 500 service error without cookie.
- Evidence: `gofmt -l .` clean; `go build ./...`, `go build -tags dev ./...`, `go vet ./...`, `go vet -tags dev ./...`, `go test -short ./...`, and `go test -short -tags dev ./...` all pass.

## Approved outline for point 2 (all steps not started unless noted)
1. Package and seam: `api/internal/httpapi/`, dev-tagged file (`//go:build dev`), small consumer-side interface with one method `DevLogin(ctx, auth.DevUserConfig) (auth.TokenData, error)`, constructor taking service + `auth.DevUserConfig` + allowed origin, returning `http.Handler`.
2. Origin check (403): untagged helper/middleware taking the allowed origin; the `http://localhost:8080` constant stays in the dev-tagged file. Table-driven tests (missing, wrong port/scheme/host, trailing slash, `null`, exact match).
3. Body check (400): reject any body incl. chunked (ContentLength -1) via `http.MaxBytesReader`; Origin checked before body.
4. Success/failure: call service; set `__Host-session` (Secure, HttpOnly, SameSite=Lax, Path=/, no Domain, Expires = service expiry); 204 empty body. Service error: log server-side (never the token), 500, no cookie.
5. Handler tests through a `ServeMux` with pattern `POST /api/auth/dev-login` and a spy fake service: valid (cookie+204), wrong method 405, 403 with service not called, 400, 500 with no cookie. Verify build/vet/test with and without `-tags dev`.

## Decisions and answers
- User approved the outline and its defaults: handler tests use `httptest.NewRecorder` (matches existing tests; open question 6 on `httptest.NewTestServer` not adopted for now); route registration and the `APP_ENV` gate stay in point 3; plain status codes, no JSON error body (ticket 06 owns error format and D31 wiring).
- User decided (2026-11-17) to keep `DevLoginService` in package `auth` (with a `var _ DevLoginService = (*Service)(nil)` assertion in service.go), not in `httpapi`: `auth` cannot import `httpapi` (cycle), so the assertion could not sit beside `Service` otherwise. Agent accepted. Step 1 constructor is `httpapi.DevLoginHandler(service auth.DevLoginService, devUserDetails auth.DevUserConfig) http.HandlerFunc` (placeholder 501). Step 2 will change the return type to `http.Handler` because it wraps the inner handler with `requireOrigin`.
- Step 2 design given to user: untagged `origin.go` with unexported `requireOrigin(allowed string) func(http.Handler) http.Handler`; package doc comment moves there; 403 via `http.Error`; exact `==` compare; guard the empty-allowed/empty-header case; constant `http://localhost:8080` and the wrapping live in the dev-tagged file's constructor; table-driven tests in untagged `origin_test.go` (match, missing, host/port/scheme differ, trailing slash, path, `null`, prefix trick, both empty; rejections assert 403 and `next` not called).
- Plan file was NOT edited; its status snapshot and resume checklist are stale (say point 1 unfinished). Offer to refresh it later; user has not answered.
- Earlier undecided nits from point 1 (clock injection, etc.) are not blocking point 2.

## Remaining after point 2
Point 3 config/construction/route gates (`dev` and `!dev` registration files in `api/cmd/api/`, startup validation, `.env.example` placeholders: it currently has only POSTGRES_PASSWORD and DB_URL). Point 4 end-to-end and browser check of the Secure cookie on localhost. Then remaining ticket 05: session middleware (401), D31 Origin on authenticated mutations, logout, expiry/idle tests.

## Learning state
User wrote the auth service, repository and tests independently; comfortable with interfaces, fakes and table tests. First time in `net/http` handler details (cookies, request bodies, build tags): explain carefully.

## Commands (from `api/`)
`gofmt -l .`, `go build ./...`, `go build -tags dev ./...`, `go vet ./...`, `go vet -tags dev ./...`, `go test -short ./...`, `go test ./...` (Docker). Last run 2026-11-17: all pass.

## Gotchas
- `go vet ./...` and `go test ./...` without `-tags dev` do not compile dev-tagged files or their tests: always run both variants.
- `docs/project-journal.html` has an uncommitted edit in the working tree (user's point-1 checkpoint entry).
- Stored session hash is SHA-256 of the base64 cookie string; the future middleware must hash the same way.
- Go's `http.SetCookie` may silently drop invalid cookies; verify the header in tests.
