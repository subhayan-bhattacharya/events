# Complete PostgreSQL Database Model for the Life Events App

> **Status:** under active revision. See the Decision log below before trusting any
> section. The original document assumed an AWS stack; that assumption is void.

***

## Decision log

Last updated during grilling session 2.

### Settled

| # | Decision | Rationale |
|---|---|---|
| D1 | **Go backend**, not AWS-managed services | Runs on a self-managed VPS |
| D2 | **VPS deployment**, Kubernetes eventually | Replaces Lambda / EventBridge / RDS assumptions |
| D3 | **Separate frontend app** calling the Go service | Frontend framework still open |
| D4 | **Google OIDC** for login, replacing AWS Cognito | Google OAuth token is needed anyway for Drive / Photos attachments. Stored as `oidc_issuer` + `oidc_subject` so a second provider is a row, not a migration |
| D5 | **sqlc + pgx** for database access | Every query here is hand-tuned SQL (`LATERAL`, `jsonb_agg`, CTE inserts). An ORM would fight all of it |
| D6 | **Goroutine ticker** now, **River** later | River earns its place when delivery retry/backoff logic becomes a burden, *not* because there are many reminders (those are rows, not jobs) |
| D7 | **Telegram only** for v1 notifications | One HTTPS POST to the Bot API. No spam folder, no VAPID keys, no approval process |
| D8 | **WhatsApp deferred, not rejected** | Cloud API requires pre-approved message templates for business-initiated messages (a 9 AM reminder is business-initiated). `whatsmeow` avoids that but violates ToS and risks a ban on a personal number. `notification_endpoints` already supports adding it later as one enum value |
| D9 | **EMAIL and WEB_PUSH dropped from v1** | Email from a datacenter IP needs a paid relay or ongoing SPF/DKIM/DMARC reputation work; web push needs VAPID keys and service workers |
| D10 | Date is derived **in Go from `users.time_zone`**, never from `CURRENT_DATE` | See Hole 1 below |
| D11 | **Event/detail pairing is enforced by the app only**: no deferred trigger, no composite FK on `kind` | Base event and detail row are always written in one Go transaction, and no endpoint creates them separately. Accepted risk: a manual SQL insert or a buggy query can still create a mismatched pair |
| D12 | **`kind` is never updated in place**. To change type, the app deletes the original event and creates a new one, with the new fields filled in by the user | Keeps the pairing rule (D11) simple: no sqlc query updates `kind`. See open question on what carries over |
| D13 | **Trackers have a `status` of `ACTIVE` or `PAUSED`**, replacing the `active` boolean. A paused tracker computes no counter | Pausing is the everyday way to stop tracking (e.g. after a relapse). History is kept; nothing is deleted |
| D14 | **Pausing logs an entry** (the relapse / end of streak) in the same transaction as setting `status = 'PAUSED'` | Otherwise history shows when a streak started but never when it ended |
| D15 | **Resuming logs a new start entry** (today or a user-chosen date) in the same transaction as setting `status = 'ACTIVE'` | The counter anchors on the latest entry; without a new one it would count straight through the paused period |
| D16 | **Paused trackers stay on the dashboard, greyed out**, showing "Paused since <date of pause entry>" instead of a counter | No extra column needed: the date comes from the D14 entry |
| D17 | **`entry_kind` adopted** on `tracker_entries`: `START`, `RELAPSE`, `OCCURRENCE` | Pause/resume (D14/D15) makes history a mix of relapses and restarts; without a type, "how many times did I relapse?" is unanswerable and cannot be backfilled |
| D18 | **One tracker entry per tracker, per day, per `entry_kind`**: `UNIQUE (tracker_id, entry_date, entry_kind)`. `entry_date` is copied from `life_events.starts_on` in the same insert | Stops double-taps and retries from creating duplicate tracker entries, while still allowing start and relapse on the same day. The copy is needed because one constraint cannot span two tables |
| D19 | **No duplicate protection for other event kinds in v1**; client-generated IDs not adopted for now | Two purchases in a day is legitimate, so a per-day rule is wrong there. A duplicate purchase is visible and easy to delete, unlike a silently wrong counter |
| D20 | **Reminders attach to events only**, never to trackers. `reminder_rules.tracker_id` and the exactly-one-target check are removed; `event_id` is `NOT NULL` | User decision: trackers have no reminders. Also makes pause (D13) independent of reminders |
| D21 | **"Stop reminding me"** sets `status = 'CANCELLED'`, `next_fire_at = NULL`, and records `stopped_at = now()`. The planned `ends_at` is left untouched | Setting `ends_at = now()` would violate `ck_reminder_end` (if stopped before the first fire) and `ck_reminder_one_limit` (if `max_occurrences` is set). `stopped_at` keeps "planned end" and "actually stopped" separate |
| D22 | **Delete means delete.** `life_events.archived_at` is dropped. Deleting an event hard-deletes it, and FK cascades remove its details, tracker link, attachment links, tag links, reminders, targets and deliveries. Planned events are **cancelled** (`status = 'CANCELLED'`), not deleted. Trackers are **paused** (D13) or **deleted forever**: the Go service deletes the tracker's events and then the tracker in one transaction, and `tracker_entries.tracker_id` is `NO ACTION`, so the database refuses to leave orphaned events | One delete path instead of three. Removes the archive filter from ~12 queries and resolves the Hole 3 follow-up (a corrected entry is hard-deleted, freeing its D18 slot). `NO ACTION` rather than `RESTRICT` so that deleting a user, which cascades through `life_events` in the same statement, still succeeds. Trade-off accepted: no undo, only a confirmation dialog |
| D23 | **Account deletion erases immediately**; `users.deleted_at` dropped, email uniqueness is a plain unique index | Simpler, and it matches GDPR erasure expectations |
| D24 | **Telegram endpoints are disabled (`enabled = false`), not deleted**, in normal use. `reminder_deliveries.endpoint_id` gets `ON DELETE CASCADE` so account deletion never gets stuck | Keeps delivery history while the account exists |
| D25 | **Recurring reminders keep local wall-clock time.** Go computes the next fire time in `reminder_rules.time_zone` (`time.Date`/`AddDate` in that location), never by adding fixed 24h intervals. Monthly/yearly on a day that doesn't exist (31st, 29 Feb) clamps to the last day of that month | "Daily at 9:00" must stay 9:00 across DST changes |
| D26 | **`tracker_mode` dropped** (enum and column) | `SINCE_LAST_ENTRY` was the only behaviour ever implemented and `SINCE_START` was unreachable. Re-add if a real tracker needs it |
| D27 | **No Google token storage in v1**; attachments are plain links | Plain links cover "attach my Drive folder". `attachments.provider_resource_id` leaves room for a `user_oauth_tokens` table (encrypted refresh token) when Drive API integration is wanted |
| D28 | **Server-side sessions in an `HttpOnly` cookie**, backed by a `sessions` table | Tokens never touch JavaScript; logout and revocation are a row delete |
| D29 | **goose** for migrations | Plain SQL files that sqlc reads directly; runs as a Kubernetes Job before each deploy |
| D30 | **Single origin: `https://events.poulomi-subhayan.click`**. The frontend is served at `/` and the Go API at `/api/...`, routed by the Kubernetes ingress. Session cookie `__Host-session; Secure; HttpOnly; SameSite=Lax; Path=/`, no `Domain`. Google OIDC callback: `https://events.poulomi-subhayan.click/api/auth/callback` | One origin means no CORS configuration and no shared cookie domain. The `__Host-` prefix makes the browser pin the cookie to this exact host |
| D31 | **Origin check on every mutating request** (POST/PUT/PATCH/DELETE): the `Origin` header must equal `https://events.poulomi-subhayan.click`, else 403 | All `*.poulomi-subhayan.click` subdomains count as the same site, so `SameSite=Lax` alone does not stop requests from a compromised sibling app |
| D32 | **Read queries take `@today` as a parameter**, computed in Go from `users.time_zone`; SQL never calls `CURRENT_DATE`. The `current_tracker_state` view exposes `anchor_date` only; Go computes elapsed days | Applies D10 to the read side. One source of "today", testable by passing a fixed date |

