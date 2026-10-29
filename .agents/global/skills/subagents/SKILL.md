---
name: subagents
description: Delegate work to subagents through the subagent tool, which runs agents in isolated contexts singly, in parallel or chained. Use when a task would flood the context, splits into independent parts such as researching several libraries or questions at once, needs codebase recon before planning, or needs a review unbiased by the conversation that produced the work.
---

# Subagents

A subagent runs in its own `pi` process with a fresh context window and returns only its
final answer. Keep the main thread for talking with the user and making decisions; push
digging, parallel research and independent review out to subagents.

## The tool

The global extension `~/.pi/agent/extensions/subagent/` registers the `subagent` tool.
If the tool is missing from your tools, tell the user (the extension may have failed to
load; `/reload` re-reads it) and ask how to proceed rather than silently doing the work
inline.

Agents are Markdown files with frontmatter (`name`, `description`, optional `tools`,
optional `model`; without `model` the agent inherits the current session's model):

- **Global** `~/.pi/agent/agents/`: always visible.
- **Project** `<repo>/.pi/agents/`: visible only with `agentScope: "both"` (or
  `"project"`). A project agent overrides a global one with the same name. **If the
  current repo has `.pi/agents/`, pass `agentScope: "both"` on every call**; project
  agents usually carry that repo's rules.

Global agents:

| Agent | Use for | Writes files |
|---|---|---|
| `scout` | Mapping the code, docs and decisions relevant to a task | No |
| `researcher` | One docs/API/package question, version-matched, several sources | No |
| `reviewer` | Reviewing a diff against the repo's documented rules and the task | No |
| `worker` | A delegated, self-contained writing task | Yes |

Modes:
- single `{ agent, task, agentScope }`
- parallel `{ tasks: [{ agent, task }, ...], agentScope }` (max 8 tasks, 4 running at once)
- chain `{ chain: [{ agent, task }, ...], agentScope }`, where `{previous}` in a task
  inserts the previous step's output

## When to dispatch

- **Parallel research:** several independent questions, e.g. one `researcher` per
  candidate library, or per library whose docs a plan depends on.
- **Recon before planning:** a `scout` when the relevant material spans many files.
- **Independent review:** a `reviewer` after a chunk of work, so the review isn't shaped
  by the conversation that produced it.
- **Isolated writing:** a `worker` for a self-contained file set, then review its diff
  yourself before reporting.

Stay inline for anything one `read` or `grep` answers. Dispatching costs a whole process.

## Writing the task

A subagent sees only its task string, not this conversation. Every task states:
1. the goal and the exact question or deliverable;
2. the files, versions and docs it must use (with paths), and any skill it should load;
3. what it may and may not change;
4. the output shape you need back;
5. "If anything is ambiguous, report it as a question instead of choosing."

## Handling results

- Questions a subagent reports go to the user. Never answer them on the user's behalf.
- Verify claims that a decision rests on (open the cited file or URL) before passing
  them on as fact.
- Run parallel tasks only when they don't write the same files.
