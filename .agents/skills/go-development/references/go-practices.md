# Go practices for this project

## Authorities, in order

When they disagree, a project decision wins, then the earlier entry here.

1. **Project decisions**: `project.md`, the D-log, resolved tickets in `.scratch/`.
   Layout and layering: `.scratch/api-db-layer/issues/01-go-project-layout.md` (`## Answer`).
2. [Go release notes 1.27](https://go.dev/doc/go1.27) and the [language spec](https://go.dev/ref/spec).
3. [Effective Go](https://go.dev/doc/effective_go): idioms, naming, errors, concurrency.
4. [Go Code Review Comments](https://go.dev/wiki/CodeReviewComments): the Go team's
   review checklist (error strings lower-case without punctuation, `ctx` first parameter,
   receiver names, indent error flow, interfaces belong to the consumer).
5. [Google Go Style Guide](https://google.github.io/styleguide/go/): the
   [guide](https://google.github.io/styleguide/go/guide),
   [decisions](https://google.github.io/styleguide/go/decisions) and
   [best practices](https://google.github.io/styleguide/go/best-practices).
6. Topic docs: [module layout](https://go.dev/doc/modules/layout),
   [database access](https://go.dev/doc/database/),
   [security best practices](https://go.dev/doc/security/best-practices),
   [ServeMux routing](https://go.dev/blog/routing-enhancements).

Cite the specific section when suggesting a change, so the user can read the reasoning.

## Project layering (from ticket 01)

handler (`internal/httpapi`) → Service (business rules, decides transactions) →
Repository interface (data access only, sqlc-backed, offers `InTx(func(Repository) error)`).
Generated sqlc structs never leave the repository layer and are never JSON-encoded;
`httpapi` has its own request/response types. Packages are grouped by feature
(`auth`, `event`, `reminder`, `telegram`, `tracker`, `tag`).

## Time and dates (D10, D25, D32)

- "Today" is computed in Go from `users.time_zone` via `time.LoadLocation`, and passed to
  SQL as a parameter. SQL never uses `CURRENT_DATE` for user-facing logic.
- Recurrence keeps wall-clock time: build the next fire time with `time.Date(...)` /
  `AddDate` in the rule's location, never by adding `24*time.Hour`. Clamp day 31 / 29 Feb
  to the month's last day (D25).
- pgx returns `timestamptz` in `time.Local`; convert with `.In(loc)` before deriving a
  date (see `pgx.md`, open question 4).
- Inject a clock (a `func() time.Time` or small interface) into code that reads "now", so
  tests pass a fixed time. For goroutine/timer code, `testing/synctest` gives a fake
  clock (`synctest.Test`, and `synctest.Sleep` new in 1.27).

## Errors

- Wrap with context using `fmt.Errorf("verb noun: %w", err)`; match with `errors.Is` /
  `errors.As`. Go 1.27 `vet` now flags `%w` with a `*E` operand when `E` itself
  implements `error`.
- Translate database errors to domain errors in the repository (e.g. `pgx.ErrNoRows` →
  a `NotFound` error, unique violation → a conflict error), so services and handlers
  never import pgx. Exact error types and HTTP status mapping belong to the
  "API conventions" ticket: ask before inventing them.
- `errors.Join` for combining cleanup errors (already used in `internal/testdb`).

## context.Context

First parameter, named `ctx`, never stored in structs. Handlers derive from
`r.Context()`; add a timeout for outbound calls (the health handler uses 2 s). Background
workers (the reminder scheduler) take a root context cancelled on shutdown.

## HTTP (standard library only, D-log and project.md)

- Routing via `http.ServeMux` patterns: `"GET /api/healthz"`, wildcards `{id}`, read with
  `r.PathValue("id")`, `{$}` for an exact trailing slash. Ambiguous overlapping patterns
  panic at registration. Method mismatch yields 405 automatically.
- Every mutating request (POST/PUT/PATCH/DELETE) needs the exact `Origin` check (D31).
- Session cookie `__Host-session; Secure; HttpOnly; SameSite=Lax; Path=/`, no `Domain` (D30).
- Deferred to deployment (ticket 04 answer): `ReadHeaderTimeout`, graceful shutdown on
  SIGTERM via `Server.Shutdown`, `PORT` env var. Go 1.27 adds
  `Server.MaxHeaderValueCount`. Raise these when the deployment chunk starts.
- Logging: `log/slog` is the planned direction but deferred to "API conventions".

## Go 1.27 changes worth knowing here

- Generic methods; struct-literal keys may be promoted field selectors.
- `go test` runs the `stdversion` vet check: using std symbols newer than `go 1.27` fails.
- `go mod tidy` merges `require` blocks into one direct and one indirect block.
- `go doc pkg@version` and `go doc -ex`.
- New packages: `uuid`, `encoding/json/v2`, `encoding/json/jsontext`, `crypto/mldsa`
  (see open questions 1 and 2 before adopting).
- `net/http/httptest.NewTestServer` (open question 6); `strings.CutLast`, `bytes.CutLast`;
  `url.URL.Clone`, `url.Values.Clone`.
- The `asynctimerchan` GODEBUG is gone: `time` channels are always unbuffered.

## Tooling

- `gofmt` (1.27 changed column alignment slightly, so expect small diffs), `go vet`.
- Dependencies as Go tools: `go get -tool <module>`, run with `go tool <name>` (how sqlc
  and goose are installed).
- Vulnerabilities: [`govulncheck`](https://go.dev/doc/tutorial/govulncheck) reports only
  vulnerabilities your code actually reaches. It is not yet a tool in `go.mod`: ask
  before adding it.
