---
name: go-development
description: Go work in the Life Events api/ module. Covers Go 1.27, net/http handlers, pgx, sqlc, goose and testcontainers-go tests, Go best practice, and choosing a Go package for a task. Use when writing, reviewing, explaining or debugging Go code, or when a task needs a Go library.
metadata:
  versions-checked: "2026-10-29"
---

# Go development: Life Events API

## Ground rules

- **Ask, never assume.** When a requirement, version, API behaviour or design choice is
  unclear, stop and ask the user. Verify every library claim in the docs for the pinned
  version before stating it. "I don't know yet, let me check" beats a confident guess.
- **The user writes the Go code** (`*.go`, tests included) unless they explicitly ask the
  agent to write it. The agent explains, points to the exact doc section, proposes what to
  test, reviews, and runs the build and tests. Non-Go files (SQL, Dockerfiles, manifests)
  the agent writes and the user reviews.
- **Subagents** for parallel research, recon and independent review:
  [`../../shared/subagents.md`](../../shared/subagents.md).
- **Undecided technology choices** are listed in
  [`../../shared/open-questions.md`](../../shared/open-questions.md). Raise the relevant
  one when work touches it.

## Versions in use

`api/go.mod` and `docker-compose.yml` are the source of truth; this table is a snapshot
from the date in the frontmatter. Before relying on a version-specific behaviour, run
`grep -E 'jackc|sqlc|goose|testcontainers|google/uuid|^go ' api/go.mod` and use the
docs for the version it prints.

| Component | Version | Version-pinned docs |
|---|---|---|
| Go toolchain / `go` directive | 1.27.2 / `go 1.27` | [release notes](https://go.dev/doc/go1.27), `https://pkg.go.dev/<pkg>@go1.27.2` |
| pgx | v5.11.0 | [pkg.go.dev@v5.11.0](https://pkg.go.dev/github.com/jackc/pgx/v5@v5.11.0) |
| sqlc (Go tool) | v1.31.1 | [docs.sqlc.dev/en/v1.31.1](https://docs.sqlc.dev/en/v1.31.1/) |
| goose (Go tool) | v3.28.0 | [pkg.go.dev@v3.28.0](https://pkg.go.dev/github.com/pressly/goose/v3@v3.28.0) |
| testcontainers-go postgres | v0.44.0 | [pkg.go.dev@v0.44.0](https://pkg.go.dev/github.com/testcontainers/testcontainers-go/modules/postgres@v0.44.0) |
| google/uuid | v1.6.0 | [pkg.go.dev@v1.6.0](https://pkg.go.dev/github.com/google/uuid@v1.6.0) |
| PostgreSQL (server) | 17 | see the `postgres-sql` skill |

Versioned docs beat "latest": the latest sqlc docs already describe behaviour v1.31.1
lacks. Offline, `go doc <pkg>@<version> <Symbol>` (Go 1.27) reads docs for an exact version.

## Steps

1. **Pin the context.** Read the ticket or request, `project.md`, and any decision the
   task cites (D-numbers in `Complete PostgreSQL Database Model for the Life Events App.md`,
   resolved tickets in `.scratch/api-db-layer/issues/`). Done when you can name the
   decisions that constrain the work, or have asked the user about the gap.
2. **Load the reference for each branch the task touches** (several can apply):
   - Go style, project layout, errors, context, HTTP server → [references/go-practices.md](references/go-practices.md)
   - pgx pool, transactions, errors, types → [references/pgx.md](references/pgx.md)
   - sqlc config and query files, goose migrations → [references/sqlc-goose.md](references/sqlc-goose.md)
   - unit and integration tests, testcontainers → [references/testing.md](references/testing.md)
   - the task needs functionality not already in `go.mod` or the standard library →
     [references/choosing-packages.md](references/choosing-packages.md)
3. **Check the docs for anything not covered.** Open the version-pinned page from the
   table above; for a broad or multi-library question, dispatch `researcher` subagents in
   parallel. Done when every API you mention is confirmed for the pinned version.
4. **Do the work in the mode the user wants.** Default: guide the user, who writes the
   code. Point to the doc section and the project decision behind each suggestion.
5. **Verify** from `api/`: `go build ./...`, `go vet ./...`, `go test -short ./...`, and
   with Docker running `go test ./...`. For generated code, `go tool sqlc generate` and
   `go tool sqlc vet` if configured. Report pass/fail with concrete fixes. For a
   finished chunk, dispatch a `reviewer` subagent for an independent review.
