#!/usr/bin/env bats
# lint-poll-scroll-restore.bats — a poll repaint that clears a container, in a file with a scroller and no scroll
# restore, WARNs (#909). The rule's golden cases live in go/internal/lint/rules/pollscrollrestore.

setup() {
  KIT="$(cd "$BATS_TEST_DIRNAME/.." && pwd)/build-n4-module-kit"
  L="$KIT/toolbelt/lint-poll-scroll-restore.sh"
  ROOT="$BATS_TEST_TMPDIR/ux"
  mkdir -p "$ROOT"
}

@test "PSR-usage: no argument prints the usage and exits 3" {
  run "$L"
  [ "$status" -eq 3 ]
  [ "$output" = "usage: lint-poll-scroll-restore.sh [--strict] <ux-src-root>" ]
}

@test "PSR-fixture: the kit fixtures are clean under --strict" {
  run "$L" --strict "$KIT/fixtures"
  [ "$status" -eq 0 ]
  [ -z "$output" ]
}

@test "PSR1: a poll that clears a list in a scrolling page WARNs; a scroll restore makes it clean" {
  # Mutation: PSR1 -- ignoring a scroll call anywhere in the file WARNs a page that restores its scroll.
  printf '%s\n' '.list { overflow-y: auto; }' 'function poll() {' '  list.innerHTML = "";' '}' \
    'setInterval(poll, 5000);' > "$ROOT/app.js"
  run "$L" "$ROOT"
  [ "$status" -eq 0 ]
  [[ "$output" == "WARN  lint-poll-scroll-restore  $ROOT/app.js:3  poll repaint 'poll()' clears a container "* ]]
  run "$L" --strict "$ROOT"
  [ "$status" -eq 1 ]
  printf '%s\n' 'list.scrollTop = saved;' >> "$ROOT/app.js"
  run "$L" --strict "$ROOT"
  [ "$status" -eq 0 ]
  [ -z "$output" ]
}

@test "PSR2: the same poll with no scrolling container in the file is clean" {
  # Mutation: PSR2 -- WARNing without a scrolling container in the file flags every page that clears a small list.
  printf '%s\n' 'function poll() {' '  list.innerHTML = "";' '}' 'setInterval(poll, 5000);' > "$ROOT/app.js"
  run "$L" --strict "$ROOT"
  [ "$status" -eq 0 ]
  [ -z "$output" ]
}

@test "PSR-K20: a root with no *.html/*.js is the ERROR row, exit 3" {
  run "$L" "$ROOT"
  [ "$status" -eq 3 ]
  [ "$output" = "ERROR  lint-poll-scroll-restore  $ROOT  no files to scan (K20: wrong path or empty scaffold)" ]
}