### Open — resume here

**Hole 2: pairing settled by D11/D12.** Deferred by user: when an event's type is
changed (delete + re-create), do its attachments, tags, and reminders carry over
to the new event, or are they lost to `ON DELETE CASCADE`?

Orphaned-tracker-events bug: fixed by D22. Reminders: settled by D20/D21.
Hole 3: settled by D17/D18/D19; its archive follow-up resolved by D22.

Origin and sessions settled by D28/D30/D31. `CURRENT_DATE` rewrite applied (D32).

**Only remaining open item (deferred by user):** Hole 2's carry-over question above.

***

## Purpose

The database should preserve a history of facts rather than only storing the application's current display state. A vacation is a dated life event, a purchase is a dated life event with financial details, and a tracker reset is a dated life event attached to a tracker. Reminders are schedules attached to events, while notification deliveries are an audit trail of what the system actually attempted to send.

The recommended design uses PostgreSQL and is normalized around these responsibilities:

1. **Identity** — users and their preferences.
2. **Life history** — events and event-specific detail tables.
3. **Tracking** — tracker definitions and immutable tracker entries.
4. **Organization** — tags and attachments.
5. **Scheduling** — reminder rules, notification endpoints, and delivery history.

Primary keys, foreign keys, unique constraints, and check constraints protect row identity, relationships, uniqueness, and domain rules. PostgreSQL foreign keys maintain referential integrity between related tables.[^1]

***

## Conceptual model

```text
users
  |
  +--< life_events
  |      |
  |      +---- vacation_details          (0 or 1)
  |      +---- purchase_details          (0 or 1)
  |      +---- tracker_entries >---- trackers
  |      |
  |      +--< attachments
  |      +--< event_tags >---- tags
  |      +--< reminder_rules
  |                |
  |                +--< reminder_targets >---- notification_endpoints
  |                +--< reminder_deliveries
  |
  +--< trackers
  +--< tags
  +--< notification_endpoints
```

The symbols mean:

- `A +--< B`: one A can have many B rows.
- `(0 or 1)`: an optional one-to-one subtype.
- `>----<`: many-to-many through a joining table.

***

## Why this shape

### Events are facts

`life_events` holds the common attributes of anything that happened or occupies a date range: owner, kind, title, dates, notes, and audit timestamps. It does not contain vacation destinations, purchase prices, tracker definitions, or reminder recurrence settings.

### Detail tables contain typed data

`vacation_details` and `purchase_details` are one-to-one subtype tables. They prevent unrelated nullable fields such as `merchant`, `destination`, and `amount` from accumulating in the central event table.

### Trackers and entries differ

A tracker is a long-lived definition such as “Smoking” or “Red meat.” A tracker entry is an immutable occurrence such as “smoked today,” “manually restarted the counter,” or “ate red meat.” The current count is calculated from the latest qualifying entry; an old date is never overwritten.

### Schedules and deliveries differ

A reminder rule says when a notification should fire. A reminder delivery says that the system attempted a particular notification at a particular time. This distinction supports retries, failure logs, multiple delivery channels, and protection against duplicate sends.

***

## Complete SQL schema

The following DDL is a coherent PostgreSQL starting schema. UUIDs make identifiers safe to expose through an API without revealing sequential row counts.

### Extensions and enums

```sql
CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE TYPE event_kind AS ENUM (
    'VACATION',
    'PURCHASE',
    'MILESTONE',
    'TRACKER_ENTRY',
    'GENERAL'
);

CREATE TYPE event_status AS ENUM (
    'PLANNED',
    'COMPLETED',
    'CANCELLED'
);

-- D13
CREATE TYPE tracker_status AS ENUM (
    'ACTIVE',
    'PAUSED'
);

-- D17
CREATE TYPE tracker_entry_kind AS ENUM (
    'START',       -- tracker created, or resumed after a pause (D15)
    'RELAPSE',     -- streak ended; logged when pausing (D14) or on its own
    'OCCURRENCE'   -- neutral event, e.g. "replaced water filter", "ate red meat"
);

CREATE TYPE attachment_kind AS ENUM (
    'GOOGLE_DRIVE_FOLDER',
    'GOOGLE_DRIVE_FILE',
    'GOOGLE_PHOTOS_ALBUM',
    'WEB_LINK'
);

CREATE TYPE recurrence_frequency AS ENUM (
    'ONCE',
    'DAILY',
    'WEEKLY',
    'MONTHLY',
    'YEARLY'
);

CREATE TYPE reminder_status AS ENUM (
    'ACTIVE',
    'PAUSED',
    'COMPLETED',
    'CANCELLED'
);

-- D7/D9: TELEGRAM is the only channel implemented in v1.
-- EMAIL and WEB_PUSH remain in the enum as planned future channels.
CREATE TYPE notification_channel AS ENUM (
    'TELEGRAM',
    'EMAIL',
    'WEB_PUSH'
);

CREATE TYPE delivery_status AS ENUM (
    'PENDING',
    'SENT',
    'FAILED',
    'SKIPPED'
);
```

Enums are appropriate here because these values form small, controlled domains used by application logic. If administrators must add values at runtime, replace a particular enum with a lookup table.

### Users

```sql
CREATE TABLE users (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    oidc_issuer         TEXT NOT NULL,
    oidc_subject        TEXT NOT NULL,
    email               TEXT NOT NULL,
    display_name        TEXT,
    time_zone           TEXT NOT NULL DEFAULT 'Europe/Berlin',
    preferred_currency  CHAR(3) NOT NULL DEFAULT 'EUR',
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT now(),

    UNIQUE (oidc_issuer, oidc_subject)
);

CREATE UNIQUE INDEX uq_users_email
    ON users (lower(email));
```

