---
name: project-journal
description: Maintain the Life Events HTML project journal after project discussions, decisions, implementations, verifications, or changes to the next step. Use at the end of any project work session and when asked for project status or history.
---

# Project journal

Maintain `docs/project-journal.html` as a **readable historical record**, not a replacement for the decision log or `.scratch/api-db-layer/map.md` and tickets. Follow `AGENTS.md` for when to run this skill.

1. **Gather evidence.** Read the relevant changed files, Git diff/status, `project.md`, the map, touched tickets and the database decision log. For a status question, check the current frontier, including claimed and blocked tickets. Use only the conversation you can actually see; older chats are not recoverable unless recorded in files. Done when every fact to be added has a known source.
2. **Update the HTML.** Maintain the chronological progress log, current-state/next-step section, component sections, decisions and open questions in `docs/project-journal.html`. Record what was discussed, what was actually built or verified, how problems were resolved, alternatives considered, and *why* a choice was made. Link to source files and ticket numbers using relative links; include dates only when evidenced. Label proposals, unverified claims and open questions explicitly. When changing a decision, preserve the earlier choice in the history and mark its supersession rather than silently overwriting it. Done when the page answers “where are we?”, “what next?”, and “why this design?” without implying planned work is finished.
3. **Check consistency.** Compare the page against the current ticket statuses and code, check that links point to files that exist, and ensure the HTML is valid enough to open locally. If a detail is missing, record it as unknown rather than filling it in. The map's one-ticket-per-session protocol still governs work on the map; journal upkeep does not claim or resolve tickets. Done when the page reflects the actual repo and the latest discussion.

Keep credentials, tokens, private `.env` values, and personal data out of the journal. Do not claim that all historical conversations were archived: backfill is limited to written evidence. When finishing a project session, mention the journal update briefly alongside the main result.
