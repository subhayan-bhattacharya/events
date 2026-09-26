# Scheduler claiming and retries

Type: grilling
Status: open
Blocked by:

## Question

How does the goroutine-ticker scheduler claim due reminders so that two instances never
double-send (e.g. `FOR UPDATE SKIP LOCKED`)? Tick interval, batch size, what a failed
send does (retry count, backoff), and what is recorded in `reminder_deliveries`.
