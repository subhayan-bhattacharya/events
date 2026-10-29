---
name: feature-planning
description: Plan work for Life Events. Decides whether and how to build a feature, scopes it against project decisions, picks the next piece of work, and selects the global and project skills and subagents the work needs. Use when the user wants to plan, scope or decide a feature, asks what to do next, or asks which skills or approach fit a task.
---

# Feature planning

Produces an agreed **plan**: what will be built, what will not, which decisions are still
needed, which skills and subagents to use in which order, and who writes what. The plan
is a proposal until the user approves it.

## Ground rules

- **Ask, never assume.** Restate the request and confirm it. Ask about every gap in
  scope, behaviour or priority, one question at a time, with your recommendation. An
  unanswered question stays open in the plan; it is never quietly filled in.
- **Ownership:** the user writes Go code (tests included) unless they say otherwise; the
  agent writes SQL, Dockerfiles, manifests and docs for the user to review.
- **Subagents:** load the global `subagents` skill. This repo has project agents in
  `.pi/agents/` (Life Events versions of `scout`, `researcher`, `reviewer`, `worker`), so
  pass `agentScope: "both"` on every call.
  **Undecided technology choices:** [`../../shared/open-questions.md`](../../shared/open-questions.md).

## Steps

1. **Understand the request.** Restate the goal in one or two sentences, plus what is
   explicitly out. Done when the user confirms the restatement.

2. **Load project context.** Read `project.md`, the decision log in
   `Complete PostgreSQL Database Model for the Life Events App.md`, and
   `.scratch/api-db-layer/map.md` with any tickets the request touches. If the request is
   work on the map, **the map's session protocol governs** (one ticket per session, its
   claim/resolve steps, its reply-length rule) and this skill only supplies skill and
   subagent selection. For wide requests dispatch a `scout` subagent. Done when you can
   list the D-numbers and tickets that constrain the request, and its blocking tickets.

3. **Inventory the skills.** Model-invoked skills (global and project) are already
   listed in your context by name and description. User-invoked ones are not, so list
   them:
   ```bash
   for f in ~/.agents/skills/*/SKILL.md ~/.pi/agent/skills/*/SKILL.md .agents/skills/*/SKILL.md; do
     grep -q 'disable-model-invocation: true' "$f" && echo "user-invoked: $(basename "$(dirname "$f")")"
   done | sort -u
   ```
   Read the descriptions of the candidates. Done when each candidate has a verdict:
   **use** (model-invoked: load it), **ask the user to run `/skill:<name>`** (user-invoked:
   you cannot fire it yourself), or **not needed**, each with one reason.

4. **Match the work to skills.** Hints below; confirm each skill still exists and that
   its description fits before relying on it.

   | Situation | Skill |
   |---|---|
   | Effort larger than one session, new map of decisions | `wayfinder` (user-invoked) |
   | Stress-test a plan or decision | `grilling`; `grill-with-docs` (user-invoked) also records ADRs and glossary |
   | Terminology, `CONTEXT.md`, ADRs | `domain-modeling` |
   | Unknown facts, docs, versions | `research`, or parallel `researcher` subagents |
   | A design question a throwaway experiment answers | `prototype` |
   | Module interfaces, seams, testability | `codebase-design` |
   | Conversation → spec or tickets | `to-spec`, `to-tickets` (user-invoked; check which tracker they publish to before use, since this repo keeps tickets in `.scratch/`) |
   | Build test-first | `tdd` |
   | Something broken or slow | `diagnosing-bugs` |
   | Review a branch or diff | `code-review` (it expects `docs/agents/issue-tracker.md`; ask how to proceed if missing) |
   | Steps only the user can do (Google Cloud Console, DNS, VPS) | `wizard` |
   | Go code or a Go library choice | `go-development` |
   | Schema, migrations, queries | `postgres-sql` |
   | Login, sessions | `google-oidc` |
   | Dockerfile, manifests, rollout | `kubernetes-deploy` |
   | Writing or editing skills, `AGENTS.md` | `writing-for-agents` |
   | Not sure which flow fits | `ask-matt` (user-invoked router); `find-skills` to install new ones |

   Skills aimed at other stacks (TypeScript, Husky, Neon, exercise scaffolding, other
   projects' databases) do not apply unless the user says otherwise.

5. **Check the feature against the project.** For each part of the request, record:
   the decisions it relies on or conflicts with (D-numbers, resolved tickets); "Out of
   scope" and "Not yet specified" items in the map it touches; open technology questions
   it touches; the technology skills it needs; tickets it is blocked by. A conflict with a
   settled decision goes to the user as a question, with no workaround.

6. **Propose the plan.** One message, in this shape:
   - **Goal / out of scope**
   - **Questions for you**: numbered, each with your recommendation
   - **Skills, in order**: name, why, and who invokes it (agent or `/skill:` by the user)
   - **Subagents**: which agent, what task, parallel or chained
   - **Who writes what**: user (Go) and agent (SQL, manifests, docs)
   - **Done when**: checkable conditions (commands pass, endpoints behave, decisions recorded)

   Done when the user approves the plan or the questions are answered and the plan revised.

7. **Record it where the user wants.** Ask whether the plan becomes a ticket in
   `.scratch/api-db-layer/issues/`, an update to an existing ticket, or stays in the
   conversation. Write nothing to the tracker without that answer.
