# Test setup

Type: grilling
Status: resolved
Blocked by:

## Question

How do tests get a database? Options: testcontainers-go (fresh Postgres per test run)
versus the docker-compose Postgres. Also: how is each test isolated (transaction
rollback, truncate, or schema per test), and how are SQL-level constraint checks run?

## Answer

- Integration tests use `testcontainers-go` with `postgres:17-alpine`; they never use
  the development Compose database or `.env` credentials.
- `api/internal/testdb` starts and terminates the container, exposes its connection URL
  and pgx pool, and can begin a transaction for a test.
- Each integration-test suite will share one migrated container. Each repository test
  will run through a transaction and roll it back for isolation. Repository wiring is
  deferred until the first repository exists.
- Integration tests skip under `go test -short ./...`; the complete suite requires
  Docker and runs with `go test ./...`.
- Verified with a real rollback test and a health-handler integration test. Migration
  execution will be added when the first migration exists in the users/sessions chunk.
