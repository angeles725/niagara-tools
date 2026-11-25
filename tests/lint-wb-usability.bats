#!/usr/bin/env bats
# lint-wb-usability.bats — a -wb view (extends BWbComponentView) that holds a BOrd in a file with no BComponentChooser and
# no targetType facet WARNs; the narrow part of the lint candidate in types/wb-widgets.md (UX rule). The rule's golden
# cases live in go/internal/lint/rules/wbusability.

setup() {
  KIT="$(cd "$BATS_TEST_DIRNAME/.." && pwd)/build-n4-module-kit"
  L="$KIT/toolbelt/lint-wb-usability.sh"
  FX="$BATS_TEST_DIRNAME/fixtures/lint-wb-usability"
  ROOT="$BATS_TEST_TMPDIR/wb"
  mkdir -p "$ROOT"
}

# view <file-body...>: writes $ROOT/BFooView.java, one argument per line.
view() { printf '%s\n' "$@" > "$ROOT/BFooView.java"; }

@test "WBU-usage: no argument prints the usage and exits 3" {
  run "$L"
  [ "$status" -eq 3 ]
  [ "$output" = "usage: lint-wb-usability.sh [--strict] <wb-src-root>" ]
}

@test "WBU-empty: a root with no *.java is an env error, never a clean pass" {
  run "$L" "$ROOT"
  [ "$status" -eq 3 ]
  [ "$output" = "ERROR  lint-wb-usability  $ROOT  no *.java to scan (K20: wrong path or empty scaffold)" ]
}

@test "WBU1: the warn fixtures give one WARN each, at the class name, and exit 0" {
  run "$L" "$FX/warn"
  [ "$status" -eq 0 ]
  [ "${#lines[@]}" -eq 2 ]
  [[ "${lines[0]}" == "WARN  lint-wb-usability  $FX/warn/com/example/wb/BSlotView.java:7  BSlotView extends BWbComponentView and holds a BOrd (@NiagaraProperty slot source), but its file has no BComponentChooser and no targetType facet: "* ]]
  [[ "${lines[1]}" == "WARN  lint-wb-usability  $FX/warn/com/example/wb/BTargetView.java:7  BTargetView extends BWbComponentView and holds a BOrd (field target), but its file has no BComponentChooser and no targetType facet: "* ]]
  [[ "$output" == *"let the Add and Edit flow pick it from the station (types/wb-widgets.md UX rule)" ]]
}

@test "WBU1-strict: --strict turns the WARN into exit 1 and keeps the rows" {
  run "$L" --strict "$FX/warn"
  [ "$status" -eq 1 ]
  [ "${#lines[@]}" -eq 2 ]
}

@test "WBU2: the clean fixtures (chooser, targetType facet, waiver, another base) are clean under --strict" {
  run "$L" --strict "$FX/clean"
  [ "$status" -eq 0 ]
  [ -z "$output" ]
}

@test "WBU-import: importing BComponentChooser is not opening it; calling it is" {
  # Mutation: WBU-import -- counting an import line as a BComponentChooser invocation clears a view that only imports it
  view 'import javax.baja.workbench.mgr.BComponentChooser;' 'class BFooView extends BWbComponentView {' '  BOrd target;' '}'
  run "$L" --strict "$ROOT"
  [ "$status" -eq 1 ]
  [[ "$output" == "WARN  lint-wb-usability  $ROOT/BFooView.java:2  BFooView extends BWbComponentView"* ]]
  view 'import javax.baja.workbench.mgr.BComponentChooser;' 'class BFooView extends BWbComponentView {' '  BOrd target;' \
    '  void pick() { BComponentChooser.prompt(a, b, c, d); }' '}'
  run "$L" --strict "$ROOT"
  [ "$status" -eq 0 ]
  [ -z "$output" ]
}

@test "WBU-static: a static BOrd constant is not a user-managed reference" {
  # Mutation: WBU-static -- counting a static BOrd constant as a member WARNs a view that only names a fixed ORD
  view 'class BFooView extends BWbComponentView {' '  static final BOrd HOME = BOrd.make("station:|slot:/");' '}'
  run "$L" --strict "$ROOT"
  [ "$status" -eq 0 ]
  [ -z "$output" ]
}

@test "WBU-waiver-string: a waiver needs a reason and a comment; a string literal does not waive" {
  # Mutation: WBU-waiver-string -- reading the waiver from the code view too lets a string literal waive a class
  view '@Named("lint-wb-usability: waived not a comment")' 'class BFooView extends BWbComponentView {' '  BOrd target;' '}'
  run "$L" --strict "$ROOT"
  [ "$status" -eq 1 ]
  view '// lint-wb-usability: waived' 'class BFooView extends BWbComponentView {' '  BOrd target;' '}'
  run "$L" --strict "$ROOT"
  [ "$status" -eq 1 ]
  view '// lint-wb-usability: waived set by the wizard' 'class BFooView extends BWbComponentView {' '  BOrd target;' '}'
  run "$L" --strict "$ROOT"
  [ "$status" -eq 0 ]
}

@test "WBU-base: only a BWbComponentView subclass is judged" {
  # Mutation: WBU-base -- matching any base class WARNs a BWbView or a BComponent that holds a BOrd
  view 'class BFooView extends BComponent {' '  BOrd target;' '}'
  run "$L" --strict "$ROOT"
  [ "$status" -eq 0 ]
  [ -z "$output" ]
}

@test "WBU-sibling: lint-wb-file-chooser still runs on the same tree (shared engine registration untouched)" {
  run "$KIT/toolbelt/lint-wb-file-chooser.sh" "$FX/clean"
  [ "$status" -eq 0 ]
  [ -z "$output" ]
}
