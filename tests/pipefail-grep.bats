#!/usr/bin/env bats
# pipefail-grep.bats — PG1: no kit script that enables pipefail feeds `printf`/`echo` into `grep -q`.
# grep -q exits on the first match; printf may then die of SIGPIPE (141), and pipefail turns the
# pipeline false: a found pattern reads as a miss (3-35 per 20,000 runs under load, every run once
# the input exceeds the 64 KiB pipe buffer). Use `grep -q P <<< "$X"` instead.
# [ev: retro 2026-10-22-pipefail-grep-q-race]

setup() {
  ROOT="$(cd "$BATS_TEST_DIRNAME/.." && pwd)"
  TMPDIR_T="$(mktemp -d)"; export TMPDIR_T
}
teardown() { rm -rf "$TMPDIR_T"; }

# pg_scan FILE... — print file:line for every printf/echo | grep -q… pipeline in a file that
# enables pipefail (comment lines skipped). Exit 1 when any is found.
pg_scan() {
  local f hits=0
  for f in "$@"; do
    grep -qE '^[[:space:]]*set[[:space:]]+(-[A-Za-z]*[[:space:]]+)*-[A-Za-z]*o[[:space:]]+pipefail' "$f" || continue
    if grep -nE '(^|[^A-Za-z_])(printf|echo)[[:space:]][^|]*\|[[:space:]]*grep[[:space:]]+-[A-Za-z]*q' "$f" \
        | grep -vE '^[0-9]+:[[:space:]]*#' | sed "s|^|$f:|" | grep .; then hits=1; fi
  done
  return "$hits"
}

@test "PG1: no pipefail kit script pipes printf/echo into grep -q" {
  run pg_scan "$ROOT"/scripts/*.sh "$ROOT"/scripts/lib/*.sh "$ROOT"/build-n4-module-kit/toolbelt/*.sh \
    "$ROOT"/build-n4-module-kit/toolbelt/lib/*.sh
  [ "$status" -eq 0 ]
  [ -z "$output" ]
}

@test "PG1-neg: the scan flags every grep -q flag form in a pipefail file (file:line named)" {
  # Mutation: PG1-neg -- dropping `echo` from the (printf|echo) source match hides the bad.sh:4 hit.
  cat > "$TMPDIR_T/bad.sh" <<'SH'
set -euo pipefail
printf '%s\n' "$L" | grep -q '^a$' || true
printf '%s\n' "$L" | grep -qx "$f" || true
if echo "$p" | grep -qF 'x'; then :; fi
if ! printf '%s\n' "$d" | grep -qxF "$dep"; then :; fi
printf '%s\n' "$d" | grep -qE '^b' && :
SH
  run pg_scan "$TMPDIR_T/bad.sh"
  [ "$status" -eq 1 ]
  local n; for n in 2 3 4 5 6; do [[ "$output" == *"bad.sh:$n:"* ]]; done
}

@test "PG1-pos: here-strings, comments and non-pipefail files stay clean" {
  # Mutation: PG1-pos -- dropping the pipefail-file filter flags nopf.sh (and the toolbelt's non-pipefail lints in PG1).
  cat > "$TMPDIR_T/good.sh" <<'SH'
set -euo pipefail
grep -q '^a$' <<< "$L" || true
# printf '%s\n' "$L" | grep -q x   (comment)
printf '%s\n' "$L" | grep -c x || true
SH
  cat > "$TMPDIR_T/nopf.sh" <<'SH'
#!/bin/bash
printf '%s\n' "$L" | grep -q x
SH
  run pg_scan "$TMPDIR_T/good.sh" "$TMPDIR_T/nopf.sh"
  [ "$status" -eq 0 ]
  [ -z "$output" ]
}
