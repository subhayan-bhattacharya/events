---
name: kubernetes-deploy
description: Deploying Life Events to Kubernetes on a self-managed VPS. Covers the container image, Deployment, Service, Traefik ingress routing for /api, TLS certificates, Secrets, probes, graceful shutdown, the goose migration Job and database roles. Use when writing or reviewing a Dockerfile, a Kubernetes or Traefik manifest or deployment config, or planning the production rollout.
metadata:
  versions-checked: "2026-10-29"
---

# Kubernetes deployment

## Ground rules

- **Ask, never assume.** Almost nothing about deployment is decided yet (see below).
  Before writing any manifest, get the user's answer on every choice it depends on.
  Confirm each field in the Kubernetes docs for the cluster's version.
- **The agent writes** Dockerfiles, manifests and deploy scripts; **the user reviews**
  every file. Go changes the deployment needs (signal handling, `PORT`) are written by
  the user (see `go-development`).
- **Subagents:** load the global `subagents` skill; pass `agentScope: "both"` so this
  repo's `.pi/agents/` are used.
  **Undecided choices:** [`../../shared/open-questions.md`](../../shared/open-questions.md).

## Decided already

- D2: self-managed VPS, Kubernetes. D29: goose runs as a Kubernetes **Job** before each
  deploy. D30: one host `events.poulomi-subhayan.click`, frontend at `/`, API at `/api`,
  same origin, TLS.
- **Traefik is the ingress controller** (user, 2026-10-29). For any routing, TLS or
  Traefik config work, read [references/traefik.md](references/traefik.md): version,
  version-pinned docs, project-specific facts, and the Traefik choices still open.
- Map notes (`.scratch/api-db-layer/map.md`, "Not yet specified → Deployment"): separate
  DB users `events_migrator` (owns schema, used only by the Job) and `events_app` (DML
  only, used by the API); the `postgres` superuser is local-only. Server hardening still
  to build: `ReadHeaderTimeout`, graceful shutdown on SIGTERM, `PORT`.

## Not decided: grill these first

The map says to grill deployment when the chunks reach it. Load the global `grilling`
skill and settle, one question at a time: VPS provider; distribution (k3s, kubeadm, ...)
and version; the routing API and install method for Traefik; TLS issuance (Traefik ACME
or cert-manager);
where PostgreSQL runs (in-cluster StatefulSet, operator, or outside) plus backups;
Secret management; image registry; CI/CD; observability. Record answers where the
project records decisions.

## Version and docs

Find the cluster version with `kubectl version` and read the matching docs:
`https://kubernetes.io/docs/` is the current release (v1.37 on 2026-10-29); older releases
live at `https://v1-36.docs.kubernetes.io/` and so on. Version support:
[releases](https://kubernetes.io/releases/) ·
[skew policy](https://kubernetes.io/releases/version-skew-policy/).

| Topic | Docs |
|---|---|
| Job (migrations) | [Job](https://kubernetes.io/docs/concepts/workloads/controllers/job/) · [TTL after finished](https://kubernetes.io/docs/concepts/workloads/controllers/ttlafterfinished/) |
| Deployment | [Deployment](https://kubernetes.io/docs/concepts/workloads/controllers/deployment/) |
| Probes | [liveness/readiness/startup](https://kubernetes.io/docs/concepts/configuration/liveness-readiness-startup-probes/) |
| Shutdown | [pod lifecycle: termination](https://kubernetes.io/docs/concepts/workloads/pods/pod-lifecycle/) · [lifecycle hooks](https://kubernetes.io/docs/concepts/containers/container-lifecycle-hooks/) |
| Routing | [Ingress](https://kubernetes.io/docs/concepts/services-networking/ingress/) (API frozen) · [Gateway API](https://kubernetes.io/docs/concepts/services-networking/gateway/) · Traefik: [references/traefik.md](references/traefik.md) |
| Secrets | [Secret](https://kubernetes.io/docs/concepts/configuration/secret/) |
| Pod security | [Pod Security Standards](https://kubernetes.io/docs/concepts/security/pod-security-standards/) |
| k3s (if chosen) | [docs.k3s.io](https://docs.k3s.io/) |
| TLS (if chosen) | [cert-manager](https://cert-manager.io/docs/) |
| Go image | [Docker Go guide](https://docs.docker.com/guides/golang/) · [multi-stage builds](https://docs.docker.com/build/building/multi-stage/) · [distroless](https://github.com/GoogleContainerTools/distroless) |

## Facts that shape the manifests (verified 2026-10-29)

- **ingress-nginx is retired:** no releases or security fixes after March 2026
  ([announcement](https://kubernetes.io/blog/2025/11/11/ingress-nginx-retirement/)).
  Not relevant now that Traefik is chosen, but older guides assume ingress-nginx and its
  `nginx.ingress.kubernetes.io/*` annotations, which Traefik ignores. The Ingress API
  itself is GA but frozen; Kubernetes recommends Gateway API. Traefik supports both, and
  which one to use is open question 11.
- goose's CLI takes no lock, so the migration Job must be the only runner. Keep its
  `backoffLimit` low and set `activeDeadlineSeconds`. A Job's pod template is
  effectively immutable (the Job docs allow only a few scheduling fields to change, on
  suspended Jobs), so each deploy creates a new Job: unique name, or delete first.
- The migration Job uses `events_migrator` credentials; the API Deployment uses
  `events_app`. Never mount the superuser password into the cluster.
- Readiness can use `GET /api/healthz` (returns 503 when the DB ping fails). Whether
  liveness should also depend on the DB is a design question: ask. A DB-dependent
  liveness probe restarts pods during a DB outage.
- On termination Kubernetes sends SIGTERM, then SIGKILL after
  `terminationGracePeriodSeconds`. The API currently ignores SIGTERM; graceful shutdown
  (`http.Server.Shutdown`) is a pending Go change for the user.
- The `__Host-session` cookie requires HTTPS end-to-end from the browser's view, so TLS
  must terminate at the edge for the exact host (D30).
- k3s bundles Traefik, ServiceLB and local-path storage; its stable channel was
  v1.36.5+k3s1 on 2026-10-29. Relevant only if k3s is chosen (Traefik-specific k3s notes
  are in `references/traefik.md`).

## Steps

1. Confirm which decisions above are settled; grill the rest (`grilling` skill) before
   writing anything that depends on them.
2. Write the smallest manifest set for the agreed scope, each field traced to a doc page
   for the cluster's version. Use a `worker` subagent for a large self-contained set and
   review its diff.
3. Validate without touching a cluster when possible (`kubectl apply --dry-run=server`
   needs a cluster, `--dry-run=client` does not). Ask before running anything against a
   real cluster.
4. Hand the files to the user for review with a list of the decisions each one encodes.
