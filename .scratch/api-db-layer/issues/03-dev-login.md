# Dev-only login

Type: grilling
Status: resolved
Blocked by:

## Question

Until Google login ("Real Google login") is built, how does a developer get a session?
What does the dev login endpoint look like, and how is it guaranteed absent from
production builds (build tag, env flag, or both)? Must use the D28/D30 cookie.

## Progress

- Keep authentication methods explicit: `POST /api/auth/dev-login` is separate from
  the future Google `GET /api/auth/login` and `GET /api/auth/callback` routes.
- Dev and Google authentication will share the same session-creation mechanism.
- Dev login accepts no user data in the request. It uses one fixed identity configured
  with `DEV_USER_EMAIL`, `DEV_USER_NAME`, and `DEV_USER_TIME_ZONE`.
- The fixed identity uses `oidc_issuer="dev"` and `oidc_subject="local"`. Login creates
  the user if absent, otherwise refreshes profile fields from configuration, and creates
  a new session without deleting existing sessions.
- Production safety uses both layers: dev-login route code requires the `dev` build tag,
  and route registration additionally requires `APP_ENV=development`. Production builds
  omit the tag, so the endpoint is absent from the binary.
- A dev-tagged API fails at startup unless `APP_ENV=development`, the three `DEV_USER_*`
  values are non-empty, and `DEV_USER_TIME_ZONE` names a valid IANA time zone.

## Answer

The design above is agreed. In a `dev`-tagged build with `APP_ENV=development`,
`POST /api/auth/dev-login` accepts the exact trusted local Origin
`http://localhost:8080` for browser development. A missing or different Origin is
rejected (403); the production Origin remains exactly
`https://events.poulomi-subhayan.click` per D31, with no wildcard or bypass.
The dev endpoint is excluded from production builds. Dev and Google login use the
same D28/D30 session-creation path and `__Host-session` cookie.

This ticket resolves the **design only**; no login implementation or browser cookie
verification was done. Ticket 05 implements users, sessions, dev login and the
Origin check, and verifies whether the Secure cookie works over HTTP localhost in
the target browser. If it does not, use local HTTPS rather than weakening the
production cookie requirements. The local browser origin assumes the browser
uses `localhost:8080`; any future frontend on a different origin needs a separate
explicit decision.
