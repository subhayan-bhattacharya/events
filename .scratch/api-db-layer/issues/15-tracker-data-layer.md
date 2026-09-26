# Chunk 7: Tracker data layer

Type: task
Status: open
Blocked by: 14

## Question

sqlc queries and Go functions for trackers.

## Done when

- Create tracker writes the tracker, event and START entry in one transaction
- Log entry writes event and entry in one transaction; a same-day duplicate returns the existing entry, not an error
- Pause logs a RELAPSE and sets PAUSED; resume logs a START and sets ACTIVE (one transaction each)
- Delete forever removes the tracker's events, then the tracker
- Dashboard takes `today` as a parameter; nothing uses `CURRENT_DATE`; paused trackers show "paused since"
- Every query is scoped by `user_id`

## Plan decisions to check

D10–D18, D22, D32
