# Chunk 8: Tracker HTTP API

Type: task
Status: open
Blocked by: 06, 15

## Question

HTTP endpoints for trackers, following "API conventions".

## Done when

- Endpoints for create, list/dashboard, log, pause, resume, history, delete forever
- Unauthenticated requests get 401; another user's tracker gets 404
- A mutating request with a wrong or missing `Origin` gets 403
- "Today" is computed in Go from `users.time_zone`
- The agent's curl/httptest probe script passes

## Plan decisions to check

D18, D30, D31, D32
