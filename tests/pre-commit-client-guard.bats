#!/usr/bin/env bats
bats_require_minimum_version 1.5.0
# The client-identifier guard at commit time (.githooks/pre-commit and .githooks/commit-msg): a client name
# is refused before a local commit exists, not only at pre-push. The hooks hand niagara-go's client-guard one
# pre-push-shaped stdin line naming a snapshot commit of the STAGED tree (pre-commit) or of the proposed
# message (commit-msg), built in a throwaway object directory, so the guard sees exactly what the commit
# would add. Without client-guard on PATH they refuse when a client denylist exists, else warn and go on.
# [ev: retro 2026-11-10-delivery-guards-and-ui-contrast Δ1]
#
# Hermetic: git is real (a throwaway repo, hooks copied in); client-guard is a fake that scans the commits
# the stdin line publishes for the token ZZDENY in added lines and messages, as the real guard does with
# its denylist. The denylist is a path in the test tmpdir.

setup() {
  REPO="$(cd "$BATS_TEST_DIRNAME/.." && pwd)"
  HOOKS="$BATS_TEST_TMPDIR/hooks"; mkdir -p "$HOOKS"
  for h in pre-commit commit-msg; do
    [ ! -f "$REPO/.githooks/$h" ] || cp "$REPO/.githooks/$h" "$HOOKS/$h"
  done
  FAKEBIN="$BATS_TEST_TMPDIR/bin"; mkdir -p "$FAKEBIN"
  GUARD_LOG="$BATS_TEST_TMPDIR/guard.log"; export GUARD_LOG
  cat > "$BATS_TEST_TMPDIR/client-guard" <<'GUARD'
#!/usr/bin/env bash
# fake client-guard: scan what each stdin line publishes for ZZDENY (added lines and messages)
printf 'args:%s\n' "$*" >> "$GUARD_LOG"
rc=0
while read -r _ lsha _ rsha; do
  [ -n "$lsha" ] || continue
  printf 'line:%s %s\n' "$lsha" "$rsha" >> "$GUARD_LOG"
  revs=("$lsha"); case "$rsha" in *[!0]*) revs+=(--not "$rsha") ;; esac
  # added lines of the patches (never the +++ header), and every line of every message
  out=$(git log -p -U0 --format= "${revs[@]}") || exit 2
  msgs=$(git log --format=%B "${revs[@]}") || exit 2
  printf '%s\n' "$out" >> "$GUARD_LOG.scan"
  printf '%s\n' "$msgs" >> "$GUARD_LOG.msgs"
  if grep -E '^\+[^+].*ZZDENY|^\+ZZDENY' <<<"$out" >/dev/null || grep -q ZZDENY <<<"$msgs"; then
    echo "client-guard: f:1: rule #1 (commit ${lsha:0:10})" >&2; rc=1
  fi
done
exit "$rc"
GUARD
  chmod +x "$BATS_TEST_TMPDIR/client-guard"
  DENYLIST="$BATS_TEST_TMPDIR/denylist.txt"
  GR="$BATS_TEST_TMPDIR/repo"
  git init -q "$GR"
  git -C "$GR" config user.email t@e.st
  git -C "$GR" config user.name test
  git -C "$GR" config core.hooksPath "$HOOKS"
}

with_guard() { cp "$BATS_TEST_TMPDIR/client-guard" "$FAKEBIN/"; }
seed() { printf 'base\n' > "$GR/f"; git -C "$GR" add f; commit -m base; }
# commit: git commit in the test repo with only FAKEBIN (plus the shell basics and git) on PATH
commit() {
  run env PATH="$FAKEBIN:$(dirname "$(command -v git)"):/usr/bin:/bin" NIAGARA_CLIENT_DENYLIST="$DENYLIST" \
    git -C "$GR" commit -q "$@"
}
head_count() { git -C "$GR" rev-list --count HEAD 2>/dev/null || echo 0; }

@test "PC1: a client name in a STAGED line refuses the commit; no commit is made" {
  with_guard; seed
  printf 'hello ZZDENY\n' >> "$GR/f"; git -C "$GR" add f
  commit -m clean
  [ "$status" -ne 0 ]
  [[ "$output" == *"rule #1"* ]]
  [ "$(head_count)" -eq 1 ]
}

@test "PC2: a client name only in an UNSTAGED change does not refuse (the staged diff only)" {
  with_guard; seed
  printf 'ok\n' > "$GR/g"; git -C "$GR" add g
  printf 'hello ZZDENY\n' >> "$GR/f"   # in the work tree, not staged
  commit -m clean
  [ "$status" -eq 0 ]
  [ "$(head_count)" -eq 2 ]
}

