# Chunk 1: Users and sessions

Type: task
Status: open
Blocked by: 02, 03, 04

## Question

Build the users and sessions layer.

## Done when

- Migration creates `users` and `sessions` per the plan, applies and rolls back cleanly
- Dev login (per "Dev-only login") creates a user and a session and sets the `__Host-session` cookie
- Session middleware authenticates requests via the hashed token; expired or unknown token gives 401
- Logout deletes the session row

## Plan decisions to check

D4, D23, D28, D30
