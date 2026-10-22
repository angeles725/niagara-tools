#!/usr/bin/env bats
# awk-lint.bats — toolbelt/lib/awk-lint.sh, the shared skeleton of the per-file awk lints
# (argument parsing, temp dir, fail-closed file walk, one awk run per file, exit code).
# Each test drives a tiny demo lint that sources the lib, so the contract is checked once here
# instead of only through every caller. [ev: retro 2026-10-22-awk-lint-skeleton]

setup() {
  TMPDIR_T="$(mktemp -d)"; export TMPDIR_T
  KIT="$(cd "$BATS_TEST_DIRNAME/.." && pwd)/build-n4-module-kit"
  mkdir -p "$TMPDIR_T/src/com"
  printf 'class A {}\n' > "$TMPDIR_T/src/com/A.java"
}
teardown() {
  chmod -R u+rwx "$TMPDIR_T" 2>/dev/null || true
  rm -rf "$TMPDIR_T"
}

# demo_lint <arity> [extra-setup]: writes $TMPDIR_T/demo-lint.sh, a lint that WARNs on every line
# containing BAD. <extra-setup> is shell run between the parse and the scan (e.g. AL_AWK_ARGS).
demo_lint() {
  cat > "$TMPDIR_T/demo-lint.sh" <<EOF
#!/usr/bin/env bash
set -u
. "$KIT/toolbelt/lib/awk-lint.sh"
awk_lint_parse demo-lint $1 'usage: demo-lint.sh [--strict] <src-root>' "\$@"
${2:-}
cat > "\$AL_TMP/main.awk" << 'AWKEOF'
/BAD/ { printf "WARN  demo-lint  %s:%d  bad%s\n", FILE, FNR, EXTRA }
AWKEOF
awk_lint_run -name '*.java'
EOF
}

@test "AL1: no root prints the caller's usage line and exits 3" {
  demo_lint one
  run bash "$TMPDIR_T/demo-lint.sh"
  [ "$status" -eq 3 ]
  [ "$output" = "usage: demo-lint.sh [--strict] <src-root>" ]
}

@test "AL2: an unknown flag names the flag, then the usage line, exit 3" {
  demo_lint one
  run bash "$TMPDIR_T/demo-lint.sh" --bogus "$TMPDIR_T/src"
  [ "$status" -eq 3 ]
  [ "$output" = "$(printf 'demo-lint: unknown flag: --bogus\nusage: demo-lint.sh [--strict] <src-root>')" ]
}

@test "AL3: a root that is not a directory is an env error (exit 3), prefixed with the caller's name" {
  # Mutation: AL3 -- dropping the directory check lets scan_files report it with the wrong prefix.
  demo_lint one
  run bash "$TMPDIR_T/demo-lint.sh" "$TMPDIR_T/src/com/A.java"
  [ "$status" -eq 3 ]
  [ "$output" = "demo-lint: not a directory: $TMPDIR_T/src/com/A.java" ]
}

@test "AL4: a sub-directory find cannot enter is an env error (exit 3), never a clean pass" {
  # Mutation: AL4 -- ignoring the scan_files status skips the locked directory and exits 0.
  mkdir -p "$TMPDIR_T/src/locked"
  chmod 000 "$TMPDIR_T/src/locked"
  if [ -r "$TMPDIR_T/src/locked" ]; then skip "running as root"; fi
  demo_lint one
  run bash "$TMPDIR_T/demo-lint.sh" "$TMPDIR_T/src"
  [ "$status" -eq 3 ]
  [[ "$output" == "demo-lint: cannot list every file under $TMPDIR_T/src: "* ]]
}

@test "AL5: a WARN row without --strict prints the row and exits 0" {
  printf 'BAD\n' > "$TMPDIR_T/src/com/B.java"
  demo_lint one
  run bash "$TMPDIR_T/demo-lint.sh" "$TMPDIR_T/src"
  [ "$status" -eq 0 ]
  [ "$output" = "WARN  demo-lint  $TMPDIR_T/src/com/B.java:1  bad" ]
}