`oidc_issuer` + `oidc_subject` store the stable identifier from the identity provider (Google, per D4) rather than relying on an editable email address. The pair is unique rather than the subject alone, since subject identifiers are only unique *within* an issuer — this is what allows a second provider to be added without a migration. Account deletion is a hard delete (D23), so no soft-delete column or partial index is needed.

`time_zone` is load-bearing, not cosmetic: per D10 it is the authority for deciding which calendar day an event belongs to.

### Sessions

```sql
-- D28
CREATE TABLE sessions (
    token_hash      BYTEA PRIMARY KEY,      -- SHA-256 of the random value in the cookie
    user_id         UUID NOT NULL
                    REFERENCES users(id) ON DELETE CASCADE,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    last_seen_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    expires_at      TIMESTAMPTZ NOT NULL
);

CREATE INDEX idx_sessions_user ON sessions (user_id);
CREATE INDEX idx_sessions_expiry ON sessions (expires_at);
```

After the Google OIDC callback, the Go service creates a random 32-byte token, stores only its hash, and sets it as an `HttpOnly; Secure; SameSite=Lax` cookie. Storing the hash means a leaked database dump cannot be replayed as live sessions. Logout deletes the row; the scheduler process can purge expired rows.

### Life events

```sql
CREATE TABLE life_events (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id         UUID NOT NULL
                    REFERENCES users(id) ON DELETE CASCADE,

    kind            event_kind NOT NULL,
    status          event_status NOT NULL DEFAULT 'COMPLETED',
    title           TEXT NOT NULL CHECK (length(btrim(title)) > 0),
    description     TEXT,

    starts_on       DATE NOT NULL,
    ends_on         DATE,
    occurred_at     TIMESTAMPTZ,
    time_zone       TEXT,

    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now(),

    CONSTRAINT ck_event_date_order
        CHECK (ends_on IS NULL OR ends_on >= starts_on),

    CONSTRAINT ck_event_exact_time
        CHECK (
            occurred_at IS NULL
            OR time_zone IS NOT NULL
        )
);

CREATE INDEX idx_events_user_start
    ON life_events (user_id, starts_on DESC);

CREATE INDEX idx_events_user_kind_start
    ON life_events (user_id, kind, starts_on DESC);

CREATE INDEX idx_events_user_status_start
    ON life_events (user_id, status, starts_on);
```

> **D10 — which clock defines the date.** `starts_on` must be computed **in Go from
> `users.time_zone`**, never from Postgres's `CURRENT_DATE`. The service and database
> run in UTC; an event logged at 00:30 Tuesday in Europe/Berlin is still 23:30 Monday
> in UTC, so `CURRENT_DATE` would record the wrong day and every "days since" counter
> would be permanently off by one. The read side matches (D32): every query below takes
> `@today`, which Go computes as `time.Now().In(userLoc)` truncated to a date.

`starts_on` is the canonical local calendar date. `ends_on` is inclusive and is used for date ranges such as vacations. `occurred_at` is optional and records the exact instant when precision matters, while `time_zone` records the event's original time-zone context. PostgreSQL distinguishes `date`, timestamp without time zone, and timestamp with time zone, and provides date/time arithmetic and conversion operations.[^3][^4]

Examples:

| Event | `starts_on` | `ends_on` | `occurred_at` |
|---|---:|---:|---:|
| Vacation | Departure date | Return date | Usually null |
| Purchase | Purchase date | Null | Optional exact transaction time |
| Quit smoking | Quit date | Null | Optional exact time |
| Ate red meat | Eating date | Null | Optional exact time |
| Conference | First day | Final day | Optional |

### Vacation details

```sql
CREATE TABLE vacation_details (
    event_id            UUID PRIMARY KEY
                        REFERENCES life_events(id) ON DELETE CASCADE,
    destination_name    TEXT NOT NULL,
    country_code        CHAR(2),
    city                TEXT,
    accommodation       TEXT,
    travel_notes        TEXT
);
```

A vacation is represented by one `life_events` row of kind `VACATION` plus exactly one `vacation_details` row. Multiple countries or stops can later be normalized into a `vacation_stops` table if multi-destination journeys become an important feature.

Optional future expansion:

```sql
CREATE TABLE vacation_stops (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    vacation_id     UUID NOT NULL
                    REFERENCES vacation_details(event_id) ON DELETE CASCADE,
    sequence_no     SMALLINT NOT NULL CHECK (sequence_no > 0),
    place_name      TEXT NOT NULL,
    country_code    CHAR(2),
    arrives_on      DATE,
    departs_on      DATE,
    UNIQUE (vacation_id, sequence_no),
    CHECK (
        departs_on IS NULL
        OR arrives_on IS NULL
        OR departs_on >= arrives_on
    )
);
```

Do not add `vacation_stops` initially unless multi-stop querying is actually required.

### Purchase details

```sql
CREATE TABLE purchase_details (
    event_id            UUID PRIMARY KEY
                        REFERENCES life_events(id) ON DELETE CASCADE,
    item_name           TEXT NOT NULL CHECK (length(btrim(item_name)) > 0),
    merchant            TEXT,
    amount              NUMERIC(14,2) NOT NULL CHECK (amount >= 0),
    currency            CHAR(3) NOT NULL,
    category            TEXT,
    warranty_ends_on    DATE,
    serial_number       TEXT
);

CREATE INDEX idx_purchase_amount
    ON purchase_details (currency, amount DESC);
```

`is_expensive` is intentionally absent. “Expensive” is a query parameter, not a permanent property. A €600 threshold and a €2,000 threshold should work against the same stored purchase.

For meaningful comparisons across currencies, a later version can add a converted amount and preserve the conversion source:

```sql
CREATE TABLE purchase_valuations (
    purchase_event_id   UUID NOT NULL
                        REFERENCES purchase_details(event_id)
                        ON DELETE CASCADE,
    base_currency       CHAR(3) NOT NULL,
    base_amount         NUMERIC(14,2) NOT NULL CHECK (base_amount >= 0),
    exchange_rate       NUMERIC(20,10) NOT NULL CHECK (exchange_rate > 0),
    rate_date           DATE NOT NULL,
    PRIMARY KEY (purchase_event_id, base_currency)
);
```

### Trackers

```sql
CREATE TABLE trackers (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id             UUID NOT NULL
                        REFERENCES users(id) ON DELETE CASCADE,
    name                TEXT NOT NULL,
    display_label       TEXT NOT NULL,
    unit                TEXT NOT NULL DEFAULT 'DAYS'
                        CHECK (unit IN ('DAYS', 'HOURS')),
    color               VARCHAR(7),
    icon                TEXT,
    status              tracker_status NOT NULL DEFAULT 'ACTIVE',
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT now(),

    UNIQUE (user_id, name),
    CHECK (color IS NULL OR color ~ '^#[0-9A-Fa-f]{6}$')
);
```

