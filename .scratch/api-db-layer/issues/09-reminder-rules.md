# Chunk 3: Reminder rules

Type: task
Status: open
Blocked by: 07, 08

## Question

Reminder rule schema, next-fire-time calculation, and endpoints.

## Done when

- A pure Go next-fire function passes table-driven tests: daily 09:00 across the March and October DST changes, monthly on the 31st, yearly on 29 February, computed from `first_fire_at` plus N periods
- "Stop" sets `CANCELLED`, clears `next_fire_at`, sets `stopped_at`
- Reminders exist only on events

## Plan decisions to check

D20, D21, D25