@test "AL6: a WARN row under --strict exits 1; a clean tree under --strict exits 0" {
  # Mutation: AL6 -- ignoring --strict exits 0 on a WARN.
  demo_lint one
  run bash "$TMPDIR_T/demo-lint.sh" --strict "$TMPDIR_T/src"
  [ "$status" -eq 0 ]
  [ -z "$output" ]
  printf 'BAD\n' > "$TMPDIR_T/src/com/B.java"
  run bash "$TMPDIR_T/demo-lint.sh" --strict "$TMPDIR_T/src"
  [ "$status" -eq 1 ]
  [ "$output" = "WARN  demo-lint  $TMPDIR_T/src/com/B.java:1  bad" ]
}

@test "AL7: an awk failure on one file is 'cannot scan' (exit 3); the other files are still scanned" {
  # Mutation: AL7 -- ignoring the awk exit status reports the unreadable file as clean.
  printf 'BAD\n' > "$TMPDIR_T/src/com/B.java"
  printf 'x\n' > "$TMPDIR_T/src/com/C.java"
  chmod 000 "$TMPDIR_T/src/com/C.java"
  if [ -r "$TMPDIR_T/src/com/C.java" ]; then skip "running as root"; fi
  demo_lint one
  run bash "$TMPDIR_T/demo-lint.sh" "$TMPDIR_T/src"
  [ "$status" -eq 3 ]
  [[ "$output" == *"WARN  demo-lint  $TMPDIR_T/src/com/B.java:1  bad"* ]]
  [[ "$output" == *"demo-lint: cannot scan $TMPDIR_T/src/com/C.java: "* ]]
}

@test "AL8: arity 'one' rejects a second root with the usage line (exit 3)" {
  demo_lint one
  run bash "$TMPDIR_T/demo-lint.sh" "$TMPDIR_T/src" "$TMPDIR_T/src"
  [ "$status" -eq 3 ]
  [ "$output" = "usage: demo-lint.sh [--strict] <src-root>" ]
}

@test "AL9: arity 'first' scans the first root and ignores the rest unchecked" {
  printf 'BAD\n' > "$TMPDIR_T/src/com/B.java"
  demo_lint first
  run bash "$TMPDIR_T/demo-lint.sh" -- "$TMPDIR_T/src" "$TMPDIR_T/nonexist"
  [ "$status" -eq 0 ]
  [ "$output" = "WARN  demo-lint  $TMPDIR_T/src/com/B.java:1  bad" ]
}

@test "AL10: arity 'each' checks every root before scanning, then scans them all in order" {
  # Mutation: AL10 -- scanning only the first root misses the WARN in the second.
  mkdir -p "$TMPDIR_T/src2"
  printf 'BAD\n' > "$TMPDIR_T/src2/D.java"
  demo_lint each
  run bash "$TMPDIR_T/demo-lint.sh" "$TMPDIR_T/src" "$TMPDIR_T/src2"
  [ "$status" -eq 0 ]
  [ "$output" = "WARN  demo-lint  $TMPDIR_T/src2/D.java:1  bad" ]
  run bash "$TMPDIR_T/demo-lint.sh" "$TMPDIR_T/src2" "$TMPDIR_T/nonexist"
  [ "$status" -eq 3 ]
  [ "$output" = "demo-lint: not a directory: $TMPDIR_T/nonexist" ]
}

@test "AL11: AL_AWK_ARGS reaches awk ahead of the main program (extra -v / -f)" {
  printf 'BAD\n' > "$TMPDIR_T/src/com/B.java"
  demo_lint one 'AL_AWK_ARGS=(-v EXTRA=" +extra")'
  run bash "$TMPDIR_T/demo-lint.sh" "$TMPDIR_T/src"
  [ "$status" -eq 0 ]
  [ "$output" = "WARN  demo-lint  $TMPDIR_T/src/com/B.java:1  bad +extra" ]
}

