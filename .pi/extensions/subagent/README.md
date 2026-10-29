# subagent extension (project copy)

`index.ts` and `agents.ts` are copied unchanged from Pi's example extension
(`@earendil-works/pi-coding-agent` 1.4.0, `examples/extensions/subagent`, MIT licence).
It registers a `subagent` tool that runs each agent in a separate `pi` process with its
own context window. Pi loads it from `.pi/extensions/` once the project is trusted.

Project agents live in `../../agents/*.md`. The tool only sees them when it is called
with `agentScope: "project"` or `"both"`.

To update after a Pi upgrade, copy the two files again from
`$(npm root -g)/@earendil-works/pi-coding-agent/examples/extensions/subagent/`.

How the skills use it: `.agents/shared/subagents.md`.
