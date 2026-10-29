#!/usr/bin/env bash
# Installs the global subagent setup the project skills rely on:
#   - Pi's example `subagent` extension  -> ~/.pi/agent/extensions/subagent/
#   - the `subagents` skill             -> ~/.agents/skills/subagents/
#   - generic scout/researcher/reviewer/worker agents -> ~/.pi/agent/agents/
# Existing files that differ are left alone unless --force is given.
set -euo pipefail

FORCE=0
[[ "${1:-}" == "--force" ]] && FORCE=1

HERE="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

if ! command -v pi >/dev/null 2>&1; then
  echo "error: 'pi' not found on PATH. Install Pi first: https://pi.dev" >&2
  exit 1
fi

NPM_ROOT="$(npm root -g 2>/dev/null || true)"
EXAMPLE="$NPM_ROOT/@earendil-works/pi-coding-agent/examples/extensions/subagent"
if [[ ! -f "$EXAMPLE/index.ts" ]]; then
  echo "error: Pi's subagent example not found at $EXAMPLE" >&2
  echo "       (is Pi installed globally with npm?)" >&2
  exit 1
fi

install_file() { # src dest
  local src="$1" dest="$2"
  mkdir -p "$(dirname "$dest")"
  if [[ -e "$dest" ]] && ! cmp -s "$src" "$dest"; then
    if [[ $FORCE -eq 0 ]]; then
      echo "skip (differs, use --force): $dest"
      return
    fi
    cp "$dest" "$dest.bak"
    echo "backup: $dest.bak"
  fi
  cp "$src" "$dest"
  echo "installed: $dest"
}

install_file "$EXAMPLE/index.ts"  "$HOME/.pi/agent/extensions/subagent/index.ts"
install_file "$EXAMPLE/agents.ts" "$HOME/.pi/agent/extensions/subagent/agents.ts"
install_file "$HERE/skills/subagents/SKILL.md" "$HOME/.agents/skills/subagents/SKILL.md"
for f in "$HERE"/pi-agents/*.md; do
  install_file "$f" "$HOME/.pi/agent/agents/$(basename "$f")"
done

echo
echo "Done. Restart Pi (or run /reload) and trust this project when asked."
