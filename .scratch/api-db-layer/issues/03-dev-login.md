# Dev-only login

Type: grilling
Status: claimed
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
