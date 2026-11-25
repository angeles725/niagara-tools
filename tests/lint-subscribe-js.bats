#!/usr/bin/env bats
bats_require_minimum_version 1.5.0
# lint-subscribe-js.sh — a `new baja.Subscriber()` that its .js file never unsubscribes WARNs (FAIL under --strict), and
# so does a `.subscribe(` on a Subscriber-named receiver in a file with no teardown. Every test builds its own source
# root from tests/fixtures/lint-subscribe-js (JavaScript files that are read, never run) and asserts the exact rows and
# exit code. Evidence: the header of the wrapper (Subscriber.js:30,336,632,758, the getting-started tutorial's
# `unsubscribeAll(); detach();`, Widget.js doDestroy). [ev: corpus API survey 2026-11-24 transports row K6, finding G76]

setup() {
  ROOT="$(cd "$BATS_TEST_DIRNAME/.." && pwd)"
  KIT="$ROOT/build-n4-module-kit"
  L="$KIT/toolbelt/lint-subscribe-js.sh"
  FIX="$ROOT/tests/fixtures/lint-subscribe-js"
  SRC="$BATS_TEST_TMPDIR/src"; mkdir -p "$SRC"
  W="WARN  lint-subscribe-js  $SRC"
  TAIL=': the subscriptions outlive the view; detach() only removes handlers, so call unsubscribeAll() in a finally or a widget doDestroy (types/dashboard.md, BajaScript section)'
  NT="new-no-teardown: new baja.Subscriber() is never unsubscribed in this file (no .unsubscribeAll( or .unsubscribe( call)$TAIL"
  ST="subscribe-no-teardown: .subscribe( on a Subscriber that this file never unsubscribes (no .unsubscribeAll( or .unsubscribe( call)$TAIL"
}

# stage <Fixture>... : copy fixtures into the source root under test
stage() { local f; for f in "$@"; do cp "$FIX/$f" "$SRC/"; done; }

# shellcheck disable=SC2154  # $stderr is set by bats run --separate-stderr
@test "SJS-usage: no argument exits 3 with the usage on stderr, nothing on stdout" {
  run --separate-stderr "$L"
  [ "$status" -eq 3 ]
  [ -z "$output" ]
  [[ "$stderr" == "usage: lint-subscribe-js.sh [--strict] <src-root>" ]]
}

@test "SJS-badroot: an unknown option or a missing directory exits 3" {
  stage ok.js
  run "$L" --nope "$SRC"
  [ "$status" -eq 3 ]
  run "$L" "$BATS_TEST_TMPDIR/absent"
  [ "$status" -eq 3 ]
}

@test "SJS-empty: a root with no *.js is an ERROR row and exit 3, never a clean pass" {
  printf 'notes\n' > "$SRC/README.txt"
  run "$L" "$SRC"
  [ "$status" -eq 3 ]
  [ "$output" = "ERROR  lint-subscribe-js  $SRC  no *.js sources found (wrong path or empty source tree)" ]
}

@test "SJS-leak: a new baja.Subscriber() the file never unsubscribes is one WARN row, exit 0" {
  # Mutation: SJS-leak -- ignoring the new baja.Subscriber( site reports nothing for an un-torn-down Subscriber
  stage leak.js
  run "$L" "$SRC"
  [ "$status" -eq 0 ]
  [ "$output" = "$W/leak.js:3  $NT" ]
}

@test "SJS-strict: --strict prints the WARN row as FAIL and exits 1; a paired file stays exit 0" {
  stage leak.js
  run "$L" --strict "$SRC"
  [ "$status" -eq 1 ]
  [ "$output" = "FAIL  lint-subscribe-js  $SRC/leak.js:3  $NT" ]
  mkdir -p "$BATS_TEST_TMPDIR/clean"; cp "$FIX/ok.js" "$BATS_TEST_TMPDIR/clean/"
  run "$L" --strict "$BATS_TEST_TMPDIR/clean"
  [ "$status" -eq 0 ]
  [ -z "$output" ]
}

@test "SJS-paired: the tutorial finally, a widget doDestroy hook and a single unsubscribe each pair the Subscriber" {
  # Mutation: SJS-paired -- reading only teardown calls above the site WARNs the doDestroy hook and the one-call file
  stage paired-finally.js paired-hook.js paired-one.js
  run "$L" "$SRC"
  [ "$status" -eq 0 ]
  [ -z "$output" ]
}

@test "SJS-mixin: a file that loads bajaux/mixin/subscriberMixIn is paired (its destroy unsubscribes); a comment naming it is not" {
  # Mutation: SJS-mixin -- ignoring the mixin id WARNs the view whose widget destroy already unsubscribes
  stage paired-mixin.js mixin-comment.js
  run "$L" "$SRC"
  [ "$status" -eq 0 ]
  [ "$output" = "$W/mixin-comment.js:2  $NT" ]
}

