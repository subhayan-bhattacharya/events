# Open technology questions

Found while researching the stack on 2026-10-29. None is decided. When work touches one
of these, **raise it with the user and wait for an answer**; do not pick for them. When the
user decides, record the answer where the project records decisions (a ticket's
`## Answer`, or the decision log) and delete the entry here.

1. **Standard library `uuid` vs `github.com/google/uuid`.** Go 1.27 added package
   [`uuid`](https://pkg.go.dev/uuid) (`New`, `NewV4`, `NewV7`, `Parse`; no `NullUUID`).
   `sqlc.yaml` currently overrides `uuid` to `github.com/google/uuid`. sqlc's *latest*
   docs show stdlib overrides and say a future sqlc will default to it; the v1.31.1 docs
   do not mention it. Touches: `api/sqlc.yaml`, every generated struct.
2. **`encoding/json/v2`.** GA in Go 1.27 ([notes](https://go.dev/doc/go1.27#jsonv2)).
   v1 `encoding/json` now runs on the v2 implementation; v2 rejects invalid UTF-8 and
   duplicate object names. Belongs to the "API conventions" ticket.
3. **goose v3.28.0 → v3.29.0.** v3.29.0 only adds a YDB option and dependency bumps
   ([releases](https://github.com/pressly/goose/releases)). Upgrade or stay.
4. **pgx 5.11 `timestamptz` location.** Values scanned in text format now come back in
   `time.Local` (binary always did). Decide whether to pin the codec's `ScanLocation` to
   `time.UTC` or always convert with `.In(loc)` before date logic (D10, D25, D32).
   [CHANGELOG](https://github.com/jackc/pgx/blob/master/CHANGELOG.md).
5. **Repository tests and service transactions.** Tests plan "rollback transaction per
   test" (ticket 02), while services own transactions through `InTx` (ticket 01). In pgx,
   `Tx.Begin` on an existing transaction creates a savepoint, which allows both, but
   `Pool.Begin` inside `InTx` would escape the test transaction. Sequences never roll
   back. Decide when the first repository test is wired.
6. **`httptest.NewTestServer`** (Go 1.27): in-memory test server that works with
   `testing/synctest`. Adopt for handler tests or keep `httptest.NewRecorder`.
7. **PostgreSQL 17 vs 18.** The project pins 17 (supported to 2029-11-08, current minor
   17.11); 18.6 is current. `postgres:17-alpine` floats to the latest 17.x minor; decide
   whether to pin a minor tag.
8. **Kubernetes distribution and version.** Upstream stable is v1.37.2; k3s stable channel
   is v1.36.5+k3s1 (2026-10-29). Not chosen; part of the deployment grilling in the map.
9. **Nullable column Go types.** With pgx/v5, nullable non-overridden columns become
   `pgtype.*` (e.g. `pgtype.Text`). sqlc's `emit_pointers_for_null_types` would make them
   pointers. Not yet configured either way.
10. **`CREATE EXTENSION pgcrypto`** in the database model. PostgreSQL 17 has
    `gen_random_uuid()` in core ([docs](https://www.postgresql.org/docs/17/functions-uuid.html));
    pgcrypto's copy is marked obsolete. Keep the extension (e.g. for other pgcrypto
    functions) or drop it from the first migration.
11. **Routing API for Traefik.** Controller decided: **Traefik** (user, 2026-10-29). Still
    open: route with the standard `Ingress`, Traefik's `IngressRoute` CRD, or
    [Gateway API](https://kubernetes.io/docs/concepts/services-networking/gateway/) (the
    Ingress API is GA but frozen; Kubernetes recommends Gateway API). Plus Traefik's
    install method and certificate issuance. Details and links:
    `.agents/skills/kubernetes-deploy/references/traefik.md`.
