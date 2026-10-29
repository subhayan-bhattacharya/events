# Map: API and database layer

Label: wayfinder:map

## Destination

A working, tested Go API and PostgreSQL layer for the Life Events app (trackers,
vacations, purchases, tags, attachment links, reminders, Telegram, Google login),
built chunk by chunk by the user, each chunk verified by the agent against the plan,
and **deployed to Kubernetes on a VPS** at `events.poulomi-subhayan.click`.

## Notes

**This map carries execution** (overrides wayfinder's plan-only default): chunk
tickets (`Type: task`) are built here.

**Source of truth for the data model:** `../../Complete PostgreSQL Database Model for the Life Events App.md`
(decision log D1–D32). Tickets cite D-numbers; never restate them.

**Session protocol: one ticket per session, then the user clears the context.**

1. User says "work the map `.scratch/api-db-layer/map.md`" (optionally naming a ticket).
2. Agent reads this map, scans `issues/` for the frontier (open, unblocked, unclaimed;
   lowest number wins), sets `Status: claimed`, and saves before any work.
3. **Decision ticket** (`Type: grilling`): load the grilling and domain-modeling
   skills, grill the user, and record the answer.
4. **Chunk ticket** (`Type: task`):
   - **The user writes all implementation code. The agent never writes implementation code.**
   - **The user writes >90% of all code, tests included.** The agent's code is limited
     to small acceptance or break-it probes used during verification (load the `tdd`
     skill), and only when the user agrees. The agent proposes *what* to test; the user
     writes it.
   - When the user says "verify", the agent reads the code, runs migrations and tests,
     runs its own break-it probes, and reviews against the cited D-numbers and resolved
     decision tickets. It reports pass/fail with concrete fixes, and the user fixes them.
     Repeat until both agree.
5. Resolve: append `## Answer` (what was built, file paths, facts later tickets need),
   set `Status: resolved`, and add one line to "Decisions so far" below. User clears context.

Docker must be running for verification. Plain-language explanations preferred.

**Replies to the user: never more than 10 lines.** Ask one question at a time.

## Decisions so far

- [Database plan, D1–D32](../../Complete%20PostgreSQL%20Database%20Model%20for%20the%20Life%20Events%20App.md): full schema and behaviour settled in two grilling sessions before this map
- [Go project layout](issues/01-go-project-layout.md): module in `api/`, packages grouped by feature, repository pattern (Service → Repository → sqlc), separate API types, sqlc/goose as Go tools. Build order changed: reminders before trackers
- [Test setup](issues/02-test-setup.md): testcontainers-go with PostgreSQL 17; one migrated container per suite and rollback transactions per repository test; integration tests skip in short mode
- [Chunk 0: Skeleton](issues/04-skeleton.md): module builds; `GET /api/healthz` returns 200/503 from a real DB ping; config via root `.env`
- [Dev-only login](issues/03-dev-login.md): dev-tagged route and development env gate; fixed configured identity, shared session creation; exact `http://localhost:8080` Origin accepted only in development, missing/wrong Origin rejected; production keeps D31's exact HTTPS Origin. Implementation and localhost Secure-cookie verification belong to 05.

## Not yet specified

- What carries over (attachments, tags, reminders) when an event's type is changed via
  delete + re-create (plan Hole 2 / Q6, deferred by user). Likely surfaces around
  "Vacations and purchases".
- Pagination and filtering for history/list endpoints: shape depends on the API
  conventions decision.
- Observability (logging format, request IDs) and configuration loading.
- **Deployment** (in scope; grill it when the chunks reach that stage): VPS provider and
  Kubernetes flavour, container image, manifests, ingress routing `/api` (D30), TLS for
  `events.poulomi-subhayan.click`, goose migration Job (D29), secrets, CI/CD, backups,
  HTTP server hardening (`ReadHeaderTimeout`, graceful shutdown on SIGTERM, `PORT`).
  Includes **separate database users**: `events_migrator` (owns schema, used only by the
  migration Job) and `events_app` (SELECT/INSERT/UPDATE/DELETE only, used by the API).
  The `postgres` superuser in the Makefile is for local development only.

## Out of scope

- Frontend app (framework undecided, D3): a separate effort.
