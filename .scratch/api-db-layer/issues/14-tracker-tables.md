# Chunk 6: Tracker tables

Type: task
Status: open
Blocked by: 07

## Question

Migration for `trackers`, `tracker_entries` and their enums (`life_events` already exists from "Chunk 2: Basic events").

## Done when

- A second tracker entry of the same kind on the same day is rejected
- Deleting a tracker that still has entries fails (NO ACTION)
- Deleting a life event removes its tracker entry
- Deleting a user removes everything (does not fail on the NO ACTION FK)
- Paused status and entry kinds exist as in the plan

## Plan decisions to check

D13, D17, D18, D22, D23
