#!/usr/bin/env bats
# lint-empty-root.bats — a bash lint never reports green on input it did not read. An EMPTY root (or one
# with no file the lint scans: a typo'd sub-path, a wrong module) used to exit 0 with no output on most
# lints. The shared layer now owns the rule once: awk_lint_run/awk_lint_list (lib/awk-lint.sh) and
# scan_files_nonempty_or_exit (lib/scan-files.sh) print `ERROR  <name>  <root>  ...` and exit 3.
# ER1 sweeps every bash lint wired through that layer; the Go-backed wrappers (go/), lint-structure.sh
# (being ported to Go) and the documented exceptions below are out of scope.

setup() {
  KIT="$(cd "$BATS_TEST_DIRNAME/.." && pwd)/build-n4-module-kit"
  EMPTY="$BATS_TEST_TMPDIR/empty"; mkdir -p "$EMPTY"
}

# Documented exceptions (ER5 pins them): lint-jasmine-ux.sh and lint-uberjar-api-conflict.sh are advisory
# with "Exit: 0 always" (report-module runs the latter on every module root), lint-lexicon-ascii.sh's
# LA5 contract is "no *.lexicon files exits 0", and its companion lint-lexicon-apostrophe.sh keeps the same
# contract (report-module runs it warnOnly on every artifact, go/internal/reportmodule/members.go). lint-structure.sh is out of scope (Go port). lint-manual-labels.sh
# takes a manual before its ui-dirs (one root argument is its usage error); tests/lint-manual-labels.bats pins it.
EXEMPT=" lint-jasmine-ux.sh lint-lexicon-apostrophe.sh lint-lexicon-ascii.sh lint-manual-labels.sh lint-structure.sh lint-uberjar-api-conflict.sh "

# shared_layer_lints: every toolbelt lint-*.sh that sources lib/awk-lint.sh or lib/scan-files.sh, or is an n4lint
# wrapper (the Go lint engine prints the same ERROR row, go/internal/lint Spec.EmptyError), minus EXEMPT; one basename
# per line. Counting the Go wrappers keeps ER1's floor steady as bash lints are ported.
shared_layer_lints() {
  local l b
  for l in "$KIT"/toolbelt/lint-*.sh; do
    b="${l##*/}"
    case "$EXEMPT" in *" $b "*) continue ;; esac
    # shellcheck disable=SC2016  # the wrapper's literal source text, not an expansion
    grep -qE 'lib/(awk-lint|scan-files)\.sh|go_tool_bin "\$GO_DIR" n4lint' "$l" || continue
    printf '%s\n' "$b"
  done
}

@test "ER1: every shared-layer bash lint exits 3 with an ERROR row naming an EMPTY root" {
  # Mutation: ER1 -- dropping the empty-list check from _al_list or scan_files_nonempty_or_exit names the lints here.
  local b bad="" n=0
  for b in $(shared_layer_lints); do
    n=$((n + 1))
    run timeout 60 bash "$KIT/toolbelt/$b" "$EMPTY"
    if [ "$status" -ne 3 ] || ! printf '%s\n' "$output" | grep -qE "^(ERROR  [^ ]+  $EMPTY  |[^ ]+  ERROR  $EMPTY  )"; then
      bad+=" $b(rc=$status)"
    fi
  done
  [ "$n" -ge 24 ]  # one fewer per lint ported to Go (lint-session-store-lazy-evict, 2026-10-23)
  [ -z "$bad" ] || { printf 'silent on an empty root:%s\n' "$bad"; return 1; }
}

@test "ER2: a root holding only files no lint scans (a wrong sub-path) is the same ERROR, exit 3" {
  printf 'notes\n' > "$EMPTY/README.txt"
  local b bad=""
  for b in $(shared_layer_lints); do
    run timeout 60 bash "$KIT/toolbelt/$b" "$EMPTY"
    [ "$status" -eq 3 ] || bad+=" $b(rc=$status)"
  done
  [ -z "$bad" ] || { printf 'silent on a root with nothing to scan:%s\n' "$bad"; return 1; }
}

@test "ER3: awk_lint_run with arity 'each' names every empty root, scans the others, exits 3" {
  mkdir -p "$BATS_TEST_TMPDIR/full"; printf 'BAD\n' > "$BATS_TEST_TMPDIR/full/A.java"
  cat > "$BATS_TEST_TMPDIR/each.sh" <<SH
#!/usr/bin/env bash
set -u
. "$KIT/toolbelt/lib/awk-lint.sh"
awk_lint_parse demo each 'usage: demo <root>...' "\$@"
printf '/BAD/ { printf "WARN  demo  %%s:%%d  bad\\\\n", FILE, FNR }\n' > "\$AL_TMP/main.awk"
awk_lint_run -name '*.java'
SH
  run bash "$BATS_TEST_TMPDIR/each.sh" "$EMPTY" "$BATS_TEST_TMPDIR/full"
  [ "$status" -eq 3 ]
  [ "${lines[0]}" = "ERROR  demo  $EMPTY  no files to scan (K20: wrong path or empty scaffold)" ]
  [ "${lines[1]}" = "WARN  demo  $BATS_TEST_TMPDIR/full/A.java:1  bad" ]
}

@test "ER4: scan_files_nonempty_or_exit exits 3 with the ERROR row on an empty list, returns 0 otherwise" {
  # shellcheck disable=SC1091
  run bash -c '. "$1/toolbelt/lib/scan-files.sh"; scan_files_nonempty_or_exit demo "$2/out" "$3" -name "*.java"; echo reached' \
    _ "$KIT" "$BATS_TEST_TMPDIR" "$EMPTY"
  [ "$status" -eq 3 ]
  [ "$output" = "ERROR  demo  $EMPTY  no files to scan (K20: wrong path or empty scaffold)" ]
  printf 'x\n' > "$EMPTY/A.java"
  run bash -c '. "$1/toolbelt/lib/scan-files.sh"; scan_files_nonempty_or_exit demo "$2/out" "$3" -name "*.java"; cat "$2/out"' \
    _ "$KIT" "$BATS_TEST_TMPDIR" "$EMPTY"
  [ "$status" -eq 0 ]
  [ "$output" = "$EMPTY/A.java" ]
}

@test "ER5: the documented exit-0 exceptions stay exit 0 with no output on an empty root" {
  # Mutation: ER5 -- dropping EmptyOK from rules/uberjarapi (lint-uberjar-api-conflict) turns every report-module run ERROR.
  local b
  for b in lint-jasmine-ux.sh lint-lexicon-apostrophe.sh lint-lexicon-ascii.sh lint-uberjar-api-conflict.sh; do
    run bash "$KIT/toolbelt/$b" "$EMPTY"
    [ "$status" -eq 0 ] && [ -z "$output" ] || { echo "$b: $status $output"; return 1; }
  done
}
