# Chunk 4: Scheduler

Type: task
Status: open
Blocked by: 09, 10

## Question

Build the scheduler loop per "Scheduler claiming and retries".

## Done when

- Due reminders are claimed, a delivery row is recorded, and `next_fire_at` advances (or the rule completes)
- A test with two concurrent schedulers shows no double delivery
- Graceful shutdown works

## Plan decisions to check

D6, D25