Example tracker rows:

| `name` | `display_label` | Meaning of an entry |
|---|---|---|
| `Smoking` | `Days smoke-free` | Smoking or manually restarting the streak |
| `Red meat` | `Days since red meat` | Eating red meat |
| `Alcohol` | `Days alcohol-free` | Drinking alcohol |
| `Water filter` | `Days since filter replacement` | Replacing the filter |

### Tracker entries

```sql
CREATE TABLE tracker_entries (
    event_id        UUID PRIMARY KEY
                    REFERENCES life_events(id) ON DELETE CASCADE,
    tracker_id      UUID NOT NULL
                    REFERENCES trackers(id),   -- D22: NO ACTION; see Delete behavior
    entry_kind      tracker_entry_kind NOT NULL,           -- D17
    entry_date      DATE NOT NULL,                         -- D18: copy of life_events.starts_on
    note            TEXT,

    CONSTRAINT uq_tracker_entry_per_day                   -- D18
        UNIQUE (tracker_id, entry_date, entry_kind)
);

CREATE INDEX idx_tracker_entries_tracker
    ON tracker_entries (tracker_id, event_id);
```

`entry_date` must always equal the event's `starts_on`. The insert examples below guarantee this by taking it from the `RETURNING` clause of the `life_events` insert rather than from a separate parameter. Any future "edit entry date" operation must update both columns in one transaction.

The entry's date, exact time, title, and description come from its `life_events` row. `tracker_entries` only establishes that a dated event belongs to a particular tracker. Every tracker entry is therefore an anchor that starts or restarts the elapsed-time calculation.

A tracker row answers **“what is being measured?”** For example, `Smoking — Days smoke-free`. A tracker-entry row answers **“which dated event restarted this tracker?”** One tracker can have any number of entries, so `event_id` belongs in the child table rather than in `trackers`.

`entry_kind` (D17) records what each entry means, so queries like "how many times did I relapse?" can filter on `entry_kind = 'RELAPSE'` instead of parsing titles. Every kind still restarts the "days since" counter; the counter query does not filter on it.

Never update an earlier tracker entry when resetting. Insert a new `life_events` row and `tracker_entries` row in one transaction.

### Attachments

```sql
CREATE TABLE attachments (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    event_id            UUID NOT NULL
                        REFERENCES life_events(id) ON DELETE CASCADE,
    kind                attachment_kind NOT NULL,
    label               TEXT NOT NULL CHECK (length(btrim(label)) > 0),
    external_url        TEXT NOT NULL,
    provider_resource_id TEXT,
    mime_type           TEXT,
    sort_order          INTEGER NOT NULL DEFAULT 0,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now(),

    UNIQUE (event_id, external_url)
);

CREATE INDEX idx_attachments_event
    ON attachments (event_id, sort_order);
```

For a vacation, this table can hold a Google Drive photo folder, Google Photos album, booking confirmation, map, and journal link. `provider_resource_id` leaves room for later Google Drive API integration without making the URL the only identifier.

### Tags

```sql
CREATE TABLE tags (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     UUID NOT NULL
                REFERENCES users(id) ON DELETE CASCADE,
    name        TEXT NOT NULL,
    color       VARCHAR(7),

    UNIQUE (user_id, name),
    CHECK (color IS NULL OR color ~ '^#[0-9A-Fa-f]{6}$')
);

CREATE TABLE event_tags (
    event_id    UUID NOT NULL
                REFERENCES life_events(id) ON DELETE CASCADE,
    tag_id      UUID NOT NULL
                REFERENCES tags(id) ON DELETE CASCADE,
    PRIMARY KEY (event_id, tag_id)
);

CREATE INDEX idx_event_tags_tag
    ON event_tags (tag_id, event_id);
```

Tags are for cross-cutting organization such as `Health`, `Family`, `Italy`, or `Electronics`. They should not determine whether a row is a vacation or purchase; `life_events.kind` and subtype rows provide that meaning.

### Reminder rules

```sql
CREATE TABLE reminder_rules (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    -- D20: reminders belong to events only.
    event_id            UUID NOT NULL
                        REFERENCES life_events(id) ON DELETE CASCADE,

    name                TEXT NOT NULL,
    first_fire_at       TIMESTAMPTZ NOT NULL,
    time_zone           TEXT NOT NULL,

    frequency           recurrence_frequency NOT NULL DEFAULT 'ONCE',
    interval_value      INTEGER NOT NULL DEFAULT 1
                        CHECK (interval_value > 0),

    ends_at             TIMESTAMPTZ,
    max_occurrences     INTEGER CHECK (max_occurrences > 0),
    next_fire_at        TIMESTAMPTZ,
    occurrences_created INTEGER NOT NULL DEFAULT 0
                        CHECK (occurrences_created >= 0),

    status              reminder_status NOT NULL DEFAULT 'ACTIVE',
    stopped_at          TIMESTAMPTZ,                        -- D21
    message             TEXT,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT now(),

    CONSTRAINT ck_reminder_once
        CHECK (
            frequency <> 'ONCE'
            OR (
                interval_value = 1
                AND max_occurrences IS NULL
            )
        ),

    CONSTRAINT ck_reminder_end
        CHECK (ends_at IS NULL OR ends_at >= first_fire_at),

    CONSTRAINT ck_reminder_one_limit
        CHECK (NOT (ends_at IS NOT NULL AND max_occurrences IS NOT NULL)),

    CONSTRAINT ck_active_reminder_next_fire
        CHECK (
            status <> 'ACTIVE'
            OR next_fire_at IS NOT NULL
        ),

    CONSTRAINT ck_reminder_stopped             -- D21
        CHECK (stopped_at IS NULL OR status = 'CANCELLED')
);

CREATE INDEX idx_reminders_due
    ON reminder_rules (next_fire_at)
    WHERE status = 'ACTIVE';

CREATE INDEX idx_reminders_event
    ON reminder_rules (event_id);
```

A partial index is ideal for the scheduler because it only needs active schedules, not completed or cancelled rows.[^5][^2]

Examples:

| Requirement | `frequency` | `interval_value` | Limits |
|---|---:|---:|---|
| Remind once | `ONCE` | `1` | None |
| Every day | `DAILY` | `1` | Optional |
| Every 15 days | `DAILY` | `15` | Optional |
| Every two weeks | `WEEKLY` | `2` | Optional |
| Every year | `YEARLY` | `1` | Optional |

`first_fire_at` is the authoritative start of the schedule. `next_fire_at` is a deliberately cached operational value so the background worker can efficiently find due work. After a successful or terminal attempt, the application computes the following occurrence and updates `next_fire_at`; date and interval operations are supported by PostgreSQL's date/time functions.[^3]

