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
# GitHub with no lag: once a merge call succeeded, the PR reads as merged (unless the case scripts the views)
case "$*" in "pr view"*) [ -f "$GHS/merged" ] && [ ! -f "$GHS/no-lagless" ] && { cat "$GHS/merged.json"; exit 0; } ;; esac
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
  cat "$f"
  case "$*" in "pr merge"*) : > "$GHS/merged" ;; esac
  exit 0
done
echo "stub gh: unexpected call $*" >&2; exit 9
GH
  chmod +x "$GHS/bin/gh"
  reply pr_view "{\"state\":\"OPEN\",\"isDraft\":false,\"headRefOid\":\"$HEAD_SHA\",\"baseRefName\":\"main\"}"
  reply api_compare '{"behind_by":0}'
  reply pr_merge ''
  MERGED_JSON="{\"state\":\"MERGED\",\"isDraft\":false,\"headRefOid\":\"$HEAD_SHA\",\"baseRefName\":\"main\",\"mergeCommit\":{\"oid\":\"9999999999999999999999999999999999999999\"}}"
  printf '%s\n' "$MERGED_JSON" > "$GHS/merged.json"
}

reply() { local key="$1" i=0; shift; for r in "$@"; do i=$((i + 1)); printf '%s\n' "$r" > "$GHS/$key.$i"; done; }
check() { printf '{"id":%s,"name":"%s","status":"%s","conclusion":"%s","details_url":"https://github.com/o/r/actions/runs/77/job/%s"}' "$1" "$2" "$3" "$4" "$1"; }
mwg() { PATH="$GHS/bin:$PATH" run "$MWG" 5 --interval 0s --timeout 1m; }

@test "MWG1: green checks of the exact head merge that head with --match-head-commit (rebase by default)" {
  # Mutation: MWG1 -- dropping --match-head-commit merges whatever the head is at merge time.
  reply api_checks "{\"check_runs\":[$(check 1 go in_progress '')]}" "{\"check_runs\":[$(check 1 go completed success)]}"
  mwg
  [ "$status" -eq 0 ]
  [[ "$output" == *"merged #5 at $HEAD_SHA (--rebase), merge commit 9999999999999999999999999999999999999999"* ]]
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

# fail_once <key> <stderr text> — the first reply to <key> is that gh error (exit 1), the later ones as scripted.
fail_once() {
  local key="$1" n=0 i
  while [ -f "$GHS/$key.$((n + 1))" ]; do n=$((n + 1)); done
  for ((i = n; i >= 1; i--)); do mv "$GHS/$key.$i" "$GHS/$key.$((i + 1))"; done
  printf '%s\n' "$2" > "$GHS/$key.1"
  printf '1\n' > "$GHS/$key.1.rc"
}

@test "MWG7: a network error is printed and retried, never hidden" {
  # Mutation: MWG7 -- dropping withRetries ends the wait with exit 3 on the first unreachable network.
  reply api_checks "{\"check_runs\":[$(check 1 go completed success)]}"
  fail_once api_checks 'Post "https://api.github.com/graphql": dial tcp 140.82.112.6:443: connect: network is unreachable'
  mwg
  [ "$status" -eq 0 ]
  [[ "$output" == *"network is unreachable"* ]]
  [[ "$output" == *"a network error; retry 1/3"* ]]
  [ "$(grep -c "check-runs" "$GHS/calls")" -eq 2 ]
}

@test "MWG8: a merge call that fails while GitHub merged anyway is a success, read from the PR state" {
  # Mutation: MWG8 -- deciding on the merge call's exit reports a refusal for a PR that is merged.
  : > "$GHS/no-lagless"
  reply api_checks "{\"check_runs\":[$(check 1 go completed success)]}"
  reply pr_view "{\"state\":\"OPEN\",\"isDraft\":false,\"headRefOid\":\"$HEAD_SHA\",\"baseRefName\":\"main\"}" "$MERGED_JSON"
  fail_once pr_merge 'read tcp 10.0.0.2:51000->140.82.112.6:443: read: connection reset by peer'
  mwg
  [ "$status" -eq 0 ]
  [[ "$output" == *"connection reset by peer"* ]]
  [[ "$output" == *"merge commit 9999999999999999999999999999999999999999"* ]]
}

@test "MWG9: a PR already merged at start is a success naming its merge commit, and nothing is merged again" {
  reply pr_view "$MERGED_JSON"
  mwg
  [ "$status" -eq 0 ]
  [[ "$output" == *"#5 is already merged at $HEAD_SHA, merge commit 9999999999999999999999999999999999999999"* ]]
  run grep -c "pr merge" "$GHS/calls"
  [ "$output" = 0 ]
}

@test "MWG10: a merge call that returned while the PR stays OPEN (a merge queue, auto-merge) exits 9, not 8" {
  # Mutation: MWG10 -- reporting a refusal (8) for a merge that GitHub only queued. (retro 2026-11-15 coordination-stretch T16)
  : > "$GHS/no-lagless"
  reply api_checks "{\"check_runs\":[$(check 1 go completed success)]}"
  mwg
  [ "$status" -eq 9 ]
  [[ "$output" == *"the PR is still OPEN after 4 read(s) (4 answered): a merge queue or auto-merge holds it"* ]]
  [ "$(grep -c "^pr view" "$GHS/calls")" -eq 5 ]
}

@test "MWG11: a merge GitHub refused outright exits 8 after one state read, not 1+--retries" {
  # Mutation: MWG11 -- reading 1+--retries times after a refusal delays exit 8 by --retries intervals. (retro 2026-11-15 coordination-stretch T23)
  : > "$GHS/no-lagless"
  reply api_checks "{\"check_runs\":[$(check 1 go completed success)]}"
  fail_once pr_merge 'GraphQL: Pull Request is not mergeable (mergePullRequest)'
  mwg
  [ "$status" -eq 8 ]
  [[ "$output" == *"is not mergeable"* ]]
  [ "$(grep -c "^pr view" "$GHS/calls")" -eq 2 ] || { cat "$GHS/calls"; return 1; } # the start and one read
}
