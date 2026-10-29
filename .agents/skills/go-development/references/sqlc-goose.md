# sqlc v1.31.1 and goose v3.28.0

The agent writes migration and query files; the user reviews them. SQL content rules
(types, constraints, indexes, time zones) live in the `postgres-sql` skill; this file
covers the tool mechanics.

## How they fit together here

- `api/db/migrations/` holds goose plain-SQL migrations. sqlc reads the same folder as its
  schema (`schema: "db/migrations"`), parsing only `-- +goose Up` sections. The migrations
  **are** the schema.
- `api/db/queries/*.sql` holds sqlc queries, one file per feature.
- Generated code goes to `api/internal/db` (package `db`, `sql_package: pgx/v5`,
  `emit_json_tags: false`, uuid overridden to `github.com/google/uuid`). Never hand-edit it.
- Make targets (run from `api/`): `make migrate-new name=<snake_name>`, `make migrate-up`,
  `make migrate-down`, `make sqlc`. They read `DB_URL` from the root `.env`.

## goose

Docs: [site](https://pressly.github.io/goose/) ·
[annotations](https://pressly.github.io/goose/documentation/annotations/) ·
[CLI commands](https://pressly.github.io/goose/documentation/cli-commands/) ·
[environment variables](https://pressly.github.io/goose/documentation/environment-variables/) ·
[Provider](https://pressly.github.io/goose/documentation/provider/) ·
[goose + sqlc](https://pressly.github.io/goose/blog/2024/goose-sqlc/) ·
[pkg.go.dev@v3.28.0](https://pkg.go.dev/github.com/pressly/goose/v3@v3.28.0) ·
[releases](https://github.com/pressly/goose/releases)

- Create files with `make migrate-new` (`goose create <name> sql -s`), which gives
  zero-padded sequential names (`00001_users_sessions.sql`). sqlc reads files in
  **lexicographic** order, so the padding matters.
- Annotations sit on their own line with **no leading whitespace**; they are
  case-insensitive. Exactly one `-- +goose Up`; `-- +goose Down` is optional and must come
  after it. Write a Down section unless the user says otherwise (ask if the Down step
  would destroy data).
- Statements split on `;`. Wrap PL/pgSQL functions and triggers (e.g. the `updated_at`
  trigger in the database model) in `-- +goose StatementBegin` / `-- +goose StatementEnd`.
  Since v3.27 `goose create` templates no longer include these markers.
- Each file runs in one transaction by default. `-- +goose NO TRANSACTION` at the top is
  required for `CREATE INDEX CONCURRENTLY` and similar statements.
- `-- +goose ENVSUB ON` enables `${VAR}` substitution; it is off by default. Keep it off
  unless the user asks.
- The CLI takes no lock. In Kubernetes, migrations run as a Job (D29); concurrent runners
  need the Provider's `WithSessionLocker` (`lock.NewPostgresSessionLocker`).
- Programmatic use (e.g. migrating a test container): `goose.NewProvider(goose.DialectPostgres,
  db *sql.DB, fsys fs.FS, opts...)`. It needs `database/sql`, so open the DB through
  `github.com/jackc/pgx/v5/stdlib` (part of the pgx module already required). How tests
  should run migrations is still undecided: ask.
- v3.28.0 requires Go ≥ 1.26.

## sqlc

Docs (pinned): [index](https://docs.sqlc.dev/en/v1.31.1/) ·
[config](https://docs.sqlc.dev/en/v1.31.1/reference/config.html) ·
[query annotations](https://docs.sqlc.dev/en/v1.31.1/reference/query-annotations.html) ·
[macros](https://docs.sqlc.dev/en/v1.31.1/reference/macros.html) ·
[datatypes](https://docs.sqlc.dev/en/v1.31.1/reference/datatypes.html) ·
[overrides](https://docs.sqlc.dev/en/v1.31.1/howto/overrides.html) ·
[named parameters](https://docs.sqlc.dev/en/v1.31.1/howto/named_parameters.html) ·
[transactions](https://docs.sqlc.dev/en/v1.31.1/howto/transactions.html) ·
[migrations as schema](https://docs.sqlc.dev/en/v1.31.1/howto/ddl.html) ·
[using pgx](https://docs.sqlc.dev/en/v1.31.1/guides/using-go-and-pgx.html) ·
[vet](https://docs.sqlc.dev/en/v1.31.1/howto/vet.html) ·
[changelog](https://docs.sqlc.dev/en/latest/reference/changelog.html)

- Every query starts with `-- name: <GoName> <command>`. Commands: `:one` (returns
  `pgx.ErrNoRows` when empty), `:many`, `:exec`, `:execrows` (affected row count),
  `:execresult`, and pgx-only `:batchexec`, `:batchmany`, `:batchone`, `:copyfrom`.
- Name parameters with `sqlc.arg(name)` or the `@name` shorthand; force a nullable
  parameter with `sqlc.narg(name)` (no `@` form). Cast where sqlc cannot infer a type:
  `@today::date`.
- D32: date-sensitive reads take `@today::date` from Go; never `CURRENT_DATE`.
- For a list parameter in PostgreSQL use `= ANY(@ids::uuid[])`.
- Prefer explicit column lists over `SELECT *` so a migration that adds a column does not
  silently change result structs and diffs stay readable. Ask if the user prefers `*`.
- Generated `Queries.WithTx(tx)` binds queries to a pgx transaction.
- Check after every change: `go tool sqlc generate` (or `make sqlc`), then `go build ./...`.
  `go tool sqlc compile` type-checks without writing files.
- Option changes in `sqlc.yaml` (`emit_interface`, `emit_pointers_for_null_types`,
  `emit_empty_slices`, overrides) change every generated struct: ask first.
- The *latest* sqlc docs can describe unreleased behaviour (e.g. stdlib `uuid`); stay on
  the `v1.31.1` pages.
