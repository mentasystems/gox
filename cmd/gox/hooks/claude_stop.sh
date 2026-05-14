#!/usr/bin/env bash
# Stop hook for Claude Code: when the turn ends, find the .go files that
# changed (according to git) and run `gox check` once per affected package.
# If issues are found, return them as a blocking JSON object so the model
# sees them in its next turn.
#
# This used to be a PostToolUse hook (ran on every Edit/Write — too
# frequent and slow). It now runs once per turn instead.
#
# Installed by: gox install claude
# Schema reference: https://docs.anthropic.com/en/docs/claude-code/hooks

set -u

PAYLOAD=$(cat)
CWD=$(printf '%s' "$PAYLOAD" | jq -r '.cwd // empty')
[ -n "$CWD" ] || CWD="$PWD"
[ -d "$CWD" ] || exit 0

GOX="${GOX_BIN:-${HOME}/go/bin/gox}"
[ -x "$GOX" ] || exit 0

# Resolve git root; bail quietly if not a repo.
ROOT=$(cd "$CWD" && git rev-parse --show-toplevel 2>/dev/null) || exit 0

# Collect changed/staged/untracked .go files.
FILES=$(
  cd "$ROOT" && {
    git diff --name-only -- '*.go'
    git diff --name-only --cached -- '*.go'
    git ls-files --others --exclude-standard -- '*.go'
  } 2>/dev/null | sort -u
)
[ -n "$FILES" ] || exit 0

# Unique directories of those files.
DIRS=$(
  while IFS= read -r f; do
    [ -n "$f" ] || continue
    [ -f "$ROOT/$f" ] || continue
    dirname "$ROOT/$f"
  done <<< "$FILES" | sort -u
)
[ -n "$DIRS" ] || exit 0

OUT=""
while IFS= read -r d; do
  [ -d "$d" ] || continue
  RES=$(cd "$d" && "$GOX" check . 2>&1)
  if [ -n "$RES" ]; then
    OUT="${OUT}${OUT:+$'\n\n'}${RES}"
  fi
done <<< "$DIRS"

[ -n "$OUT" ] || exit 0

jq -nR --arg msg "$OUT" '{
  decision: "block",
  reason: ("gox found issues in the packages you edited this turn (fix or annotate with // safe-ignore: / // global-ok: / // any-ok: / // goroutine-ok: / // exhaustive-ok: as appropriate):\n\n" + $msg)
}'
