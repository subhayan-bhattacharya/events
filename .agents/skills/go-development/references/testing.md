# Testing

Decisions: `.scratch/api-db-layer/issues/02-test-setup.md` (`## Answer`). The user writes
the tests; the agent proposes cases (load the global `tdd` skill for test-first work) and
may write small break-it probes only when the user agrees.

Docs: [testing](https://pkg.go.dev/testing@go1.27.2) ·
[httptest](https://pkg.go.dev/net/http/httptest@go1.27.2) ·
[synctest](https://pkg.go.dev/testing/synctest@go1.27.2) ·
[testcontainers-go](https://golang.testcontainers.org/) ·
[postgres module](https://golang.testcontainers.org/modules/postgres/) ·
[postgres module @v0.44.0](https://pkg.go.dev/github.com/testcontainers/testcontainers-go/modules/postgres@v0.44.0) ·
[v0.44.0 release](https://github.com/testcontainers/testcontainers-go/releases/tag/v0.44.0)

## Project conventions

- Integration tests begin with `if testing.Short() { t.Skip("integration test requires Docker") }`.
  `go test -short ./...` runs unit tests only; `go test ./...` needs Docker.
- `internal/testdb.Start(ctx)` starts `postgres:17-alpine` (database `events_test`) and
  returns a pgx pool and URL; `Close` terminates it. `Begin` starts a per-test transaction
  that the test rolls back. Tests never touch the Compose database or `.env`.
- Planned: one migrated container per suite (`TestMain`) and rolled-back transactions
  per repository test. How migrations run in tests, and how `InTx` composes with the
  test transaction, are open (open question 5): ask before wiring them.
- Unit tests use fakes for small interfaces (e.g. `Pinger` in `cmd/api`), table-driven
  with `t.Run`.

## testcontainers-go v0.44.0 (postgres module)

- `postgres.Run(ctx, image, opts...)`. The older `RunContainer` is deprecated.
- Options: `WithDatabase`, `WithUsername`, `WithPassword`, `WithInitScripts`,
  `WithOrderedInitScripts`, `WithConfigFile`, `BasicWaitStrategies()` (the module sets
  no wait strategy by default; this waits for the second "ready" log plus the port,
  which avoids flakiness on macOS).
- `ctr.ConnectionString(ctx, "sslmode=disable")`; the container has no TLS.
- Cleanup: `testcontainers.CleanupContainer(t, ctr)` registers termination with
  `t.Cleanup`; `testcontainers.TerminateContainer(ctr)` outside tests. Ryuk also reaps
  leftovers.
- Snapshot/Restore (`ctr.Snapshot`, `ctr.Restore`) resets the whole database between
  tests: an alternative to rollback isolation for tests that need real commits (HTTP or
  scheduler tests). It requires a database name other than `postgres`;
  `WithSQLDriver("pgx")` makes it use pgx's `stdlib` driver instead of `docker exec`.
  Not adopted: ask.
- "failed to connect to the docker API at unix:///Users/.../.docker/run/docker.sock"
  means Docker Desktop is not running. Tell the user; don't try workarounds.

## Isolation facts that affect test design

- Rolled-back transactions do not reset sequences (`nextval` is non-transactional), so
  never assert on generated serial values. UUID keys avoid this.
- `now()` is fixed for the whole transaction; time-based assertions should use values
  passed in from Go (D32) rather than the database clock.
- Code that commits on its own connection (pool-level `Begin`) escapes a test transaction.

## HTTP and time

- Handler tests: `httptest.NewRequest` + `httptest.NewRecorder` (current style).
  Go 1.27 `httptest.NewTestServer` runs a full server on an in-memory network and pairs
  with `testing/synctest` (open question 6).
- Scheduler and reminder timing: `testing/synctest` (`synctest.Test(t, func(t *testing.T){...})`,
  `synctest.Wait`, Go 1.27 `synctest.Sleep`) runs goroutines on a fake clock. Pure
  next-fire-time functions should take the time and location as arguments and need no
  fake clock at all. Cover DST transitions (ask the user which time zones matter) and the
  month-end clamp (D25).

## Commands (from `api/`)

`go test -short ./...` · `go test ./...` · `go test -run TestName ./internal/...` ·
`go test -race ./...` · `go vet ./...` (Go 1.27 `go test` also runs the `stdversion` check).
