#!/usr/bin/env bats
# lint-servlet-write-guards.sh — the wrapper contract, and the types/servlet-write.md template linted by it
# and by lint-servlet, so the template and the lints cannot drift apart. The rule's own cases are the Go
# goldens (go/internal/lint/rules/servletwriteguard/testdata). [ev: retro 2026-11-10-mcpn4-history-batch-tools Δ6]

setup() {
  KIT="$(cd "$BATS_TEST_DIRNAME/.." && pwd)/build-n4-module-kit"
  LINT="$KIT/toolbelt/lint-servlet-write-guards.sh"
  SRC="$BATS_TEST_TMPDIR/src"; mkdir -p "$SRC"
}

# template: the first ```java block under "## Template" of types/servlet-write.md, as a source file
template() {
  awk '/^## Template/{t=1} t&&/^```java$/{j=1;next} j&&/^```$/{exit} j' "$KIT/types/servlet-write.md" > "$SRC/BPanelServlet.java"
  [ -s "$SRC/BPanelServlet.java" ]
}

@test "SWG1: the types/servlet-write.md template is clean under --strict (no row, exit 0)" {
  template
  run "$LINT" --strict "$SRC"
  [ "$status" -eq 0 ]
  [ -z "$output" ]
}

@test "SWG2: the template has no FAIL or WARN row in lint-servlet either" {
  template
  run "$KIT/toolbelt/lint-servlet.sh" "$SRC"
  [ "$status" -eq 0 ]
  [[ "$output" != *"FAIL"* ]] && [[ "$output" != *"WARN"* ]]
}

@test "SWG3: a permission read off the servlet itself FAILs (exit 1) and names the template" {
  printf 'class A extends BWebServlet {\n  void doPost(WebOp op) {\n    CsrfUtil.verifyCsrfToken(op.getRequest());\n    if (!getPermissionsFor(this).hasOperatorWrite()) return;\n    t.set(p, v, op);\n  }\n}\n' > "$SRC/A.java"
  run "$LINT" "$SRC"
  [ "$status" -eq 1 ]
  [[ "$output" == *"FAIL  self-permission  $SRC/A.java:4"* ]]
  [[ "$output" == *"types/servlet-write.md"* ]]
}

@test "SWG4: a root with no *.java is an env error (exit 3), never a clean pass" {
  run "$LINT" "$SRC"
  [ "$status" -eq 3 ]
}
