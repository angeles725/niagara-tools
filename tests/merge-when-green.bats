#!/usr/bin/env bats
bats_require_minimum_version 1.5.0
# merge-when-green.bats — scripts/merge-when-green.sh (go/internal/mergewait) against a stub gh: it never calls
# GitHub. The stub answers from files under $GHS (one reply per call, the last one repeated) and logs every call.
# (retro 2026-11-15 coordination-stretch C5)

setup_file() {
  REPO="$(cd "$BATS_TEST_DIRNAME/.." && pwd)"
  # shellcheck disable=SC1091  # go/ helper, resolved at runtime
  . "$REPO/go/go-tool.sh"
  go_tool_bin "$REPO/go" merge-when-green >/dev/null
}

setup() {
  REPO="$(cd "$BATS_TEST_DIRNAME/.." && pwd)"
  MWG="$REPO/scripts/merge-when-green.sh"
  HEAD_SHA=1111111111111111111111111111111111111111
  GHS="$BATS_TEST_TMPDIR/gh"; export GHS
  mkdir -p "$GHS/bin"
  # reply <key> <json>... — the replies, in order, to the gh calls whose words start with <key> (spaces as _)
  cat > "$GHS/bin/gh" <<'GH'
#!/usr/bin/env bash
printf '%s\n' "$*" >> "$GHS/calls"
for key in pr_view pr_merge api_compare api_checks run_view run_rerun; do
  case "$key:$*" in
    pr_view:"pr view"*|pr_merge:"pr merge"*|run_view:"run view"*|run_rerun:"run rerun"*) ;;
    api_compare:api*compare*|api_checks:api*check-runs*) ;;
    *) continue ;;
  esac
  n="$(cat "$GHS/$key.n" 2>/dev/null || echo 0)"; n=$((n + 1)); echo "$n" > "$GHS/$key.n"
  f="$GHS/$key.$n"; [ -f "$f" ] || f="$(ls "$GHS/$key".[0-9]* 2>/dev/null | sort -t. -k2 -n | tail -n 1)"
  [ -n "$f" ] || { echo "stub gh: no reply for $*" >&2; exit 9; }
  if [ -f "$f.rc" ]; then cat "$f" >&2; exit "$(cat "$f.rc")"; fi
  cat "$f"; exit 0
done
echo "stub gh: unexpected call $*" >&2; exit 9
GH
  chmod +x "$GHS/bin/gh"
  reply pr_view "{\"state\":\"OPEN\",\"isDraft\":false,\"headRefOid\":\"$HEAD_SHA\",\"baseRefName\":\"main\"}"
  reply api_compare '{"behind_by":0}'
  reply pr_merge ''
}

reply() { local key="$1" i=0; shift; for r in "$@"; do i=$((i + 1)); printf '%s\n' "$r" > "$GHS/$key.$i"; done; }
check() { printf '{"id":%s,"name":"%s","status":"%s","conclusion":"%s","details_url":"https://github.com/o/r/actions/runs/77/job/%s"}' "$1" "$2" "$3" "$4" "$1"; }
mwg() { PATH="$GHS/bin:$PATH" run "$MWG" 5 --interval 0s --timeout 1m; }

@test "MWG1: green checks of the exact head merge that head with --match-head-commit (rebase by default)" {
  # Mutation: MWG1 -- dropping --match-head-commit merges whatever the head is at merge time.
  reply api_checks "{\"check_runs\":[$(check 1 go in_progress '')]}" "{\"check_runs\":[$(check 1 go completed success)]}"
  mwg
  [ "$status" -eq 0 ]
  [[ "$output" == *"merged #5 at $HEAD_SHA (--rebase)"* ]]
  grep -qx "pr merge 5 --rebase --match-head-commit $HEAD_SHA" "$GHS/calls"
  grep -q "commits/$HEAD_SHA/check-runs" "$GHS/calls"
}

@test "MWG2: a draft is refused (exit 4) and nothing is merged" {
  reply pr_view "{\"state\":\"OPEN\",\"isDraft\":true,\"headRefOid\":\"$HEAD_SHA\",\"baseRefName\":\"main\"}"
  mwg
  [ "$status" -eq 4 ]
  [[ "$output" == *"#5 is a draft; a draft is never green"* ]]
  run grep -c "pr merge" "$GHS/calls"
  [ "$output" = 0 ]
}

@test "MWG3: a branch behind main is refused (exit 5) before any wait" {
  reply api_compare '{"behind_by":3}'
  mwg
  [ "$status" -eq 5 ]
  [[ "$output" == *"does not contain the current main (3 commit(s) behind)"* ]]
}

@test "MWG4: a failed check exits 1 with the last lines of its log" {
  reply api_checks "{\"check_runs\":[$(check 9 bats completed failure)]}"
  printf 'a\nb\nnot ok 12 the case\n' > "$GHS/run_view.1"
  PATH="$GHS/bin:$PATH" run "$MWG" 5 --interval 0s --log-lines 2
  [ "$status" -eq 1 ]
  [[ "$output" == *'check "bats" failed (failure)'* ]]
  [[ "$output" == *"last 2 line(s) of its log:"$'\n'"b"$'\n'"not ok 12 the case"* ]]
}

@test "MWG5: a check that failed on a 429 download is re-run once, then the green head merges" {
  reply api_checks "{\"check_runs\":[$(check 9 bats completed failure)]}" "{\"check_runs\":[$(check 10 bats completed success)]}"
  printf 'curl: (22) The requested URL returned error: 429\n' > "$GHS/run_view.1"
  reply run_rerun ''
  mwg
  [ "$status" -eq 0 ]
  grep -qx "run rerun 77 --failed" "$GHS/calls"
}

@test "MWG6: exits 2 on usage, 3 when gh is missing, 0 on --help; zsh runs the launcher like bash" {
  run "$MWG"
  [ "$status" -eq 2 ]
  run "$MWG" 5 --squash --merge
  [ "$status" -eq 2 ]
  run "$MWG" --help
  [ "$status" -eq 0 ]
  [[ "$output" == *"usage: merge-when-green <pr>"* ]]
  local t nogh="$BATS_TEST_TMPDIR/nogh"   # the launcher's tools without gh
  mkdir -p "$nogh"
  for t in dirname find; do ln -s "$(command -v "$t")" "$nogh/$t"; done
  PATH="$nogh" run "$(command -v bash)" "$MWG" 5
  [ "$status" -eq 3 ]
  [[ "$output" == *"gh is not on PATH"* ]]
  # the session shell is zsh: when it is installed, it must run the launcher too (bash always does, above)
  local sh ran=0
  for sh in bash zsh; do
    command -v "$sh" >/dev/null || continue
    reply api_checks "{\"check_runs\":[$(check 1 go completed success)]}"
    PATH="$GHS/bin:$PATH" run "$sh" "$MWG" 5 --interval 0s
    [ "$status" -eq 0 ] || { echo "$sh: exit $status: $output"; return 1; }
    ran=$((ran + 1))
  done
  [ "$ran" -ge 1 ]
}