@test "PC3: the guard gets a snapshot of the staged tree whose parent is HEAD; the repository never holds it" {
  with_guard; seed
  base=$(git -C "$GR" rev-parse HEAD); : > "$GUARD_LOG"   # only this commit's calls
  printf 'staged-line\n' > "$GR/g"; git -C "$GR" add g
  commit -m clean
  [ "$status" -eq 0 ]
  run grep -c '^line:' "$GUARD_LOG"
  [ "$output" -eq 2 ]   # pre-commit (the staged tree) and commit-msg (the message)
  snap=$(grep -m1 '^line:' "$GUARD_LOG" | cut -d' ' -f1 | cut -d: -f2)
  [ "$(grep -m1 '^line:' "$GUARD_LOG" | cut -d' ' -f2)" = "$base" ]
  grep -q '^+staged-line$' "$GUARD_LOG.scan"
  # the snapshot was written to a throwaway object directory: the repository never holds it
  run git -C "$GR" cat-file -e "$snap"
  [ "$status" -ne 0 ]
}

@test "PC4: the first commit of a repository (no HEAD) is guarded too" {
  with_guard
  printf 'ZZDENY\n' > "$GR/f"; git -C "$GR" add f
  commit -m first
  [ "$status" -ne 0 ]
  [ "$(head_count)" -eq 0 ]
}

@test "PC5: commit-msg refuses a client name in the commit message" {
  with_guard; seed
  printf 'x\n' > "$GR/g"; git -C "$GR" add g
  commit -m "fix for ZZDENY"
  [ "$status" -ne 0 ]
  [ "$(head_count)" -eq 1 ]
}

@test "PC6: commit-msg ignores comment lines git strips from the message" {
  with_guard; seed
  printf 'x\n' > "$GR/g"; git -C "$GR" add g
  printf 'clean subject\n# ZZDENY in a comment git strips\n' > "$BATS_TEST_TMPDIR/msg"
  commit -F "$BATS_TEST_TMPDIR/msg" --cleanup=strip
  [ "$status" -eq 0 ]
}

@test "PC7: a guard that breaks (exit 2) refuses the commit" {
  seed
  printf '#!/usr/bin/env bash\nexit 2\n' > "$FAKEBIN/client-guard"; chmod +x "$FAKEBIN/client-guard"
  printf 'x\n' > "$GR/g"; git -C "$GR" add g
  commit -m clean
  [ "$status" -ne 0 ]
  [ "$(head_count)" -eq 1 ]
}

@test "PC8: no client-guard on PATH and a denylist present refuses with an install hint" {
  seed
  : > "$DENYLIST"
  printf 'x\n' > "$GR/g"; git -C "$GR" add g
  commit -m clean
  [ "$status" -ne 0 ]
  [[ "$output" == *"client-guard is not on PATH"* ]]
  [[ "$output" == *"go install ./cmd/client-guard"* ]]
  [ "$(head_count)" -eq 1 ]
}

@test "PC9: no client-guard and a denylist that cannot be examined counts as present (fail closed)" {
  [ "$(id -u)" -ne 0 ] || skip "running as root: chmod 000 does not block the check"
  seed
  mkdir -p "$BATS_TEST_TMPDIR/locked"; DENYLIST="$BATS_TEST_TMPDIR/locked/denylist.txt"; : > "$DENYLIST"
  chmod 000 "$BATS_TEST_TMPDIR/locked"
  printf 'x\n' > "$GR/g"; git -C "$GR" add g
  commit -m clean
  chmod 755 "$BATS_TEST_TMPDIR/locked"
  [ "$status" -ne 0 ]
  [[ "$output" == *"client-guard is not on PATH"* ]]
}

@test "PC10: no client-guard and no denylist warns and lets the commit go on" {
  seed
  printf 'x\n' > "$GR/g"; git -C "$GR" add g
  commit -m clean
  [ "$status" -eq 0 ]
  [[ "$output" == *"client-identifier check skipped"* ]]
  [ "$(head_count)" -eq 2 ]
}

@test "PC11: commit -v: the staged diff git appends below the scissors line is not part of the message" {
  with_guard
  printf 'hello ZZDENY\n' > "$GR/f"; git -C "$GR" add f
  run env PATH="$FAKEBIN:$(dirname "$(command -v git)"):/usr/bin:/bin" NIAGARA_CLIENT_DENYLIST="$DENYLIST" \
    git -C "$GR" commit -q --no-verify -m base
  [ "$status" -eq 0 ]
  printf 'clean\n' > "$GR/f"; git -C "$GR" add f   # scrub the name: its removed line is in the -v diff
  GIT_EDITOR=true commit -v -e -m "scrub a name"
  [ "$status" -eq 0 ]
  [ "$(head_count)" -eq 2 ]
}
