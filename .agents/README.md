# Agent skills for Life Events

Project skills follow the [Agent Skills specification](https://agentskills.io/specification).
Pi discovers `.agents/skills/` automatically once you trust the project; the agent sees
only each skill's name and description until a task needs the full skill (type
`/skill:<name>` to force one).

| Skill | Use for |
|---|---|
| `feature-planning` | Planning and scoping features; picks the global and project skills and subagents a task needs |
| `go-development` | Go 1.27 code in `api/`: net/http, pgx, sqlc, goose, testcontainers, tests, choosing Go packages |
| `postgres-sql` | PostgreSQL 17 migrations and queries |
| `google-oidc` | Google login, sessions |
| `kubernetes-deploy` | Dockerfile, Kubernetes manifests, Traefik routing and TLS |

Also in this folder:
- `shared/open-questions.md`: technology choices not yet decided. The skills raise them
  instead of picking an answer.
- `global/`: the subagent setup, which is installed per user, not per project (below).

Working agreements the skills enforce: the agent asks rather than assumes; Go code is
written by the developer unless they ask otherwise; the agent writes SQL, Dockerfiles and
manifests for review.

## One-time setup for each developer

The skills tell the agent to load the global `subagents` skill and call the `subagent`
tool. Install both once per machine:

```bash
.agents/global/install.sh            # add --force to overwrite differing files (backups kept)
```

This installs:
- Pi's example `subagent` extension (MIT) into `~/.pi/agent/extensions/subagent/`
- the `subagents` skill into `~/.agents/skills/subagents/`
- generic `scout`, `researcher`, `reviewer` and `worker` agents into `~/.pi/agent/agents/`

This repo's own versions of those agents are in `.pi/agents/`, and the skills call the
tool with `agentScope: "both"`, so inside this project they replace the generic ones.

After a Pi upgrade, run the script with `--force` to refresh the extension. Changes to
the `subagents` skill or the generic agents go in `.agents/global/` first, then run the
script.