@test "SJS-mixin-code: only a module id in a define/require list or an import pairs; a block comment, a string, another call or a longer id does not" {
  # Mutation: SJS-mixin-code -- matching the id as raw text pairs the block-comment, string, log and longer-id files
  stage mixin-block-comment.js mixin-block-multi.js mixin-import.js mixin-longer-id.js mixin-named-define.js
  stage mixin-other-call.js mixin-require.js mixin-string.js
  run "$L" "$SRC"
  [ "$status" -eq 0 ]
  [ "$output" = "$W/mixin-block-comment.js:2  $NT
$W/mixin-block-multi.js:4  $NT
$W/mixin-longer-id.js:2  $NT
$W/mixin-other-call.js:2  $NT
$W/mixin-string.js:2  $NT" ]
}

@test "SJS-waiver-subscribe: a waiver on the first .subscribe( line or the comment-only line above silences its row; a later line's does not" {
  # Mutation: SJS-waiver-subscribe -- skipping the waiver check on the subscribe-no-teardown row WARNs the waived files
  stage subscribe-waived-inline.js subscribe-waived-above.js subscribe-waived-later.js
  run "$L" "$SRC"
  [ "$status" -eq 0 ]
  [ "$output" = "$W/subscribe-waived-later.js:1  $ST" ]
}

@test "SJS-bare-paired: a bare marker is a FAIL even in a file with a teardown or the mixin (exit 1, no WARN row)" {
  # Mutation: SJS-bare-paired -- returning at the teardown before the bare-marker scan hides the FAIL in a paired file
  stage bare-paired.js bare-mixin.js
  run "$L" "$SRC"
  [ "$status" -eq 1 ]
  [ "$output" = "FAIL  lint-subscribe-js  $SRC/bare-mixin.js:1  waiver-no-reason: lint-subscribe-js: needs a reason after the colon
FAIL  lint-subscribe-js  $SRC/bare-paired.js:1  waiver-no-reason: lint-subscribe-js: needs a reason after the colon" ]
}

@test "SJS-detach: detach() alone is not a teardown (it removes handlers, the components stay subscribed)" {
  # Mutation: SJS-detach -- counting detach( as a teardown passes the Subscriber that is only detached
  stage detach.js
  run "$L" "$SRC"
  [ "$status" -eq 0 ]
  [ "$output" = "$W/detach.js:2  $NT" ]
}

@test "SJS-notcode: a comment, a string or a local function named unsubscribeAll pairs nothing; a commented or quoted site is none" {
  # Mutation: SJS-notcode -- matching raw text pairs a leak with a commented or quoted unsubscribe and flags a commented site
  stage comment-pair.js not-a-site.js local-fn.js
  run "$L" "$SRC"
  [ "$status" -eq 0 ]
  [ "$output" = "$W/comment-pair.js:1  $NT
$W/local-fn.js:2  $NT" ]
}

@test "SJS-subscribe: .subscribe( on a Subscriber-named receiver WARNs once per file when no site and no teardown exist" {
  # Mutation: SJS-subscribe -- matching any receiver WARNs subject.subscribe( and store.subscribe(, or reports every subscribe
  stage subscribe-leak.js subscribe-paired.js rx-subscribe.js new-and-subscribe.js
  run "$L" "$SRC"
  [ "$status" -eq 0 ]
  [ "$output" = "$W/new-and-subscribe.js:1  $NT
$W/subscribe-leak.js:2  $ST" ]
}

@test "SJS-two: every new baja.Subscriber( site of a file with no teardown is its own row" {
  stage two-sites.js
  run "$L" "$SRC"
  [ "$status" -eq 0 ]
  [ "$output" = "$W/two-sites.js:1  $NT
$W/two-sites.js:3  $NT" ]
}

@test "SJS-waiver: a marker with a reason on the site's line or the comment-only line above waives it, no other line" {
  # Mutation: SJS-waiver -- ignoring the marker WARNs the waived sites of waived.js
  stage waived.js
  run "$L" "$SRC"
  [ "$status" -eq 0 ]
  [ "$output" = "$W/waived.js:4  $NT
$W/waived.js:6  $NT" ]
}

@test "SJS-waiver-bare: a marker with no reason is a FAIL (exit 1) and the site is still a WARN row" {
  # Mutation: SJS-waiver-bare -- accepting a bare marker waives the site without a stated reason
  stage waiver-bare.js
  run "$L" "$SRC"
  [ "$status" -eq 1 ]
  [ "$output" = "FAIL  lint-subscribe-js  $SRC/waiver-bare.js:1  waiver-no-reason: lint-subscribe-js: needs a reason after the colon
$W/waiver-bare.js:2  $NT" ]
}

@test "SJS-waiver-literals: a URL or an apostrophe before the marker still waives; a marker inside a string does not" {
  # Mutation: SJS-waiver-literals -- reading a .js line with the Java comment finder loses the marker after an apostrophe in a template
  stage waiver-literals.js waiver-in-string.js
  run "$L" "$SRC"
  [ "$status" -eq 0 ]
  [ "$output" = "$W/waiver-in-string.js:1  $NT" ]
}

@test "SJS-skip: *.min.js, build/, node_modules/ and Java files are not read" {
  # Mutation: SJS-skip -- reading *.min.js, build/, node_modules/ or Java WARNs files the module does not own
  stage ok.js lib.min.js A.java
  mkdir -p "$SRC/build" "$SRC/node_modules/d"
  cp "$FIX/build-gen.js" "$SRC/build/gen.js"; cp "$FIX/node-dep.js" "$SRC/node_modules/d/dep.js"
  run "$L" "$SRC"
  [ "$status" -eq 0 ]
  [ -z "$output" ]
}
