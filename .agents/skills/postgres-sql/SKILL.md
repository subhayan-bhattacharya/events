---
name: postgres-sql
description: PostgreSQL 17 schema and SQL for Life Events. Covers goose migrations, sqlc query files, constraints, indexes, enums, triggers, time zones, locking and database roles. Use when writing or reviewing a migration or query, changing the schema, or answering how the database behaves.
metadata:
  versions-checked: "2026-10-29"
---

# PostgreSQL 17: Life Events schema and SQL

## Ground rules

- **Ask, never assume.** If the schema document, a D-number and a ticket disagree, or a
  rule is missing, stop and ask. Confirm PostgreSQL behaviour in the version-17 docs
  before stating it.
- **The agent writes SQL** (migrations, sqlc query files); the user reviews every file.
  Go code stays with the user (see the `go-development` skill).
- **Subagents:** [`../../shared/subagents.md`](../../shared/subagents.md).
  **Undecided choices:** [`../../shared/open-questions.md`](../../shared/open-questions.md).

## Version and docs

Server: `postgres:17-alpine` (`docker-compose.yml`, `api/internal/testdb`). The tag floats
to the newest 17.x minor (17.11 on 2026-10-29). Use `https://www.postgresql.org/docs/17/`,
never `/docs/current/`, which is 18 and documents features 17 lacks (e.g. `uuidv7()`).
Check the running server with `SELECT version();`.

| Topic | Docs (17) |
|---|---|
| Constraints, FKs, `ON DELETE` | [ddl-constraints](https://www.postgresql.org/docs/17/ddl-constraints.html) |
| Indexes, partial indexes | [CREATE INDEX](https://www.postgresql.org/docs/17/sql-createindex.html) · [partial](https://www.postgresql.org/docs/17/indexes-partial.html) |
| Date/time types and functions | [types](https://www.postgresql.org/docs/17/datatype-datetime.html) · [functions](https://www.postgresql.org/docs/17/functions-datetime.html) |
| Enums | [datatype-enum](https://www.postgresql.org/docs/17/datatype-enum.html) · [ALTER TYPE](https://www.postgresql.org/docs/17/sql-altertype.html) |
| Triggers (PL/pgSQL) | [plpgsql-trigger](https://www.postgresql.org/docs/17/plpgsql-trigger.html) |
| CTEs (`WITH ... INSERT`) | [queries-with](https://www.postgresql.org/docs/17/queries-with.html) |
| JSON (`jsonb_agg`) | [functions-json](https://www.postgresql.org/docs/17/functions-json.html) |
| Locking, `FOR UPDATE SKIP LOCKED` | [SELECT locking clause](https://www.postgresql.org/docs/17/sql-select.html#SQL-FOR-UPDATE-SHARE) · [explicit locking](https://www.postgresql.org/docs/17/explicit-locking.html) |
| Isolation levels | [transaction-iso](https://www.postgresql.org/docs/17/transaction-iso.html) |
| Roles and grants | [ddl-priv](https://www.postgresql.org/docs/17/ddl-priv.html) · [GRANT](https://www.postgresql.org/docs/17/sql-grant.html) · [ALTER DEFAULT PRIVILEGES](https://www.postgresql.org/docs/17/sql-alterdefaultprivileges.html) |
| Error codes | [errcodes-appendix](https://www.postgresql.org/docs/17/errcodes-appendix.html) |
| Release notes | [release-17](https://www.postgresql.org/docs/17/release-17.html) · [versioning](https://www.postgresql.org/support/versioning/) |
| Image | [Docker Hub postgres](https://hub.docker.com/_/postgres) |

## Sources of truth for the schema

`Complete PostgreSQL Database Model for the Life Events App.md`: its **decision log
(D1–D32) overrides** the DDL sections below it, which the document itself marks as
under revision. Then the ticket being worked (`.scratch/api-db-layer/issues/`), which
names the tables in scope. Copy DDL from the document only after checking every
relevant D-number; when they conflict, ask.

## Steps for a migration

1. List the tables, enums, constraints, indexes and triggers in scope, each with the
   D-numbers that shape it. Done when every object traces to the document or the ticket.
2. `make migrate-new name=<snake_name>` from `api/`, then write `-- +goose Up` and
   `-- +goose Down` (mechanics: `../go-development/references/sqlc-goose.md`).
3. Write the queries the ticket needs in `api/db/queries/<feature>.sql`.
4. Verify with Docker running, from `api/`: `make db-up`, `make migrate-up`,
   `make migrate-down`, `make migrate-up` (Down must cleanly undo Up), `make sqlc`,
   `go build ./...`. For larger changes dispatch a `reviewer` subagent.
5. Hand the files to the user for review with a short list of what each object enforces
   and which D-number it implements.

## Rules that are easy to get wrong here

- **Dates (D10, D32):** no `CURRENT_DATE` or `now()::date` in user-facing logic; take
  `@today::date` from Go. `timestamptz` stores an instant (UTC internally); conversion to
  a local date happens in Go with the user's IANA zone.
- **UUIDs:** `gen_random_uuid()` is core since PostgreSQL 13. The document's
  `CREATE EXTENSION pgcrypto` is not needed for it (pgcrypto's version is marked
  obsolete). `uuidv7()` exists only from 18. Ask before dropping or keeping the extension.
- **Enums:** a value added with `ALTER TYPE ... ADD VALUE` inside a transaction cannot be
  used until that transaction commits. goose wraps each file in a transaction, so add the
  value and use it in separate migrations. Values cannot be removed: think before adding.
- **Constraint names** are part of the API: Go maps errors by `ConstraintName` (e.g.
  `ck_reminder_end`, `ck_reminder_one_limit`). Name every constraint explicitly; never
  rename one without checking Go code.
- **Deletes (D22–D24):** cascade chains and the deliberate `NO ACTION` on
  `tracker_entries.tracker_id` are design decisions. Change none without asking.
- **D18 uniqueness** `UNIQUE (tracker_id, entry_date, entry_kind)`, with `entry_date`
  copied from `life_events.starts_on` in the same insert.
- **Triggers and functions** need `-- +goose StatementBegin/StatementEnd`.
- **Large-table index changes** use `CREATE INDEX CONCURRENTLY` in a
  `-- +goose NO TRANSACTION` file. At the current data sizes a plain `CREATE INDEX` is
  fine: ask if unsure.
- **Roles (deployment):** `events_migrator` owns the schema and runs goose;
  `events_app` gets only SELECT/INSERT/UPDATE/DELETE. That needs `GRANT`s and
  `ALTER DEFAULT PRIVILEGES`. Not built yet; part of the deployment grilling.
