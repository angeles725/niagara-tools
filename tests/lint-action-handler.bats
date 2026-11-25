#!/usr/bin/env bats
# lint-action-handler.bats — a frozen @NiagaraAction needs its public do<X> handler of the action's return type,
# or the station refuses the type at load (ComplexIntrospector.java:316-343). The rule's golden cases live in
# go/internal/lint/rules/actionhandler. [ev: corpus B1231 §1231.4] [ev: ledger G1]

bats_require_minimum_version 1.5.0

setup() {
  KIT="$(cd "$BATS_TEST_DIRNAME/.." && pwd)/build-n4-module-kit"
  FX="$BATS_TEST_DIRNAME/fixtures/lint-action-handler"
  LINT="$KIT/toolbelt/lint-action-handler.sh"
  SRC="$BATS_TEST_TMPDIR/Mod-rt/src/com/x"; mkdir -p "$SRC"
  NODO='-- the station refuses the type with "Missing action do method" (ComplexIntrospector.java:316-343, types/actions.md § 1)'
}

# cls <name> <annotation line>... -- <member line>...: a Niagara class with the given annotations and members.
cls() {
  local d="$SRC/$1" f; shift; mkdir -p "$d"; f="$d/X.java"
  { printf '@NiagaraType\n'
    while [ "$#" -gt 0 ] && [ "$1" != "--" ]; do printf '%s\n' "$1"; shift; done
    shift
    printf 'public class X extends BComponent\n{\n'; printf '%s\n' "$@"; printf '}\n'; } > "$f"
}

@test "AH-usage: no argument exits 3 with the usage line on stderr" {
  run --separate-stderr "$LINT"
  [ "$status" -eq 3 ]
  # shellcheck disable=SC2154  # $stderr is set by bats run --separate-stderr
  [ "$stderr" = "usage: lint-action-handler.sh [--strict] <src-root>..." ]
}

@test "AH-help: -h and --help print the header and exit 0, never a code line" {
  run "$LINT" -h
  [ "$status" -eq 0 ]
  [[ "$output" == "lint-action-handler.sh — FAILs a Java class whose @NiagaraAction"* ]]
  [[ "$output" == *"Usage:  lint-action-handler.sh [--strict] <src-root>..."* ]]
  [ "$(printf '%s\n' "$output" | grep -cE '^(set -u|BIN=|exec |here=|GO_DIR=|case )')" -eq 0 ]
  run "$LINT" --help
  [ "$status" -eq 0 ]
}

@test "AH0: the RED fixture (doReset misspelled doReeset) FAILs on the annotation's name line, exit 1" {
  run "$LINT" "$FX/bad"
  [ "$status" -eq 1 ]
  [ "$output" = "FAIL  lint-action-handler  $FX/bad/com/angeles/Demo/BMisspelled.java:11  @NiagaraAction \"reset\" has no matching do method: declare public void doReset(Context cx), or doReset() $NODO" ]
}

@test "AH0-green: the same class with doReset spelled right is clean (exit 0, no output)" {
  run "$LINT" "$FX/good"
  [ "$status" -eq 0 ]
  [ -z "$output" ]
}

@test "AH1: a do method whose return type differs from the action's FAILs; the same type passes" {
  # Mutation: AH1 -- dropping the return-type comparison lets a do method that returns another type than the action through
  cls Bad '@NiagaraAction(name = "label", returnType = "BString")' -- '  public void doLabel() {}'
  cls Good '@NiagaraAction(name = "label", returnType = "BString")' -- '  public BString doLabel() { return null; }'
  run "$LINT" "$SRC/Bad"
  [ "$status" -eq 1 ]
  [[ "$output" == "FAIL  lint-action-handler  $SRC/Bad/X.java:5  @NiagaraAction \"label\": doLabel returns void but the action returns BString -- "* ]]
  [[ "$output" == *'"Action method and do method have mismatched signature" (ComplexIntrospector.java:341)' ]]
  run "$LINT" "$SRC/Good"
  [ "$status" -eq 0 ]
  [ -z "$output" ]
}