For advanced rules such as “the second Tuesday of each month,” add an `rrule TEXT` column and migrate recurrence calculation to an RFC 5545 library. Do not store both decomposed recurrence fields and an RRULE as simultaneous sources of truth.

### Notification endpoints

```sql
CREATE TABLE notification_endpoints (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id         UUID NOT NULL
                    REFERENCES users(id) ON DELETE CASCADE,
    channel         notification_channel NOT NULL,
    endpoint        TEXT NOT NULL,
    verified_at     TIMESTAMPTZ,
    enabled         BOOLEAN NOT NULL DEFAULT TRUE,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),

    UNIQUE (user_id, channel, endpoint)
);
```

For `TELEGRAM` (the only v1 channel, per D7), `endpoint` contains the numeric Telegram `chat_id`. Delivery is a single `POST https://api.telegram.org/bot<TOKEN>/sendMessage`.

For `EMAIL`, `endpoint` would contain an email address. For `WEB_PUSH`, a serialized push subscription or an opaque reference to encrypted subscription data stored separately. Neither is implemented in v1 (D9).

`verified_at` is meaningful for Telegram too: it records that the user actually started a chat with the bot, without which the bot cannot message them first.

### Reminder targets

```sql
CREATE TABLE reminder_targets (
    reminder_id     UUID NOT NULL
                    REFERENCES reminder_rules(id) ON DELETE CASCADE,
    endpoint_id     UUID NOT NULL
                    REFERENCES notification_endpoints(id) ON DELETE CASCADE,
    PRIMARY KEY (reminder_id, endpoint_id)
);
```

This join table allows one reminder to be delivered to both email and web push without duplicating the recurrence definition.

### Reminder deliveries

```sql
CREATE TABLE reminder_deliveries (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    reminder_id         UUID NOT NULL
                        REFERENCES reminder_rules(id) ON DELETE CASCADE,
    endpoint_id         UUID NOT NULL
                        REFERENCES notification_endpoints(id)
                        ON DELETE CASCADE,                  -- D24

    scheduled_for       TIMESTAMPTZ NOT NULL,
    attempt_number      SMALLINT NOT NULL DEFAULT 1
                        CHECK (attempt_number > 0),
    status              delivery_status NOT NULL DEFAULT 'PENDING',
    attempted_at        TIMESTAMPTZ,
    delivered_at        TIMESTAMPTZ,
    provider_message_id TEXT,
    error_code          TEXT,
    error_message       TEXT,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now(),

    UNIQUE (
        reminder_id,
        endpoint_id,
        scheduled_for,
        attempt_number
    )
);

CREATE INDEX idx_delivery_pending
    ON reminder_deliveries (created_at)
    WHERE status IN ('PENDING', 'FAILED');

CREATE INDEX idx_delivery_reminder_history
    ON reminder_deliveries (reminder_id, scheduled_for DESC);
```

If only one delivery row per scheduled occurrence is preferred, move retries into a separate `delivery_attempts` table. The current model keeps each attempt as an auditable row distinguished by `attempt_number`.

***

## Relationship rules

The intended cardinalities are:

| Parent | Child | Relationship |
|---|---|---|
| User | Life event | One-to-many |
| Life event | Vacation details | Zero-or-one |
| Life event | Purchase details | Zero-or-one |
| Life event | Tracker entry | Zero-or-one |
| Tracker | Tracker entries | One-to-many |
| Life event | Attachments | One-to-many |
| Life event | Tags | Many-to-many |
| Life event | Reminder rules | One-to-many (D20: the only reminder target) |
| Reminder rule | Endpoints | Many-to-many |
| Reminder rule | Deliveries | One-to-many |

A database constraint should also ensure subtype consistency:

- `kind = 'VACATION'` requires one `vacation_details` row.
- `kind = 'PURCHASE'` requires one `purchase_details` row.
- `kind = 'TRACKER_ENTRY'` requires one `tracker_entries` row.
- Other kinds must not have those subtype rows.

Ordinary foreign keys enforce that a detail row cannot exist without its base event, but they cannot by themselves require the reverse relationship. Per D11, the rule is enforced only by the application: the base event and its subtype are inserted together in one API transaction, and no endpoint creates them separately. Per D12, `kind` is never updated; changing an event's type deletes it and creates a new one. A deferred constraint trigger was considered and rejected for now; it remains the hardening path if mismatched rows ever appear.

***

## Transaction examples

### Add a vacation

```sql
BEGIN;

WITH inserted_event AS (
    INSERT INTO life_events (
        user_id,
        kind,
        status,
        title,
        description,
        starts_on,
        ends_on
    ) VALUES (
        $1,
        'VACATION',
        'COMPLETED',
        'Summer vacation in Italy',
        'Rome and Florence',
        DATE '2026-07-04',
        DATE '2026-07-14'
    )
    RETURNING id
)
INSERT INTO vacation_details (
    event_id,
    destination_name,
    country_code,
    city
)
SELECT id, 'Italy', 'IT', 'Rome / Florence'
FROM inserted_event;

COMMIT;
```

Add the Google Drive folder in the same transaction or in a later attachment request:

```sql
INSERT INTO attachments (
    event_id,
    kind,
    label,
    external_url,
    provider_resource_id
) VALUES (
    $1,
    'GOOGLE_DRIVE_FOLDER',
    'Italy vacation photos',
    $2,
    $3
);
```

### Add an expensive purchase

```sql
BEGIN;

WITH inserted_event AS (
    INSERT INTO life_events (
        user_id,
        kind,
        status,
        title,
        starts_on,
        occurred_at,
        time_zone
    ) VALUES (
        $1,
        'PURCHASE',
        'COMPLETED',
        'Bought a new laptop',
        DATE '2026-09-24',
        TIMESTAMPTZ '2026-09-24 16:30:00+02',
        'Europe/Berlin'
    )
    RETURNING id
)
INSERT INTO purchase_details (
    event_id,
    item_name,
    merchant,
    amount,
    currency,
    category
)
SELECT
    id,
    'Laptop',
    'Example Shop',
    1999.00,
    'EUR',
    'Electronics'
FROM inserted_event;

COMMIT;
```

### Start a smoking tracker

First create the stable tracker definition:

```sql
INSERT INTO trackers (
    user_id,
    name,
    display_label,
    unit,
    icon
) VALUES (
    $1,
    'Smoking',
    'Days smoke-free',
    'DAYS',
    'no-smoking'
)
RETURNING id;
```

Then create the first entry:

```sql
BEGIN;

WITH inserted_event AS (
    INSERT INTO life_events (
        user_id,
        kind,
        status,
        title,
        starts_on,
        occurred_at,
        time_zone
    ) VALUES (
        $1,
        'TRACKER_ENTRY',
        'COMPLETED',
        'Quit smoking',
        $2,
        $3,
        $4
    )
    RETURNING id, starts_on
)
INSERT INTO tracker_entries (
    event_id,
    tracker_id,
    entry_kind,
    entry_date,
    note
)
SELECT
    id,
    $5,
    'START',
    starts_on,
    'Beginning of first tracked streak'
FROM inserted_event;

COMMIT;
```

