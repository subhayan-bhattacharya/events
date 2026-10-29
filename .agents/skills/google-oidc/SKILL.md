---
name: google-oidc
description: Google OpenID Connect login for Life Events. Covers the authorization-code flow with PKCE, state and nonce, ID-token validation, mapping Google identities to users, sessions and the dev-login boundary. Use when designing, building, reviewing or debugging login, logout, the auth callback or session creation.
metadata:
  versions-checked: "2026-10-29"
---

# Google OIDC login

## Ground rules

- **Ask, never assume.** Login is security-critical. Every design choice below that is
  marked *decide* goes to the user, and no flow detail is filled in from memory: confirm
  it in the docs linked here.
- **The user writes the Go code.** The agent explains the flow, cites the spec section,
  proposes tests and reviews. Go library choice follows
  `../go-development/references/choosing-packages.md`.
- **Subagents:** load the global `subagents` skill; pass `agentScope: "both"` so this
  repo's `.pi/agents/` are used. Use parallel `researcher` agents to re-check the Google
  docs and RFCs before building.
  **Undecided choices:** [`../../shared/open-questions.md`](../../shared/open-questions.md).

## Decided already (do not re-open without the user)

- D4: Google OIDC; users keyed by `oidc_issuer` + `oidc_subject`, never by email.
- D28: server-side sessions in a `sessions` table, referenced by an `HttpOnly` cookie.
- D30: single origin `https://events.poulomi-subhayan.click`; API under `/api`; cookie
  `__Host-session; Secure; HttpOnly; SameSite=Lax; Path=/`, no `Domain`; callback
  `https://events.poulomi-subhayan.click/api/auth/callback`.
- D31: exact `Origin` check on mutating requests.
- Ticket 03: routes `GET /api/auth/login` and `GET /api/auth/callback`, separate from
  `POST /api/auth/dev-login`. Dev and Google logins share one session-creation path. The
  dev login compiles only with the `dev` build tag and also requires `APP_ENV=development`.
- D27: no Google token storage in v1; only identity is needed.

## Authoritative docs

| Source | Use it for |
|---|---|
| [Google: OpenID Connect](https://developers.google.com/identity/openid-connect/openid-connect) | Flow, parameters, ID-token validation, claims |
| [Google discovery document](https://accounts.google.com/.well-known/openid-configuration) | Endpoints; fetch at runtime, respect cache headers |
| [OpenID Connect Core 1.0](https://openid.net/specs/openid-connect-core-1_0.html) | ID-token validation rules (§3.1.3.7), nonce |
| [RFC 9700 OAuth 2.0 Security BCP](https://www.rfc-editor.org/rfc/rfc9700.html) | PKCE for confidential clients, CSRF, redirect handling |
| [RFC 7636 PKCE](https://www.rfc-editor.org/rfc/rfc7636.html) | `code_verifier` / `S256` |
| [MDN Set-Cookie](https://developer.mozilla.org/en-US/docs/Web/HTTP/Reference/Headers/Set-Cookie) | `__Host-` prefix rules, `SameSite` |
| [OWASP Session Management](https://cheatsheetseries.owasp.org/cheatsheets/Session_Management_Cheat_Sheet.html) | Session ID entropy, rotation, expiry |

## Facts from Google's docs (verified 2026-10-29)

- Hard-code only the discovery URL; read `authorization_endpoint`, `token_endpoint` and
  `jwks_uri` from it.
- Validate the ID token: signature against `jwks_uri` keys; `iss` equals
  `https://accounts.google.com` **or** `accounts.google.com`; `aud` equals the client ID;
  `exp` not passed; `hd` only if you requested it.
- `sub` never changes: use it as the identifier (matches D4). `email` can change and may
  not be unique, so store it as profile data only.
- `nonce` is required by Google and protects against replay; `state` is "strongly
  recommended" and must match what the app issued.
- `redirect_uri` must match a Console-registered URI exactly (scheme, case, trailing slash).
- The `tokeninfo` endpoint is for debugging only; validate locally in production.

## Candidate Go libraries (research only, **not adopted**)

Found during research; adopting either requires the package-choice procedure and the
user's approval.
- `golang.org/x/oauth2` v0.37.0: `Config.AuthCodeURL`, `Exchange`, PKCE helpers
  `GenerateVerifier`, `S256ChallengeOption`, `VerifierOption`.
  [docs](https://pkg.go.dev/golang.org/x/oauth2)
- `github.com/coreos/go-oidc/v3` v3.21.0: `NewProvider` (discovery), `Provider.Verifier`,
  `IDTokenVerifier.Verify` (signature, issuer, audience, expiry; accepts Google's
  scheme-less issuer). **`Verify` does not check `nonce`**: the caller must.
  `StaticKeySet` allows offline tests. [docs](https://pkg.go.dev/github.com/coreos/go-oidc/v3/oidc)

## To decide with the user before building

1. Where the per-login `state`, `nonce` and PKCE `code_verifier` live between `/login`
   and `/callback`: a short-lived `__Host-` cookie or a server-side row, and its lifetime.
2. The canonical `oidc_issuer` value stored, given Google can send either issuer form.
3. Session lifetime, idle timeout, rotation on login, and what logout deletes (D28).
4. Whether to require `email_verified`, and what happens when a new `sub` signs in
   (create user vs allow-list). This is a personal app: who may log in?
5. Scopes (`openid email profile`?), `prompt` / `access_type` parameters.
6. Error handling on the callback (denied consent, mismatched state, expired code):
   which page or status the user sees. Depends on the "API conventions" ticket.
7. Local development against real Google (redirect URI registration for a dev host) or
   dev login only.
8. Where the client ID and secret come from (Kubernetes Secret in production, `.env`
   locally) and their variable names.

## Steps

1. Load decisions (above) plus the current ticket; list the open items from the decide
   list that the ticket touches. Ask them, one at a time, until each has an answer.
2. Write the flow as a numbered sequence (request → redirect → callback → validation →
   user upsert → session), naming the doc section behind each step. The user confirms it.
3. Propose tests: tampered `state`, wrong `nonce`, wrong `aud`, expired token, unknown
   `kid`, scheme-less `iss`, repeat login of the same `sub` with a changed email. The user
   writes them.
4. Review the user's code against the flow and the validation list; run `go vet` and the
   tests; dispatch a `reviewer` subagent for an independent pass.
