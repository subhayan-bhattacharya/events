# Chunk 9: Vacations and purchases

Type: task
Status: open
Blocked by: 16

## Question

Schema, queries and endpoints for vacations and purchases.

## Done when

- Event and detail are written in one transaction; no endpoint writes a detail row alone
- Type change is delete + re-create; `kind` is never updated
- Cancel sets `status = 'CANCELLED'`; delete hard-deletes and cascades
- "Previous calendar year" and "trailing 12 months" vacation queries take `today`, and include trips crossing the boundary

## Plan decisions to check

D11, D12, D19, D22, D32