### Reset the smoking tracker

Resetting inserts another event and tracker entry; it does not update the earlier row:

```sql
BEGIN;

WITH inserted_event AS (
    INSERT INTO life_events (
        user_id,
        kind,
        status,
        title,
        description,
        starts_on,
        occurred_at,
        time_zone
    ) VALUES (
        $1,
        'TRACKER_ENTRY',
        'COMPLETED',
        'Smoking counter reset',
        $2,
        $3,
        $4,
        $5
    )
    RETURNING id, starts_on
)
INSERT INTO tracker_entries (
    event_id,
    tracker_id,
    entry_kind,
    entry_date,
    note
)
SELECT
    id,
    $6,
    'RELAPSE',
    starts_on,
    $7
FROM inserted_event;

COMMIT;
```

### Record eating red meat

Use the same transaction pattern with a `Red meat` tracker. This creates a history of every recorded occurrence and makes "days since the last one" a latest-entry query.

Use `entry_kind = 'OCCURRENCE'` (D17). A second red-meat entry on the same day is rejected by `uq_tracker_entry_per_day` (D18); the Go service should treat that unique violation as "already logged today" and return the existing entry rather than an error.

### Add a one-time reminder

```sql
INSERT INTO reminder_rules (
    event_id,
    name,
    first_fire_at,
    time_zone,
    frequency,
    interval_value,
    next_fire_at,
    message
) VALUES (
    $1,
    'One-time reminder',
    $2,
    'Europe/Berlin',
    'ONCE',
    1,
    $2,
    'Remember this event'
)
RETURNING id;
```

### Add a reminder every 15 days

```sql
INSERT INTO reminder_rules (
    event_id,
    name,
    first_fire_at,
    time_zone,
    frequency,
    interval_value,
    next_fire_at,
    message
) VALUES (
    $1,
    'Reminder every 15 days',
    $2,
    'Europe/Berlin',
    'DAILY',
    15,
    $2,
    'Review this event'
)
RETURNING id;
```

***

## Required read queries

### Current tracker values

```sql
SELECT
    t.id,
    t.name,
    t.display_label,
    latest.event_id,
    latest.starts_on AS anchor_date,
    latest.occurred_at AS anchor_time,
    @today::date - latest.starts_on AS elapsed_days   -- D32
FROM trackers t
LEFT JOIN LATERAL (
    SELECT
        te.event_id,
        e.starts_on,
        e.occurred_at
    FROM tracker_entries te
    JOIN life_events e ON e.id = te.event_id
    WHERE te.tracker_id = t.id
    ORDER BY
        e.starts_on DESC,
        e.occurred_at DESC NULLS LAST,
        e.created_at DESC
    LIMIT 1
) latest ON TRUE
WHERE t.user_id = $1
  AND t.status = 'ACTIVE'  -- D13: paused trackers show no counter
ORDER BY t.name;
```

Date subtraction produces an elapsed day count directly in PostgreSQL.[^3] `@today` is sqlc's named-parameter syntax; Go passes the current date in the user's time zone (D32), so tests can pin "today" to any date.

For hour-level trackers, calculate against `occurred_at` rather than `starts_on`, using the user's time zone in the API's presentation layer.

### When was the smoking counter reset?

```sql
SELECT
    e.starts_on,
    e.occurred_at,
    COALESCE(te.note, e.description) AS note
FROM trackers t
JOIN tracker_entries te ON te.tracker_id = t.id
JOIN life_events e ON e.id = te.event_id
WHERE t.user_id = $1
  AND t.name = 'Smoking'
ORDER BY
    e.starts_on DESC,
    e.occurred_at DESC NULLS LAST,
    e.created_at DESC
LIMIT 1;
```

### Full smoking history

```sql
SELECT
    e.id,
    e.starts_on,
    e.occurred_at,
    COALESCE(te.note, e.description) AS note
FROM trackers t
JOIN tracker_entries te ON te.tracker_id = t.id
JOIN life_events e ON e.id = te.event_id
WHERE t.user_id = $1
  AND t.name = 'Smoking'
ORDER BY
    e.starts_on DESC,
    e.occurred_at DESC NULLS LAST;
```

### Days since eating red meat

```sql
SELECT
    @today::date - e.starts_on AS days_since,   -- D32
    e.starts_on AS last_occurrence_date,
    e.occurred_at AS last_occurrence_time
FROM trackers t
JOIN tracker_entries te ON te.tracker_id = t.id
JOIN life_events e ON e.id = te.event_id
WHERE t.user_id = $1
  AND t.name = 'Red meat'
ORDER BY
    e.starts_on DESC,
    e.occurred_at DESC NULLS LAST
LIMIT 1;
```

### Vacations in the previous calendar year

```sql
WITH bounds AS (
    SELECT
        (date_trunc('year', @today::date) - INTERVAL '1 year')::date
            AS from_date,
        date_trunc('year', @today::date)::date
            AS to_date
)
SELECT
    e.id,
    e.title,
    e.starts_on,
    e.ends_on,
    v.destination_name,
    v.country_code,
    v.city
FROM life_events e
JOIN vacation_details v ON v.event_id = e.id
CROSS JOIN bounds b
WHERE e.user_id = $1
  AND e.status <> 'CANCELLED'
  AND e.starts_on < b.to_date
  AND COALESCE(e.ends_on, e.starts_on) >= b.from_date
ORDER BY e.starts_on DESC;
```

The two overlap conditions include a vacation that crossed the year boundary. “Previous calendar year” and “trailing 12 months” should be distinct UI options.

### Vacations in the trailing 12 months

```sql
SELECT
    e.id,
    e.title,
    e.starts_on,
    e.ends_on,
    v.destination_name,
    v.country_code,
    v.city
FROM life_events e
JOIN vacation_details v ON v.event_id = e.id
WHERE e.user_id = $1
  AND e.status <> 'CANCELLED'
  AND e.starts_on <= @today::date
  AND COALESCE(e.ends_on, e.starts_on)
      >= (@today::date - INTERVAL '1 year')::date
ORDER BY e.starts_on DESC;
```

### Vacation with its Drive links

```sql
SELECT
    e.id,
    e.title,
    e.starts_on,
    e.ends_on,
    v.destination_name,
    COALESCE(
        jsonb_agg(
            jsonb_build_object(
                'id', a.id,
                'kind', a.kind,
                'label', a.label,
                'url', a.external_url
            ) ORDER BY a.sort_order, a.created_at
        ) FILTER (WHERE a.id IS NOT NULL),
        '[]'::jsonb
    ) AS attachments
FROM life_events e
JOIN vacation_details v ON v.event_id = e.id
LEFT JOIN attachments a ON a.event_id = e.id
WHERE e.id = $1
  AND e.user_id = $2
GROUP BY e.id, v.event_id;
```

