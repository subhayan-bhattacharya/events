# Chunk 11: Real Google login

Type: task
Status: open
Blocked by: 07

## Question

Replace the dev login with Google OIDC.

## Done when

- Login redirects to Google; the callback at `/api/auth/callback` validates state, nonce and ID token
- A user is matched by issuer and subject, not by email
- The session cookie is set per D28/D30
- Dev login is absent from production builds

## Plan decisions to check

D4, D28, D30
