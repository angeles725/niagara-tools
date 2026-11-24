#!/usr/bin/env bats
bats_require_minimum_version 1.5.0
# issue-area.sh — the decision behind .github/workflows/issue-area.yml: an issue needs an `area:<name>` label.
# Labels come from the first argument or ISSUE_LABELS (comma or newline separated). Exits 0 area present,
# 1 none (the message is the comment the workflow posts), 3 usage. issue-area.yml is pinned structurally (IA8-IA9).
# The issue forms under .github/ISSUE_TEMPLATE/ are pinned in IA10.

setup() {
  REPO="$(cd "$BATS_TEST_DIRNAME/.." && pwd)"
  CHK="$REPO/scripts/issue-area.sh"
  WF="$REPO/.github/workflows/issue-area.yml"
  unset ISSUE_LABELS
}

@test "IA1: an area label among others -> exit 0 naming the area" {
  run bash "$CHK" "bug,area:mcp-n4,retro-delta"
  [ "$status" -eq 0 ]
  [ "$output" = "issue-area: ok (area:mcp-n4)" ]
}

@test "IA2: no area label -> exit 1, the message names the labels to add and carries the marker" {
  # Mutation: IA2-any-label-ok -- accepting any label lets an unlabelled-by-area issue through
  run bash "$CHK" "bug,retro-delta"
  [ "$status" -eq 1 ]
  [[ "$output" == *"<!-- issue-area -->"* ]]
  [[ "$output" == *"area:mcp-n4"* ]]
  [[ "$output" == *"area:build-n4-module"* ]]
}

@test "IA3: an empty label list -> exit 1, not a usage error" {
  run bash "$CHK" ""
  [ "$status" -eq 1 ]
  ISSUE_LABELS="" run bash "$CHK"
  [ "$status" -eq 1 ]
}

@test "IA4: the labels come from ISSUE_LABELS when no argument is given; the argument wins" {
  ISSUE_LABELS="area:build-n4-module" run bash "$CHK"
  [ "$status" -eq 0 ]
  [ "$output" = "issue-area: ok (area:build-n4-module)" ]
  ISSUE_LABELS="area:build-n4-module" run bash "$CHK" "bug"
  [ "$status" -eq 1 ]
}

@test "IA5: neither an argument nor ISSUE_LABELS -> exit 3 with a usage line on stderr, nothing on stdout" {
  run --separate-stderr bash "$CHK"
  [ "$status" -eq 3 ]
  [ -z "$output" ]
  # shellcheck disable=SC2154 # $stderr is set by `run --separate-stderr`
  [[ "$stderr" == *"usage: issue-area.sh"* ]]
  run --separate-stderr bash "$CHK" a b
  [ "$status" -eq 3 ]
}

@test "IA6: a bare or malformed area label does not count" {
  # Mutation: IA6-prefix-only -- matching the prefix `area:` alone accepts a label with no area name
  for l in "area:" "area: x" "Area:mcp-n4" "xarea:mcp-n4" "area:mcp n4" "areas:mcp-n4"; do
    run bash "$CHK" "$l"
    [ "$status" -eq 1 ] || { echo "accepted: $l"; return 1; }
  done
}

@test "IA7: spaces around a label and newline separators are tolerated; two areas are both reported" {
  run bash "$CHK" $'bug\n area:mcp-n4 , area:build-n4-module '
  [ "$status" -eq 0 ]
  [ "$output" = "issue-area: ok (area:mcp-n4 area:build-n4-module)" ]
}

@test "IA8: issue-area.yml runs on the four issue events, with issues: write as the only permission" {
  grep -q '^  issues:' "$WF"
  [ "$(grep -c 'types: \[opened, edited, labeled, unlabeled\]' "$WF")" -eq 1 ]
  perms=$(sed -n '/^permissions:/,/^[a-z]/p' "$WF" | grep -E '^  [a-z-]+:')
  [ "$perms" = "  issues: write" ]
  run ! grep -qE '^\s+(contents|pull-requests|actions|id-token):' "$WF"
}

@test "IA9: every action in issue-area.yml is pinned by a full commit SHA and the labels go through env" {
  # Mutation: IA9-tag-pin -- a tag pin can be moved under us
  while read -r ref; do
    [[ "$ref" =~ @[0-9a-f]{40}$ ]] || { echo "unpinned: $ref"; return 1; }
  done < <(grep -E '^\s*-?\s*uses:' "$WF" | sed -E 's/.*uses:[[:space:]]*([^[:space:]]+).*/\1/')
  # shellcheck disable=SC2016 # the pattern is literal workflow text
  grep -q 'ISSUE_LABELS: \${{ join(github.event.issue.labels.\*.name, .,.) }}' "$WF"
  run ! grep -E 'run:.*\$\{\{' "$WF"
}

@test "IA10: the issue forms apply the area label, and blank issues are off" {
  T="$REPO/.github/ISSUE_TEMPLATE"
  grep -qx 'blank_issues_enabled: false' "$T/config.yml"
  grep -q 'labels: \["area:mcp-n4"\]' "$T/mcp-n4.yml"
  grep -q 'labels: \["area:build-n4-module"\]' "$T/build-n4-module.yml"
  for f in mcp-n4 build-n4-module; do
    grep -q '^name: ' "$T/$f.yml"
    grep -q '^description: ' "$T/$f.yml"
    grep -q '^body:' "$T/$f.yml"
  done
}