### Expensive purchases

```sql
SELECT
    e.id,
    e.title,
    e.starts_on AS purchased_on,
    p.item_name,
    p.merchant,
    p.amount,
    p.currency,
    p.category
FROM life_events e
JOIN purchase_details p ON p.event_id = e.id
WHERE e.user_id = $1
  AND p.currency = $2
  AND p.amount >= $3
ORDER BY e.starts_on DESC, p.amount DESC;
```

Example parameters could be `$2 = 'EUR'` and `$3 = 1000.00`.

### Calendar month

```sql
SELECT
    e.id,
    e.kind,
    e.status,
    e.title,
    e.starts_on,
    e.ends_on,
    e.occurred_at
FROM life_events e
WHERE e.user_id = $1
  AND e.starts_on < $3::date
  AND COALESCE(e.ends_on, e.starts_on) >= $2::date
ORDER BY e.starts_on, e.occurred_at NULLS LAST;
```

Use `$2` as the inclusive visible start date and `$3` as the exclusive date after the visible calendar period.

### Due reminders

```sql
SELECT
    r.id,
    r.event_id,
    r.name,
    r.next_fire_at,
    r.frequency,
    r.interval_value,
    r.time_zone,
    r.message
FROM reminder_rules r
WHERE r.status = 'ACTIVE'
  AND r.next_fire_at <= now()
ORDER BY r.next_fire_at
FOR UPDATE SKIP LOCKED
LIMIT 100;
```

The scheduler should execute this inside a short transaction. `FOR UPDATE SKIP LOCKED` allows multiple workers to claim different due rows without processing the same row concurrently.

### Reminder history

```sql
SELECT
    d.scheduled_for,
    d.attempt_number,
    d.status,
    d.attempted_at,
    d.delivered_at,
    ep.channel,
    d.error_code,
    d.error_message
FROM reminder_deliveries d
JOIN notification_endpoints ep ON ep.id = d.endpoint_id
WHERE d.reminder_id = $1
ORDER BY d.scheduled_for DESC, d.attempt_number DESC;
```

***

## Reminder processing model

A long-running Go worker processes reminders (D6): a goroutine ticker polling roughly every 60 seconds, replacing the original AWS Lambda / EventBridge design.

**Kubernetes caveat (D2).** A ticker embedded in the API binary fires once *per replica* — scale to two pods and every reminder is delivered twice, and rolling deploys cause the same overlap briefly. Run the scheduler as its own Deployment pinned to `replicas: 1` (same binary, different subcommand), and keep the `UNIQUE (reminder_id, endpoint_id, scheduled_for, attempt_number)` constraint on `reminder_deliveries` so duplicate suppression is structural rather than dependent on the replica count staying at 1.

The database workflow should be:

1. Start a transaction.
2. Lock a batch of due `reminder_rules` rows using `FOR UPDATE SKIP LOCKED`.
3. Create one `reminder_deliveries` row for each configured target endpoint.
4. Advance `next_fire_at`, or mark a one-time rule `COMPLETED`.
5. Commit quickly.
6. Deliver the notifications outside the locking transaction.
7. Update delivery rows to `SENT` or `FAILED`.

For an `ONCE` rule, the processor sets `status = 'COMPLETED'` and `next_fire_at = NULL`. For `DAILY` with `interval_value = 15`, it advances from the previous scheduled time by 15 days, not from the actual worker execution time. This prevents scheduling drift if the worker runs late.

Per D25, the next occurrence is computed in local wall-clock time in `reminder_rules.time_zone`, so "daily at 09:00" stays 09:00 across DST changes. A monthly or yearly reminder on a day that doesn't exist in the target month (the 31st, 29 February) fires on that month's last day, and the following occurrence goes back to the original day. Always compute from `first_fire_at` plus N periods rather than from the previous (possibly clamped) fire time; otherwise one clamp to the 28th would stick forever.

***

## Data ownership and API safety

Every read or mutation must be scoped to the authenticated user's ID. Never accept a `user_id` from the request body as authoritative; derive it from the verified Google OIDC identity (`oidc_issuer` + `oidc_subject`, per D4) and map it to `users.id`.

For example, fetching an event should use both identifiers:

```sql
SELECT *
FROM life_events
WHERE id = $1
  AND user_id = $2;
```

Child-table mutations should similarly join through `life_events` or verify ownership before writing. A future hardening step can enable PostgreSQL row-level security, but application-level ownership checks remain useful for clear API behavior.

***

## Delete behavior

Per D22–D24. The UI only ever sends "delete X"; all cleanup happens in the Go service and the database.

| Thing | User action | What happens |
|---|---|---|
| Event | **Delete** | Hard delete of the `life_events` row. `ON DELETE CASCADE` removes vacation/purchase details, the tracker entry link, attachment links (not the Drive files), tag links (not the tags), reminders, reminder targets and deliveries |
| Planned event | **Cancel** | `status = 'CANCELLED'`; kept as history |
| Tracker | **Pause** | `status = 'PAUSED'` plus a `RELAPSE` entry (D13/D14) |
| Tracker | **Delete forever** | One Go transaction: delete the tracker's `life_events` (cascading their `tracker_entries`), then the `trackers` row. `tracker_entries.tracker_id` is `NO ACTION`, so a buggy path that skips the first step fails instead of orphaning events |
| Telegram endpoint | **Disable** | `enabled = false`; delivery history retained |
| Account | **Delete account** | Hard delete of `users`; everything cascades (D23) |

Deleting a tracker forever:

```sql
BEGIN;

DELETE FROM life_events e
USING tracker_entries te, trackers t
WHERE te.event_id = e.id
  AND t.id = te.tracker_id
  AND t.id = $1
  AND t.user_id = $2;

DELETE FROM trackers
WHERE id = $1
  AND user_id = $2;

COMMIT;
```

Why `NO ACTION` and not `RESTRICT`: deleting a user cascades to both `trackers` and `life_events` in one statement. `RESTRICT` checks immediately and could fail depending on cascade order; `NO ACTION` checks at the end of the statement, by which time the `life_events` cascade has removed every `tracker_entries` row.

Foreign-key delete actions are part of referential-integrity behavior and should be selected per relationship rather than applying `CASCADE` indiscriminately.[^1]

***

## Recommended views

Views can simplify the API without denormalizing storage.

### Current tracker state

