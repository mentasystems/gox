#!/usr/bin/env bash
# PostToolUse hook for Claude Code: when a .go file is edited, run
# `gox check .` in its directory. If issues are found, return them as a
# blocking JSON object so the model sees them in its next turn.
#
# Installed by: gox install claude
# Schema reference: https://docs.anthropic.com/en/docs/claude-code/hooks

set -u

PAYLOAD=$(cat)
FILE=$(printf '%s' "$PAYLOAD" | jq -r '.tool_input.file_path // empty')

case "$FILE" in
  *.go) ;;
  *) exit 0 ;;
esac

DIR=$(dirname "$FILE")
[ -d "$DIR" ] || exit 0

GOX="${GOX_BIN:-${HOME}/go/bin/gox}"
[ -x "$GOX" ] || exit 0

OUT=$(cd "$DIR" && "$GOX" check . 2>&1)
STATUS=$?

if [ "$STATUS" -eq 0 ] && [ -z "$OUT" ]; then
  exit 0
fi

jq -nR --arg msg "$OUT" '{
  decision: "block",
  reason: ("gox found issues in the package you just edited (fix or annotate with // safe-ignore: / // global-ok: / // any-ok: / // goroutine-ok: / // exhaustive-ok: as appropriate):\n\n" + $msg)
}'
