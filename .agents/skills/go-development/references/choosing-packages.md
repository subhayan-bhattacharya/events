# Choosing a Go package for a task

Use this when a task needs functionality that neither the standard library nor an
existing `go.mod` dependency provides. The outcome is a recommendation the **user
approves**; nothing is added to `go.mod` before that.

## Steps

1. **State the need** in one sentence (e.g. "verify Google ID tokens"). Done when the
   user agrees it is the real need; ask if the task is vague.
2. **Standard library first.** Search [pkg.go.dev/std](https://pkg.go.dev/std) and the
   [Go 1.27 release notes](https://go.dev/doc/go1.27) (new in 1.27: `uuid`,
   `encoding/json/v2`, `crypto/mldsa`). Check `golang.org/x/...` next: maintained by the Go
   team, but still a new dependency. Done when you can say what std covers and what it
   lacks.
3. **Check what is already there:** `grep` `api/go.mod` (including `// indirect`, which is
   pulled in already but not yet a direct choice) and the project decisions
   (e.g. D5 rules out ORMs, D6 defers River).
4. **Gather candidates from several independent sources**, at least three of:
   - [pkg.go.dev search](https://pkg.go.dev/search), with the **Imported by** count on each package page
   - the official docs of the technology involved (vendors often name a Go library)
   - [awesome-go](https://github.com/avelino/awesome-go) (its
     [inclusion criteria](https://github.com/avelino/awesome-go/blob/main/CONTRIBUTING.md)
     include SemVer releases, `go.mod`, docs and test coverage)
   - [deps.dev](https://deps.dev/) for dependency graph, advisories and OpenSSF Scorecard
   - maintainer repos and issue trackers; community discussion (r/golang, Gophers Slack
     archives, well-known blogs) as supporting evidence only
   AI-summarised search answers are leads, never evidence. For several candidates,
   dispatch one `researcher` subagent per candidate in parallel.
5. **Evaluate each candidate** (2–4 of them) on:
   - fit: does it do exactly the job, with an API that suits handler → service → repository?
   - adoption: pkg.go.dev "Imported by", known users
   - maintenance: latest release date, release cadence, open issue response, Go-version
     requirement compatible with `go 1.27`
   - stability: v1+ SemVer, breaking-change history in the changelog
   - weight: transitive dependencies (deps.dev); small and focused beats frameworks
   - security: open advisories ([vuln.go.dev](https://vuln.go.dev/)), OpenSSF Scorecard
   - licence compatible with the project (ask if unsure)
6. **Present a comparison table** with a source link for every cell, a recommendation
   with reasons, and what we lose by staying with std. Then ask the user to choose.
7. **After approval only:** the user (or the agent, if asked) runs `go get <module>@<version>`
   from `api/`, then `go mod tidy`, `go build ./...`, and, if available,
   `govulncheck ./...`. Record the choice and its reasons where the project records
   decisions.