```sql
CREATE VIEW current_tracker_state AS
SELECT
    t.id AS tracker_id,
    t.user_id,
    t.name,
    t.display_label,
    t.unit,
    latest.event_id,
    latest.starts_on AS anchor_date,
    latest.occurred_at AS anchor_time
    -- D32: no elapsed_days here; a view cannot take @today. Go computes it.
FROM trackers t
LEFT JOIN LATERAL (
    SELECT
        te.event_id,
        e.starts_on,
        e.occurred_at,
        e.created_at
    FROM tracker_entries te
    JOIN life_events e ON e.id = te.event_id
    WHERE te.tracker_id = t.id
    ORDER BY
        e.starts_on DESC,
        e.occurred_at DESC NULLS LAST,
        e.created_at DESC
    LIMIT 1
) latest ON TRUE
WHERE t.status = 'ACTIVE';  -- D13: paused trackers show no counter
```

### Event attachment summary

```sql
CREATE VIEW event_attachment_summary AS
SELECT
    e.id AS event_id,
    e.user_id,
    COUNT(a.id) AS attachment_count,
    COALESCE(
        jsonb_agg(
            jsonb_build_object(
                'id', a.id,
                'kind', a.kind,
                'label', a.label,
                'url', a.external_url
            ) ORDER BY a.sort_order, a.created_at
        ) FILTER (WHERE a.id IS NOT NULL),
        '[]'::jsonb
    ) AS attachments
FROM life_events e
LEFT JOIN attachments a ON a.event_id = e.id
GROUP BY e.id;
```

Views should expose common read shapes; writes should still go through explicit service methods and transactions.

***

## Index strategy

The initial index set intentionally matches actual access patterns:

| Query pattern | Index |
|---|---|
| Calendar and timeline | `(user_id, starts_on)` |
| Vacations or purchases by date | `(user_id, kind, starts_on)` |
| Due reminders | Partial index on active `next_fire_at` |
| Purchase threshold | `(currency, amount)` |
| Event attachments | `(event_id, sort_order)` |
| Tag filtering | `(tag_id, event_id)` |
| Reminder history | `(reminder_id, scheduled_for)` |

Indexes can make retrieval faster but add write and storage overhead, so additional indexes should be added from observed query plans rather than guessed in advance.[^6]

At personal-app scale, table partitioning is unnecessary. The reminder-delivery table is the most likely to grow, but partitioning should only be considered after real volume and maintenance requirements justify it.

***

## Integrity hardening

### Updated-at trigger

```sql
CREATE OR REPLACE FUNCTION set_updated_at()
RETURNS TRIGGER
LANGUAGE plpgsql
AS $$
BEGIN
    NEW.updated_at = now();
    RETURN NEW;
END;
$$;

CREATE TRIGGER trg_users_updated_at
BEFORE UPDATE ON users
FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE TRIGGER trg_life_events_updated_at
BEFORE UPDATE ON life_events
FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE TRIGGER trg_trackers_updated_at
BEFORE UPDATE ON trackers
FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE TRIGGER trg_reminder_rules_updated_at
BEFORE UPDATE ON reminder_rules
FOR EACH ROW EXECUTE FUNCTION set_updated_at();
```

### Prevent cross-user tracker links

The basic schema relies on the service layer to ensure that a tracker entry and its event belong to the same user. This can be made structurally enforceable by carrying `user_id` into `tracker_entries` and using composite foreign keys:

```sql
ALTER TABLE life_events
    ADD CONSTRAINT uq_event_user UNIQUE (id, user_id);

ALTER TABLE trackers
    ADD CONSTRAINT uq_tracker_user UNIQUE (id, user_id);

ALTER TABLE tracker_entries
    ADD COLUMN user_id UUID;
```

After backfilling `user_id`, make it non-null and replace the single-column foreign keys with:

```sql
FOREIGN KEY (event_id, user_id)
    REFERENCES life_events(id, user_id)
    ON DELETE CASCADE,

FOREIGN KEY (tracker_id, user_id)
    REFERENCES trackers(id, user_id)   -- D22: NO ACTION
```

Composite foreign keys are supported when the referenced column group is unique and are useful when tenant ownership must be enforced inside the database. For a single-user initial deployment, this hardening can be deferred; for a multi-user application, it is recommended before launch.[^1]

***

## What not to store

Avoid storing values that can be derived reliably:

- Do not store `days_since`; calculate it from the latest tracker anchor.
- Do not store `is_expensive`; compare an amount against a requested threshold.
- Do not copy all Google Drive files into the event row; store attachment/resource references.
- Do not pre-create infinite reminder occurrences.
- Do not overwrite tracker history during reset.
- Attach an event-specific reminder through `event_id`, such as “notify me before this vacation.”
- Do not identify semantic types solely from tags.
- Do not put destination, price, currency, and merchant into untyped JSON if they will be filtered or aggregated.

Controlled caches such as `next_fire_at` are acceptable when their purpose and authoritative source are clear.

***

## Example record map

A single vacation produces these rows:

```text
life_events
  id = E1
  kind = VACATION
  starts_on = 2026-07-04
  ends_on = 2026-07-14

vacation_details
  event_id = E1
  destination_name = Italy

attachments
  event_id = E1
  kind = GOOGLE_DRIVE_FOLDER
  external_url = ...

reminder_rules
  event_id = E1
  frequency = ONCE
  first_fire_at = 2026-07-03 18:00+02
```

A smoking reset produces:

```text
trackers
  id = T1
  name = Smoking

life_events
  id = E2
  kind = TRACKER_ENTRY
  starts_on = reset date

tracker_entries
  event_id = E2
  tracker_id = T1
  note = Starting again today
```

An expensive laptop purchase produces:

```text
life_events
  id = E3
  kind = PURCHASE
  starts_on = purchase date

purchase_details
  event_id = E3
  item_name = Laptop
  amount = 1999.00
  currency = EUR
```

This design makes each user question a direct query over typed, historical data rather than an interpretation of free-form notes.

---

## References

1. [Documentation: 18: 5.5. Constraints - PostgreSQL](https://www.postgresql.org/docs/current/ddl-constraints.html) - A foreign key constraint specifies that the values in a column (or a group of columns) must match th...

2. [Documentation: 18: 11.8. Partial Indexes - PostgreSQL](https://www.postgresql.org/docs/current/indexes-partial.html) - A partial index is an index built over a subset of a table; the subset is defined by a conditional e...

3. [18: 9.9. Date/Time Functions and Operators - PostgreSQL](https://www.postgresql.org/docs/current/functions-datetime.html) - Table 9.33 shows the available functions for date/time value processing, illustrates the behaviors o...

4. [Documentation: 18: 8.5. Date/Time Types - PostgreSQL](https://www.postgresql.org/docs/current/datatype-datetime.html) - PostgreSQL supports the full set of SQL date and time types, saving time when doing date/time arithm...

5. [Documentation: 18: CREATE INDEX - PostgreSQL](https://www.postgresql.org/docs/current/sql-createindex.html) - A partial index is an index that contains entries for only a portion of a table, usually a portion t...

6. [Documentation: 18: Chapter 11. Indexes - PostgreSQL](https://www.postgresql.org/docs/current/indexes.html) - Partial Indexes 11.9. Indexes are a common way to enhance database performance. An index allows the ...

