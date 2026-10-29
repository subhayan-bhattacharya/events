# Life Events

## What this project is

Life Events is a personal application for recording and reviewing meaningful events over time. It will support:

- general dated events
- vacations and purchases with type-specific details
- habit and streak trackers with a complete history
- tags and attachment links
- recurring or one-time reminders
- Telegram notifications

The system treats events as historical facts rather than storing only the latest display state. Tracker changes and notification attempts are therefore recorded as history instead of overwriting previous values.

## Current status

The project is in its early backend phase. The Go API skeleton, PostgreSQL connection, health endpoint, local database setup, and container-based integration test foundation exist. Users, sessions, migrations, and the feature APIs are still to be built.

The frontend is a separate future application; its framework has not been selected.

## Technology and architecture

- **Backend:** Go, in the `api/` module
- **HTTP:** Go standard library HTTP server and routes under `/api`
- **Database:** PostgreSQL 17
- **Database driver:** `pgx`
- **SQL access:** hand-written SQL with `sqlc`; no ORM
- **Migrations:** `goose`, using sequential plain-SQL migrations
- **Local development:** Docker Compose and a root `.env` file
- **Testing:** Go tests plus `testcontainers-go` with PostgreSQL 17
- **Deployment direction:** a self-managed VPS running Kubernetes

Code will be grouped by feature. HTTP handlers call services, services contain business rules and transaction boundaries, and repository interfaces isolate database access. Generated `sqlc` types remain internal; API request and response types are defined separately.

## Data model decisions

- `life_events` stores common event data; vacation and purchase data live in dedicated detail tables.
- Trackers are long-lived definitions, while tracker entries are immutable historical events.
- Trackers can be active or paused. Pausing and resuming create history entries.
- Dates are calculated in Go using the user's IANA time zone. SQL must not use `CURRENT_DATE` for user-facing date logic.
- Reminders belong to events, not trackers. Reminder rules and delivery attempts are stored separately.
- Recurring reminders preserve local wall-clock time across daylight-saving changes.
- Deletes are permanent. Related rows are removed through transactions and database cascades; planned events are cancelled rather than deleted.
- Attachments are plain links in v1. Google access tokens will not be stored initially.
- Database constraints protect ownership, relationships, uniqueness, and valid states, while some cross-table rules remain enforced by the application.

## Authentication and security

- Production login will use Google OIDC.
- Authentication identities are stored as issuer and subject rather than relying on email addresses.
- Sessions are server-side and referenced by a `__Host-session` cookie with `Secure`, `HttpOnly`, `SameSite=Lax`, and `Path=/`.
- The frontend and API will share `https://events.poulomi-subhayan.click`, with the API served under `/api`; this avoids CORS configuration.
- Mutating requests will require an exact trusted `Origin` header.
- The development login under discussion uses a fixed configured user, is compiled only with a `dev` build tag, and is additionally restricted to `APP_ENV=development`.

## Notifications

Telegram is the only notification channel planned for v1. Email, web push, and WhatsApp are deferred. The first scheduler will be a Go worker polling for due reminders; a dedicated job system such as River may be introduced later if retry and backoff requirements justify it.

## Testing decisions

Integration tests start an isolated PostgreSQL container rather than using the developer's Compose database. A test suite shares a migrated container, while repository tests use rolled-back transactions for isolation. `go test -short ./...` skips integration tests; the complete suite requires Docker.

## Open decisions

- frontend framework and implementation
- pagination, filtering, and final API conventions
- whether tags, attachments, and reminders carry over when an event changes type by delete-and-recreate
- observability, production configuration, backups, CI/CD, and final Kubernetes deployment details
