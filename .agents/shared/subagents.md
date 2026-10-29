# Subagents in this project

A subagent runs in its own `pi` process with a fresh context window. Use one when the work
would flood the main context or can run in parallel; keep the main thread for talking
with the user and for decisions.

## The tool

The project extension `.pi/extensions/subagent/` registers the `subagent` tool. Project
agents live in `.pi/agents/`, so **every call passes `agentScope: "both"`** (without it the
tool only sees user-level agents). If the tool is missing, the project is probably not
trusted in Pi or the session predates the extension: tell the user and ask them to run
`/reload` or trust the project. Do not silently fall back to doing the work inline.

| Agent | Use for | Writes files |
|---|---|---|
| `scout` | Mapping code, decisions and tickets relevant to a task | No |
| `researcher` | One docs or package question, version-matched, multi-source | No |
| `reviewer` | Reviewing a diff against project decisions and the skills | No |
| `worker` | Delegated non-Go writing: migrations, sqlc queries, Dockerfiles, manifests | Yes |

Modes: single `{ agent, task, agentScope }`, parallel `{ tasks: [...], agentScope }`
(max 8 tasks, 4 at once), chain `{ chain: [...], agentScope }` with `{previous}` passing
the last output forward.

## When to dispatch

- **Parallel research**: several independent questions, e.g. one `researcher` per
  candidate package, or one per library whose docs a plan depends on.
- **Recon before planning**: a `scout` when the relevant code or decisions span many
  files.
- **Independent review**: a `reviewer` after a chunk is written, so the review is not
  biased by the conversation that produced the code.
- **Isolated writing**: a `worker` for a self-contained non-Go file set, then review its
  diff yourself before reporting.

Stay inline for anything a single `read` or `rg` answers.

## Writing the task

A subagent sees only its task string. Every task states:
1. the goal and the exact question or deliverable;
2. the files, versions (from `api/go.mod`) and skill paths it must use;
3. the output shape you need back;
4. "If anything is ambiguous, report it as a question instead of choosing."

Questions a subagent reports come back to the user through you; never answer them on
the user's behalf.
