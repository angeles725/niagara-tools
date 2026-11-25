#!/usr/bin/env bats
# lint-async-blocking.bats — an ASYNC action that blocks the shared engine thread WARNs (ledger G28). The rule's
# golden cases live in go/internal/lint/rules/asyncblock. [ev: corpus B1238 §1238.3-§1238.5]

bats_require_minimum_version 1.5.0

setup() {
  KIT="$(cd "$BATS_TEST_DIRNAME/.." && pwd)/build-n4-module-kit"
  FX="$BATS_TEST_DIRNAME/fixtures/lint-async-blocking"
  LINT="$KIT/toolbelt/lint-async-blocking.sh"
  SRC="$BATS_TEST_TMPDIR/Mod-rt/src/com/x"; mkdir -p "$SRC"
}

# cls <name> <annotation line>... -- <member line>...: a Niagara class with the given annotations and members.
cls() {
  local d="$SRC/$1" f; shift; mkdir -p "$d"; f="$d/X.java"
  { printf '@NiagaraType\n'
    while [ "$#" -gt 0 ] && [ "$1" != "--" ]; do printf '%s\n' "$1"; shift; done
    shift
    printf 'public class X extends BComponent\n{\n'; printf '%s\n' "$@"; printf '}\n'; } > "$f"
}

@test "AB-usage: no argument exits 3 with the usage line on stderr" {
  run --separate-stderr "$LINT"
  [ "$status" -eq 3 ]
  # shellcheck disable=SC2154  # $stderr is set by bats run --separate-stderr
  [ "$stderr" = "usage: lint-async-blocking.sh [--strict] <src-root>..." ]
}

@test "AB-help: -h and --help print the header and exit 0, never a code line" {
  run "$LINT" -h
  [ "$status" -eq 0 ]
  [[ "$output" == "lint-async-blocking.sh — WARNs an ASYNC action"* ]]
  [ "$(printf '%s\n' "$output" | grep -cE '^(set -u|BIN=|exec |here=|GO_DIR=|case )')" -eq 0 ]
  run "$LINT" --help
  [ "$status" -eq 0 ]
}

@test "AB0: the Lab 9 and exam fixtures WARN once each on the do method's line, exit 0 (never FAIL)" {
  run "$LINT" "$FX/bad"
  [ "$status" -eq 0 ]
  [ "$(printf '%s\n' "$output" | wc -l)" -eq 2 ]
  [[ "$output" == *"WARN  lint-async-blocking  $FX/bad/com/angeles/Demo/BEngineSleep.java:13  ASYNC action \"execute\": doExecute blocks the shared engine thread (Thread.sleep() line 17) and the class has no post(Action...) override"* ]]
  [[ "$output" == *"WARN  lint-async-blocking  $FX/bad/com/angeles/Demo/BHistoryExport.java:15  ASYNC action \"export\": doExport blocks the shared engine thread (BQL .get() line 18, getOutputStream() line 19)"* ]]
  [[ "$output" != *FAIL* ]]
}

@test "AB2: the same sleeping handler behind a post() override handing a new Invocation is clean" {
  # Mutation: AB2 -- ignoring a post override flags the sanctioned Invocation offload
  run "$LINT" "$FX/good"
  [ "$status" -eq 0 ]
  [ -z "$output" ]
}

@test "AB1: a non-ASYNC action that sleeps is not judged; --strict turns the ASYNC WARN into exit 1" {
  # Mutation: AB1 -- judging a non-ASYNC action flags a handler the lint must not judge
  cls Plain '@NiagaraAction(name = "tick", flags = Flags.HIDDEN)' -- '  public void doTick() { Thread.sleep(5L); }'
  run "$LINT" "$SRC/Plain"
  [ "$status" -eq 0 ]
  [ -z "$output" ]
  cls Slow '@NiagaraAction(name = "tick", flags = Flags.ASYNC)' -- '  public void doTick() { Thread.sleep(5L); }'
  run "$LINT" --strict "$SRC/Slow"
  [ "$status" -eq 1 ]
  [[ "$output" == "WARN  lint-async-blocking  $SRC/Slow/X.java:5  ASYNC action \"tick\": doTick blocks the shared engine thread (Thread.sleep() line 5)"* ]]
}

@test "AB3: blocking text in a comment or a string literal is not a call" {
  # Mutation: AB3 -- reading blocking calls from comment or string text flags a quoted sleep
  cls Quoted '@NiagaraAction(name = "tick", flags = Flags.ASYNC)' -- '  public void doTick()' '  {' \
    '    // Thread.sleep(5L); c.openConnection();' '    String s = "Thread.sleep(5L); bql:select * x.get(1)";' '  }'
  run "$LINT" "$SRC/Quoted"
  [ "$status" -eq 0 ]
  [ -z "$output" ]
}

@test "AB4: post() that runs this::doX or a lambda to doX WARNs on that line; new Invocation does not" {
  # Mutation: AB4 -- dropping the `this::doX` check in post() lets the direct run through
  cls Post '@NiagaraAction(name = "submit", flags = Flags.ASYNC)' -- '  public void doSubmit() {}' \
    '  public IFuture post(Action a, BValue v, Context cx)' '  {' '    pool.execute(this::doSubmit);' \
    '    pool.submit(() -> doSubmit());' '    pool.execute(new Invocation(this, a, v, cx));' '    return null;' '  }'
  run "$LINT" "$SRC/Post"
  [ "$status" -eq 0 ]
  [ "$(printf '%s\n' "$output" | wc -l)" -eq 2 ]
  [[ "$output" == *"WARN  lint-async-blocking  $SRC/Post/X.java:8  post() runs this::doSubmit on another thread"* ]]
  [[ "$output" == *"WARN  lint-async-blocking  $SRC/Post/X.java:9  post() runs () -> doSubmit(...) on another thread"* ]]
}

@test "AB-fixture: the MinimalPan and MinimalDriver -rt sources are clean under --strict" {
  run "$LINT" --strict "$KIT/fixtures/MinimalPan/MinimalPan/MinimalPan-rt/src" "$KIT/fixtures/MinimalDriver/MinimalDriver/MinimalDriver-rt/src"
  [ "$status" -eq 0 ]
  [ -z "$output" ]
}

@test "AB-empty: a root with no *.java is an ERROR, exit 3" {
  mkdir -p "$BATS_TEST_TMPDIR/empty"
  run "$LINT" "$BATS_TEST_TMPDIR/empty"
  [ "$status" -eq 3 ]
  [ "$output" = "ERROR  lint-async-blocking  $BATS_TEST_TMPDIR/empty  no *.java sources found (wrong path or empty source tree)" ]
}
