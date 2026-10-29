# pgx v5.11.0

Docs: [pgx](https://pkg.go.dev/github.com/jackc/pgx/v5@v5.11.0) ·
[pgxpool](https://pkg.go.dev/github.com/jackc/pgx/v5@v5.11.0/pgxpool) ·
[pgconn](https://pkg.go.dev/github.com/jackc/pgx/v5@v5.11.0/pgconn) ·
[pgtype](https://pkg.go.dev/github.com/jackc/pgx/v5@v5.11.0/pgtype) ·
[CHANGELOG](https://github.com/jackc/pgx/blob/master/CHANGELOG.md) ·
[wiki](https://github.com/jackc/pgx/wiki)

Project entry point: `api/internal/postgres.Connect` (`pgxpool.ParseConfig` →
`NewWithConfig` → ping with 3 s timeout).

## Pool and connections

- `*pgxpool.Pool` is safe for concurrent use; a single `*pgx.Conn` is **not**.
- Pool options can go in the URL (`pool_max_conns`, `pool_min_conns`,
  `pool_max_conn_idle_time`, and new in 5.11 `pool_ping_timeout`) or on `pgxpool.Config`.
- An explicitly `Acquire`d connection must be `Release`d.

## Transactions

- `pool.Begin(ctx)` returns `pgx.Tx`. `defer tx.Rollback(ctx)` is safe: after a
  successful `Commit` it is a no-op.
- `pgx.BeginFunc(ctx, db, fn)` commits if `fn` returns nil, otherwise rolls back.
- **Calling `Begin` on a `pgx.Tx` creates a savepoint** (pseudo-nested transaction).
  Use `BeginTx` on a connection when you need a genuinely new transaction or a specific
  isolation level. This is what lets a repository's `InTx` run inside a test's outer
  transaction, provided `InTx` begins from the current `DBTX` rather than the pool
  (open question 5).
- sqlc's `Queries.WithTx(tx)` binds generated queries to a transaction.
- For "claim due reminders" (ticket 10) the usual pattern is
  `SELECT ... FOR UPDATE SKIP LOCKED` inside a transaction; the design is still an open
  grilling ticket, so ask before assuming.

## Errors

- No row: `errors.Is(err, pgx.ErrNoRows)` (sqlc `:one` queries return it).
- Server errors: `var pgErr *pgconn.PgError; errors.As(err, &pgErr)`, then check
  `pgErr.Code` (SQLSTATE) and `pgErr.ConstraintName`. Useful codes:
  `23505` unique_violation, `23503` foreign_key_violation, `23514` check_violation,
  `23502` not_null_violation, `40001` serialization_failure, `40P01` deadlock_detected.
  Reference: [PostgreSQL 17 error codes](https://www.postgresql.org/docs/17/errcodes-appendix.html).
  Named constants live in `github.com/jackc/pgerrcode`, which is **not** a dependency:
  ask before adding it.
- Matching on `ConstraintName` relies on the constraint names in the migrations
  (e.g. `ck_reminder_end`); keep them stable.
- `pgconn.Timeout(err)` reports timeouts; `pgconn.SafeToRetry(err)` reports errors
  guaranteed to have happened before anything was sent.

## Types

- `uuid` columns: sqlc overrides map to `github.com/google/uuid.UUID` / `*uuid.UUID`.
  pgx handles any type whose underlying type is `[16]byte`.
- Nullable columns without an override use `pgtype.*` (`pgtype.Text`, `pgtype.Date`, ...)
  with a `Valid` field (open question 9).
- `date` scans to `time.Time` at midnight UTC through `pgtype.Date`; treat it as a
  calendar date, not an instant.

## Behaviour changes in 5.11 that bite

- `timestamptz` text-format scans now return `time.Local` (or the codec's
  `ScanLocation`), matching binary format. Never rely on `Location()` of a scanned value;
  convert with `.In(userLoc)` (open question 4).
- `date` now **rejects** impossible dates (`2024-02-30`) instead of normalising them.
- Connection URIs are parsed exactly like libpq: `+` is literal, malformed
  percent-encoding is an error, a repeated parameter's **last** value wins, IPv6 hosts need
  brackets. Percent-encode special characters in passwords.
- `pgx.Rows` gained `TypeMap()`: hand-written fakes implementing `Rows` must add it.
- Struct-tag mapping (`RowToStructByName`) matches `db` tags case-insensitively, with exact
  matches preferred.

## PgBouncer (only if introduced later)

The default `QueryExecModeCacheStatement` needs PgBouncer ≥ 1.21 with
`max_prepared_statements > 0`; otherwise use `default_query_exec_mode=exec`.