@test "AH2: a non-public do method is no handler (the introspector sees only public methods); a static one FAILs" {
  # Mutation: AH2 -- counting a non-public do method as the handler hides the missing one
  cls Hidden '@NiagaraAction(name = "tick")' -- '  void doTick() {}'
  cls Static '@NiagaraAction(name = "wipe")' -- '  public static void doWipe() {}'
  run "$LINT" "$SRC/Hidden"
  [ "$status" -eq 1 ]
  [ "$output" = "FAIL  lint-action-handler  $SRC/Hidden/X.java:2  @NiagaraAction \"tick\" has no matching do method (found: doTick() not public): declare public void doTick(Context cx), or doTick() $NODO" ]
  run "$LINT" "$SRC/Static"
  [ "$status" -eq 1 ]
  [[ "$output" == "FAIL  lint-action-handler  $SRC/Static/X.java:5  @NiagaraAction \"wipe\": doWipe must not be static -- "* ]]
}

@test "AH3: annotation and do-method text in a comment or a string literal is no declaration" {
  # Mutation: AH3 -- reading annotations or do methods from comment or string text flags or clears a quoted declaration
  cls Quoted '@NiagaraAction(name = "real")' -- '  // public void doReal() {}' '  String s = "public void doReal() {}";' \
    '  // @NiagaraAction(name = "ghost")'
  run "$LINT" "$SRC/Quoted"
  [ "$status" -eq 1 ]
  [[ "$output" == "FAIL  lint-action-handler  $SRC/Quoted/X.java:2  @NiagaraAction \"real\" has no matching do method:"* ]]
  [ "$(printf '%s\n' "$output" | wc -l)" -eq 1 ]
}

@test "AH4: doX(Context) beside doX() WARNs on the Context line, exit 0; --strict exits 1" {
  # Mutation: AH4 -- dropping the doX()/doX(Context) pair check lets the Context overload that never runs through
  cls Pair '@NiagaraAction(name = "reset")' -- '  public void doReset() {}' '  public void doReset(Context cx) {}'
  run "$LINT" "$SRC/Pair"
  [ "$status" -eq 0 ]
  [ "$output" = "WARN  lint-action-handler  $SRC/Pair/X.java:6  doReset(..., Context) never runs: the station binds the action to the no-Context doReset(...) it finds first; keep one handler (types/actions.md § 1)" ]
  run "$LINT" --strict "$SRC/Pair"
  [ "$status" -eq 1 ]
}

@test "AH5: this::doX of a declared action WARNs (course F3); the same reference to a plain helper is clean" {
  # Mutation: AH5 -- flagging `this::doX` of a method that is no declared action reports an ordinary helper
  cls Ref '@NiagaraAction(name = "reset")' -- '  public void doReset(Context cx) {}' '  public void doHelper() {}' \
    '  void a() { pool.execute(this::doReset); }' '  void b() { pool.execute(this::doHelper); }'
  run "$LINT" "$SRC/Ref"
  [ "$status" -eq 0 ]
  [ "$(printf '%s\n' "$output" | wc -l)" -eq 1 ]
  [[ "$output" == "WARN  lint-action-handler  $SRC/Ref/X.java:7  this::doReset runs the handler outside the action pipeline"* ]]
}

@test "AH-fixture: the MinimalPan and MinimalDriver -rt sources are clean (exit 0, no output)" {
  run "$LINT" "$KIT/fixtures/MinimalPan/MinimalPan/MinimalPan-rt/src" "$KIT/fixtures/MinimalDriver/MinimalDriver/MinimalDriver-rt/src"
  [ "$status" -eq 0 ]
  [ -z "$output" ]
}

@test "AH-empty: a root with no *.java is an ERROR, exit 3" {
  mkdir -p "$BATS_TEST_TMPDIR/empty"
  run "$LINT" "$BATS_TEST_TMPDIR/empty"
  [ "$status" -eq 3 ]
  [ "$output" = "ERROR  lint-action-handler  $BATS_TEST_TMPDIR/empty  no *.java sources found (wrong path or empty source tree)" ]
}