@test "AL12: AL_FLAG_HOOK consumes caller-specific flags; an unconsumed flag is still unknown" {
  # Mutation: AL12 -- never calling the hook makes --level an unknown flag.
  cat > "$TMPDIR_T/demo-lint.sh" <<EOF
#!/usr/bin/env bash
set -u
. "$KIT/toolbelt/lib/awk-lint.sh"
LEVEL=none
demo_flag() { if [ "\$1" = --level ] && [ \$# -ge 2 ]; then LEVEL="\$2"; AL_SHIFT=2; fi; }
AL_FLAG_HOOK=demo_flag
awk_lint_parse demo-lint one 'usage: demo-lint.sh [--strict] [--level <x>] <src-root>' "\$@"
printf 'level=%s strict=%s root=%s\n' "\$LEVEL" "\$AL_STRICT" "\${AL_ROOTS[0]}"
EOF
  run bash "$TMPDIR_T/demo-lint.sh" --level 7 --strict "$TMPDIR_T/src"
  [ "$status" -eq 0 ]
  [ "$output" = "level=7 strict=1 root=$TMPDIR_T/src" ]
  run bash "$TMPDIR_T/demo-lint.sh" --other "$TMPDIR_T/src"
  [ "$status" -eq 3 ]
  [[ "$output" == "demo-lint: unknown flag: --other"* ]]
}

# list_lint: writes $TMPDIR_T/list-lint.sh, a flagless line-loop lint (arity raw) that lists *.java
# with awk_lint_list, prints one row per file and ends with awk_lint_exit <code>.
list_lint() {
  cat > "$TMPDIR_T/list-lint.sh" <<EOF
#!/usr/bin/env bash
set -u
. "$KIT/toolbelt/lib/awk-lint.sh"
awk_lint_parse list-lint raw 'usage: list-lint.sh <src-root>' "\$@"
awk_lint_list -name '*.java'
while IFS= read -r f; do printf 'ROW  %s\n' "\$f"; done < "\$AL_TMP/files"
awk_lint_exit ${1:-0}
EOF
}

@test "AL13: arity 'raw' takes the first argument verbatim as the root: no flag parsing, rest ignored" {
  # Mutation: AL13 -- parsing flags in raw mode turns '-h' into an unknown flag instead of a root.
  list_lint
  run bash "$TMPDIR_T/list-lint.sh"
  [ "$status" -eq 3 ]
  [ "$output" = "usage: list-lint.sh <src-root>" ]
  run bash "$TMPDIR_T/list-lint.sh" -h
  [ "$status" -eq 3 ]
  [ "$output" = "list-lint: not a directory: -h" ]
  run bash "$TMPDIR_T/list-lint.sh" "$TMPDIR_T/src" --strict "$TMPDIR_T/nonexist"
  [ "$status" -eq 0 ]
  [ "$output" = "ROW  $TMPDIR_T/src/com/A.java" ]
}

@test "AL14: awk_lint_list writes the sorted file list; awk_lint_exit returns the caller's code" {
  printf 'class B {}\n' > "$TMPDIR_T/src/B.java"
  list_lint 1
  run bash "$TMPDIR_T/list-lint.sh" "$TMPDIR_T/src"
  [ "$status" -eq 1 ]
  [ "$output" = "$(printf 'ROW  %s\nROW  %s' "$TMPDIR_T/src/B.java" "$TMPDIR_T/src/com/A.java")" ]
}

@test "AL15: an incomplete awk_lint_list walk still lists the rest, then awk_lint_exit exits 3" {
  # Mutation: AL15 -- exiting on the walk error drops the rows; ignoring it exits with the caller's code.
  mkdir -p "$TMPDIR_T/src/locked"
  chmod 000 "$TMPDIR_T/src/locked"
  if [ -r "$TMPDIR_T/src/locked" ]; then skip "running as root"; fi
  list_lint 1
  run bash "$TMPDIR_T/list-lint.sh" "$TMPDIR_T/src"
  [ "$status" -eq 3 ]
  [[ "${lines[0]}" == "list-lint: cannot list every file under $TMPDIR_T/src: "* ]]
  [ "${lines[1]}" = "ROW  $TMPDIR_T/src/com/A.java" ]
}

@test "AL16: AL_AWK_ARGS is expanded only through the set -u safe idiom (empty array on bash < 4.4)" {
  # Mutation: AL16 -- a bare "\${AL_AWK_ARGS[@]}" aborts with 'unbound variable' on bash 4.3 when it is empty.
  run grep -nE '(^|[^+])"\$\{AL_AWK_ARGS\[@\]\}"' "$KIT/toolbelt/lib/awk-lint.sh"
  [ "$status" -eq 1 ]
  # shellcheck disable=SC2016  # a literal pattern: the expansion must stay unexpanded
  run grep -c '${AL_AWK_ARGS\[@\]+"${AL_AWK_ARGS\[@\]}"}' "$KIT/toolbelt/lib/awk-lint.sh"
  [ "$output" -ge 1 ]
}
