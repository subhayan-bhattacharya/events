# Traefik (ingress controller)

Decided by the user on 2026-10-29: **Traefik** routes traffic into the cluster. Not yet
decided: which routing API, how it is installed, and how TLS certificates are issued
(see "Still to decide").

## Version

Latest release on 2026-10-29: **Traefik v3.7.16**; Helm chart **v42.1.3**. Only 3.7 has
active support ([support table](https://doc.traefik.io/traefik/v3.7/deprecation/releases/));
from 3.6 on, each minor gets 6 months of support. Before relying on a behaviour, find the
version actually running (the image tag of the Traefik Deployment, e.g.
`kubectl -n <ns> get deploy traefik -o jsonpath='{..image}'`) and open the docs for that
minor at `https://doc.traefik.io/traefik/v<major.minor>/`. The unversioned
`doc.traefik.io/traefik/` is always the newest. Release notes:
[traefik releases](https://github.com/traefik/traefik/releases) ·
[chart releases](https://github.com/traefik/traefik-helm-chart/releases).

## Docs (v3.7)

| Topic | Link |
|---|---|
| Start here | [Kubernetes getting started](https://doc.traefik.io/traefik/v3.7/getting-started/kubernetes/) · [expose: basic](https://doc.traefik.io/traefik/v3.7/expose/kubernetes/basic/) · [expose: advanced](https://doc.traefik.io/traefik/v3.7/expose/kubernetes/advanced/) |
| Helm chart | [repo](https://github.com/traefik/traefik-helm-chart) · [`values.yaml`](https://github.com/traefik/traefik-helm-chart/blob/master/traefik/values.yaml) · [Artifact Hub](https://artifacthub.io/packages/helm/traefik/traefik) |
| Provider: Kubernetes Ingress | [install config](https://doc.traefik.io/traefik/v3.7/reference/install-configuration/providers/kubernetes/kubernetes-ingress/) · [routing / annotations](https://doc.traefik.io/traefik/v3.7/reference/routing-configuration/kubernetes/ingress/) |
| Provider: Traefik CRDs | [install config](https://doc.traefik.io/traefik/v3.7/reference/install-configuration/providers/kubernetes/kubernetes-crd/) · [IngressRoute](https://doc.traefik.io/traefik/v3.7/reference/routing-configuration/kubernetes/crd/http/ingressroute/) · [Middleware CRD](https://doc.traefik.io/traefik/v3.7/reference/routing-configuration/kubernetes/crd/http/middleware/) |
| Provider: Gateway API | [install config](https://doc.traefik.io/traefik/v3.7/reference/install-configuration/providers/kubernetes/kubernetes-gateway/) · [routing](https://doc.traefik.io/traefik/v3.7/reference/routing-configuration/kubernetes/gateway-api/) · [Gateway API spec](https://gateway-api.sigs.k8s.io/) |
| Entrypoints (ports, redirects, timeouts, forwarded headers) | [entrypoints](https://doc.traefik.io/traefik/v3.7/reference/install-configuration/entrypoints/) |
| Router rules and priority | [rules-and-priority](https://doc.traefik.io/traefik/v3.7/reference/routing-configuration/http/routing/rules-and-priority/) |
| Middlewares | [overview](https://doc.traefik.io/traefik/v3.7/reference/routing-configuration/http/middlewares/overview/) · [headers](https://doc.traefik.io/traefik/v3.7/reference/routing-configuration/http/middlewares/headers/) · [redirectScheme](https://doc.traefik.io/traefik/v3.7/reference/routing-configuration/http/middlewares/redirectscheme/) · [stripPrefix](https://doc.traefik.io/traefik/v3.7/reference/routing-configuration/http/middlewares/stripprefix/) |
| TLS | [ACME resolver](https://doc.traefik.io/traefik/v3.7/reference/install-configuration/tls/certificate-resolvers/acme/) · [TLS certificates](https://doc.traefik.io/traefik/v3.7/reference/routing-configuration/http/tls/tls-certificates/) · [cert-manager HTTP-01](https://cert-manager.io/docs/configuration/acme/http01/) |
| Dashboard, health, logs | [API & dashboard](https://doc.traefik.io/traefik/v3.7/reference/install-configuration/api-dashboard/) · [healthcheck](https://doc.traefik.io/traefik/v3.7/reference/install-configuration/observability/healthcheck/) · [logs & access logs](https://doc.traefik.io/traefik/v3.7/reference/install-configuration/observability/logs-and-accesslogs/) |
| Upgrades | [v3 migration notes](https://doc.traefik.io/traefik/v3.7/migrate/v3/) |
| k3s's bundled Traefik (only if k3s is chosen) | [k3s networking: Traefik](https://docs.k3s.io/networking/networking-services) · [packaged components](https://docs.k3s.io/installation/packaged-components) · [k3s Helm / HelmChartConfig](https://docs.k3s.io/add-ons/helm) · [k3s Traefik chart](https://github.com/k3s-io/k3s-charts/tree/main/charts/traefik) |

## Facts that matter for this project (verified 2026-10-29 in the v3.7 docs)

- **Three ways to define routes**, all supported by Traefik 3.7: the standard Kubernetes
  `Ingress` (with `traefik.ingress.kubernetes.io/*` annotations), Traefik's own
  `IngressRoute` CRD (`traefik.io/v1alpha1`), or Gateway API (`HTTPRoute`). The 3.7
  Gateway provider supports Gateway API **Standard v1.6.3**. Its CRDs are installed
  separately, and the provider is enabled with Helm `providers.kubernetesGateway.enabled: true`.
  Do not enable `experimentalChannel` without the Experimental CRDs: the whole Gateway
  provider then fails to start.
- **Keep the `/api` prefix.** The Go server registers routes with the prefix
  (`GET /api/healthz`), so route `PathPrefix(/api)` to the API Service with **no**
  `stripPrefix` middleware.
- **`/api` vs `/` on one host (D30):** by default Traefik orders routers by rule length,
  longest first, so `Host(...) && PathPrefix(/api)` outranks `Host(...)`. Set explicit
  priorities if rules ever change shape.
- **HTTP→HTTPS:** redirect at the `web` entrypoint
  (`entryPoints.web.http.redirections.entryPoint.to=websecure`, `scheme=https`,
  permanent by default). `__Host-session` requires HTTPS (D30).
- **TLS with Traefik's built-in ACME** stores certificates in a file (`acme.json`, needs
  a persistent volume) and is safe with **one Traefik replica only**. For several
  replicas, Traefik's docs point to cert-manager (certificates as Kubernetes Secrets).
  HTTP-01 needs port 80 reachable; TLS-ALPN-01 needs 443; wildcards need DNS-01.
- **Forwarded headers:** `X-Forwarded-*` from clients are trusted only from
  `forwardedHeaders.trustedIPs` (unset by default; `insecure` is for tests only). Relevant
  if the API ever uses client IPs (logging, rate limits).
- **Entrypoint timeouts** default to `readTimeout` 60s, `writeTimeout` 0 (none),
  `idleTimeout` 180s. They are separate from the Go server's own timeouts.
- **D31 Origin check:** the API compares the `Origin` header exactly. After deployment,
  verify with a real request through Traefik that `Origin` arrives unchanged (no
  headers middleware rewriting it).
- **Dashboard:** keep it off the public host; ask before exposing it anywhere.
- **If k3s is chosen:** k3s ships Traefik v3 as a packaged component in `kube-system`,
  customised only through a `HelmChartConfig` (never by editing its `traefik.yaml`). Its
  Traefik version follows the k3s release, and k3s documents Gateway API v1.4 for its
  bundled Traefik. Chart v40 (k3s June 2026 releases) moved `service.type` to
  `service.spec.type`. `--disable=traefik` removes the bundled copy if you install
  Traefik yourself.

## Still to decide (ask, one at a time)

1. Routing API: `Ingress`, `IngressRoute` CRD, or Gateway API (open question 11).
2. Installation: upstream Helm chart (pinned chart version) or k3s's bundled copy, if
   k3s is chosen.
3. Certificates: Traefik ACME (single replica, persistent volume) or cert-manager; which
   ACME challenge; the contact email.
4. Replica count for Traefik itself.
5. Whether the dashboard is enabled at all, and how it is reached (e.g. port-forward only).
