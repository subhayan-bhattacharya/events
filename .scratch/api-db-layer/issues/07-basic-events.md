# Chunk 2: Basic events

Type: task
Status: open
Blocked by: 05, 06

## Question

The `life_events` table and a minimal event API, enough for reminders to attach to.
Only kinds without a detail table (`GENERAL`, `MILESTONE`) get endpoints here;
vacations, purchases and tracker entries come later.

## Done when

- Migration creates `event_kind`, `event_status` and `life_events` per the plan; applies and rolls back cleanly
- `event.Service` + `event.Repository` (per "Go project layout") can create, get, list, cancel and delete an event
- Endpoints follow "API conventions"; every query is scoped by `user_id`; another user's event gives 404
- Cancel sets `status = 'CANCELLED'`; delete is a hard delete (D22)
- Dates come from Go in `users.time_zone`, never `CURRENT_DATE` (D10)

## Plan decisions to check

D10, D12, D22, D30, D31
