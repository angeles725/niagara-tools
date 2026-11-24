#!/usr/bin/env bash
# issue-area.sh — the decision behind .github/workflows/issue-area.yml: every issue carries an `area:<name>` label
# (area:mcp-n4, area:build-n4-module), so the weekly follow-up sweep and the retro tooling can route it.
#
# Usage:  issue-area.sh [labels]     labels: comma- or newline-separated label names (spaces around a name are ignored)
# Env:    ISSUE_LABELS               the labels, used when no argument is given; an empty value is an empty list
# Output: stdout, one verdict:
#           ok:      `issue-area: ok (area:a area:b)`
#           missing: the comment the workflow posts (it starts with the hidden marker `<!-- issue-area -->`, which the
#                    workflow looks for to comment once)
# Exits:  0 an area label is present · 1 none (an empty list included) · 3 usage (no labels given, or extra arguments)
# An area label is `area:` plus a name of lowercase letters, digits and hyphens that starts with a letter or digit.
set -euo pipefail

MARKER='<!-- issue-area -->'

usage() { printf 'usage: issue-area.sh [labels]   (or set ISSUE_LABELS; comma- or newline-separated)\n' >&2; exit 3; }

[ "$#" -le 1 ] || usage
if [ "$#" -eq 1 ]; then
  labels="$1"
elif [ "${ISSUE_LABELS+set}" = "set" ]; then
  labels="$ISSUE_LABELS"
else
  usage
fi

areas=()
while IFS= read -r label; do
  label="${label#"${label%%[![:space:]]*}"}"
  label="${label%"${label##*[![:space:]]}"}"
  if [[ "$label" =~ ^area:[a-z0-9][a-z0-9-]*$ ]]; then areas+=("$label"); fi
done < <(printf '%s\n' "$labels" | tr ',' '\n')

if [ "${#areas[@]}" -gt 0 ]; then
  printf 'issue-area: ok (%s)\n' "${areas[*]}"
  exit 0
fi

cat <<MSG
$MARKER
This issue has no \`area:*\` label. Add one so it can be routed:

- \`area:mcp-n4\`: the mcp-n4 MCP server (\`go/cmd/mcp-n4\`, \`go/internal/mcpn4\`, \`mcp-n4-kit\`)
- \`area:build-n4-module\`: the build-n4-module kit (METHODOLOGY, toolbelt, scripts, CI)

Issues opened from the issue forms get the label on their own; an issue opened with \`gh issue create\` needs
\`--label area:<name>\`. This check runs again when the labels change.
MSG
exit 1
