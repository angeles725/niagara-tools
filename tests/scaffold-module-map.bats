#!/usr/bin/env bats
# scaffold-module: the module map and modular layout of a fresh scaffold of every type.
# Shared setup (the Go binary, KIT, SC, the fixture and golden paths, OUT): tests/lib/scaffold-setup.bash.

load lib/scaffold-setup

# ---------------------------------------------------------------------------------------------------------
# Module map (modularity campaign M3): every scaffold type emits a strict MODULE-MAP.md, and the Baja-free
# classes live in role packages (model/ for rt, route/ for ux). [ev: retro modularity-module-map Δ2]
# [ev: retro modularity-strict-new Δ2]
# ---------------------------------------------------------------------------------------------------------

@test "TC-MAP1: a fresh scaffold of every type passes lint-module-map with no row at all (0 FAIL, 0 WARN)" {
  # Named mutation: drop the control row from the logic fixture map -> MM4 FAIL rows for its files, exit 1.
  for t in logic dashboard driver; do
    run "$SC" ModuleA "$OUT/$t" --type "$t" --vendor Example
    [ "$status" -eq 0 ]
    root="$OUT/$t/ModuleA/ModuleA"
    [ -f "$root/MODULE-MAP.md" ] || { echo "no map ($t)"; false; }
    grep -qx 'modularity: strict' "$root/MODULE-MAP.md"
    run "$KIT/toolbelt/lint-module-map.sh" "$root"
    [ "$status" -eq 0 ] || { echo "$t: exit $status"; echo "$output"; false; }
    [ -z "$output" ] || { echo "$t: $output"; false; }
  done
  # positive control: an unlisted source file FAILs, so the lint reached this exact root in strict mode
  printf 'package com.example.ModuleA.model;\npublic final class Extra {}\n' \
    > "$OUT/logic/ModuleA/ModuleA/ModuleA-rt/src/com/example/ModuleA/model/Extra.java"
  run "$KIT/toolbelt/lint-module-map.sh" "$OUT/logic/ModuleA/ModuleA"
  [ "$status" -eq 1 ]
  [[ "$output" == *"FAIL  lint-module-map  $OUT/logic/ModuleA/ModuleA/ModuleA-rt/src/com/example/ModuleA/model/Extra.java  MM4: "* ]]
}

@test "TC-MAP2: the Baja-free fixture classes sit in role packages and their pure tests run there (SKIP without javac or junit)" {
  run "$SC" ModuleA "$OUT/logic" --type logic --vendor Example
  [ "$status" -eq 0 ]
  run "$SC" ModuleA "$OUT/dashboard" --type dashboard --vendor Example
  [ "$status" -eq 0 ]
  run "$SC" ModuleA "$OUT/driver" --type driver --vendor Example
  [ "$status" -eq 0 ]
  L="$OUT/logic/ModuleA/ModuleA/ModuleA-rt" D="$OUT/dashboard/ModuleA/ModuleA" V="$OUT/driver/ModuleA/ModuleA/ModuleA-rt"
  grep -q '^package com.example.ModuleA.model;' "$L/src/com/example/ModuleA/model/ModuleALogic.java"
  grep -q '^package com.example.ModuleA.model;' "$D/ModuleA-rt/src/com/example/ModuleA/model/ModuleALogic.java"
  grep -q '^package com.example.ModuleA.ux.route;' "$D/ModuleA-ux/src/com/example/ModuleA/ux/route/ModuleADispatch.java"
  grep -q '^package com.example.ModuleA.model;' "$V/src/com/example/ModuleA/model/ModuleACodec.java"
  # the ux module-include names the servlet at its route/ package (DUX1: the router stays package-private beside it)
  grep -q 'class="com.example.ModuleA.ux.route.BModuleAServlet"' "$D/ModuleA-ux/module-include.xml"
  # no Baja-free class is left in a module root package
  run find "$L/src/com/example/ModuleA" "$D/ModuleA-rt/src/com/example/ModuleA" "$V/src/com/example/ModuleA" \
    -maxdepth 1 \( -name '*Logic.java' -o -name '*Codec.java' \)
  [ -z "$output" ]
  command -v javac >/dev/null 2>&1 || skip "no javac on PATH"
  [ -n "$(find "$HOME/.gradle" -name 'junit-4.13.2.jar' 2>/dev/null | head -1)" ] || skip "junit-4.13.2.jar not in ~/.gradle"
  for pair in "$L com.example.ModuleA.model.ModuleALogicTest" \
              "$D/ModuleA-rt com.example.ModuleA.model.ModuleALogicTest" \
              "$D/ModuleA-ux com.example.ModuleA.ux.route.ModuleADispatchTest" \
              "$V com.example.ModuleA.model.ModuleACodecTest"; do
    run "$KIT/toolbelt/run-pure-test.sh" ${pair% *} ${pair#* }
    [ "$status" -eq 0 ] || { echo "$pair: $output"; false; }
    [[ "$output" == *"OK ("* ]]
  done
}

@test "TC-MAP3: a fresh scaffold of every type passes strict lint-structure L3/L3b (no L3 row), and a stray pure class FAILs" {
  # Named mutation: report strict L3 only for -rt parts -> the dashboard's ux control below stays silent.
  for t in logic dashboard driver; do
    run "$SC" ModuleA "$OUT/$t" --type "$t" --vendor Example
    [ "$status" -eq 0 ]
    mkdir -p "$OUT/$t/ModuleA/.git"            # the L10/L14 upward walk stops here
    run "$KIT/toolbelt/lint-structure.sh" "$OUT/$t/ModuleA/ModuleA"
    [ "$status" -eq 0 ] || { echo "$t: $output"; false; }
    [[ "$output" != *"L3"* ]] || { echo "$t: $output"; false; }
  done
  # positive controls: a Baja-free class beside the servlet's package, and a model class without its test
  U="$OUT/dashboard/ModuleA/ModuleA/ModuleA-ux/src/com/example/ModuleA/ux"
  printf 'package com.example.ModuleA.ux;\npublic final class Stray {}\n' > "$U/Stray.java"
  printf 'package com.example.ModuleA.model;\npublic final class Lonely {}\n' \
    > "$OUT/dashboard/ModuleA/ModuleA/ModuleA-rt/src/com/example/ModuleA/model/Lonely.java"
  run "$KIT/toolbelt/lint-structure.sh" "$OUT/dashboard/ModuleA/ModuleA"
  [ "$status" -eq 1 ]
  [[ "$output" == *"FAIL  lint-structure  ModuleA-ux/src/com/example/ModuleA/ux/Stray.java  L3: Baja-free class outside a role package"* ]]
  [[ "$output" == *"FAIL  lint-structure  ModuleA-rt/src/com/example/ModuleA/model/Lonely.java  L3: model/ class Lonely has no srcTest LonelyTest.java"* ]]
}

@test "TC-MAP4: every part of a fresh scaffold passes lint-size with no row, and a long B-class method FAILs in the strict module" {
  # Named mutation: keep the method rule pure-class-only in a strict module -> the BComponent control below passes.
  for t in logic dashboard driver; do
    run "$SC" ModuleA "$OUT/$t" --type "$t" --vendor Example
    [ "$status" -eq 0 ]
    for src in "$OUT/$t"/ModuleA/ModuleA/*/src; do
      run "$KIT/toolbelt/lint-size.sh" "$src"
      [ "$status" -eq 0 ] || { echo "$src: $output"; false; }
      [ -z "$output" ] || { echo "$src: $output"; false; }
    done
  done
  S="$OUT/logic/ModuleA/ModuleA/ModuleA-rt/src/com/example/ModuleA"
  { printf 'package com.example.ModuleA;\npublic class BLong extends BComponent\n{\n  public void changed()\n  {\n'
    for _ in $(seq 81); do printf '    x();\n'; done
    printf '  }\n}\n'; } > "$S/BLong.java"
  run "$KIT/toolbelt/lint-size.sh" "$OUT/logic/ModuleA/ModuleA/ModuleA-rt/src"
  [ "$status" -eq 1 ]
  [[ "$output" == *"FAIL  lint-size  $S/BLong.java:4  method changed() in class BLong spans 84 lines (> 80)"* ]]
}

@test "TC-MAP5: a fresh dashboard scaffold emits the rc/js skeleton (config, services/api, store, components, pages, main) with fetch only in services/api.js" {
  # Named mutation: keep the inline poll <script> in index.html -> the fetch( count outside services/ is not 0.
  run "$SC" ModuleA "$OUT/dashboard" --type dashboard --vendor Example
  [ "$status" -eq 0 ]
  RC="$OUT/dashboard/ModuleA/ModuleA/ModuleA-ux/src/rc"
  for f in js/config.js js/services/api.js js/store.js js/main.js; do
    [ -f "$RC/$f" ] || { echo "missing $f"; false; }
  done
  [ -n "$(find "$RC/js/components" -name '*.js' 2>/dev/null)" ] || { echo "no js/components/*.js"; false; }
  [ -n "$(find "$RC/js/pages" -name '*.js' 2>/dev/null)" ] || { echo "no js/pages/*.js"; false; }
  # index.html loads the scripts in order and holds no inline <script> block
  run grep -c '<script src="js/' "$RC/index.html"
  [ "$output" -ge 6 ]
  run grep -E '<script>' "$RC/index.html"
  [ "$status" -eq 1 ]
  # the only fetch( call is in services/api.js
  run grep -rlE '(^|[^A-Za-z0-9_$])fetch[[:space:]]*\(' "$RC"
  [ "$output" = "$RC/js/services/api.js" ] || { echo "$output"; false; }
  # rc-scan reports no row on the fresh ux part
  run "$KIT/toolbelt/rc-scan.sh" "$OUT/dashboard/ModuleA/ModuleA/ModuleA-ux"
  [ "$status" -eq 0 ] || { echo "$output"; false; }
  [ -z "$output" ] || { echo "$output"; false; }
  # positive control: the fresh module is strict, so a page that fetches FAILs rc-scan fetch-outside-services
  printf '%s\n' "fetch(MinimalDashConfig.apiBase, { signal: s });" >> "$RC/js/pages/home.js"
  run "$KIT/toolbelt/rc-scan.sh" "$OUT/dashboard/ModuleA/ModuleA/ModuleA-ux"
  [ "$status" -eq 1 ]
  [[ "$output" == *"FAIL  rc-scan  src/rc/js/pages/home.js:"*"  fetch-outside-services: "* ]]
}

@test "TC-MAP6: a fresh dashboard servlet has no handler-size row, and a servlet method over 80 lines FAILs in the strict module" {
  # Named mutation: read the module map mode as legacy in lint-servlet -> the long handler below stays WARN, exit 0.
  run "$SC" ModuleA "$OUT/dashboard" --type dashboard --vendor Example
  [ "$status" -eq 0 ]
  UX="$OUT/dashboard/ModuleA/ModuleA/ModuleA-ux/src"
  run "$KIT/toolbelt/lint-servlet.sh" "$UX"
  [[ "$output" != *"handler-size"* ]] || { echo "$output"; false; }
  S="$UX/com/example/ModuleA/ux/route/BModuleAServlet.java"
  { head -n -1 "$S"
    printf '  private void handleWrite(WebOp op)\n  {\n'
    for _ in $(seq 80); do printf '    op.getRequest();\n'; done
    printf '  }\n}\n'; } > "$S.new" && mv "$S.new" "$S"
  run "$KIT/toolbelt/lint-servlet.sh" "$UX"
  [ "$status" -eq 1 ]
  [[ "$output" == *"FAIL  handler-size  com/example/ModuleA/ux/route/BModuleAServlet.java:"*"  method handleWrite() spans 83 lines (> 80) inside the servlet"* ]] || { echo "$output"; false; }
}

@test "TC-MAP7: a fresh dashboard ships the ux-rt contract (rc/js/contract.json), a StatusHandler the servlet delegates to, and a contract test on each side that bites" {
  run "$SC" ModuleA "$OUT/dashboard" --type dashboard --vendor Example
  [ "$status" -eq 0 ]
  mkdir -p "$OUT/dashboard/ModuleA/.git"
  R="$OUT/dashboard/ModuleA/ModuleA" U="$OUT/dashboard/ModuleA/ModuleA/ModuleA-ux"
  [ -f "$U/src/rc/js/contract.json" ]
  [ -f "$U/src/com/example/ModuleA/ux/route/StatusHandler.java" ]
  grep -q 'StatusHandler.respond()' "$U/src/com/example/ModuleA/ux/route/BModuleAServlet.java"
  run "$KIT/toolbelt/lint-module-map.sh" "$R"
  [ "$status" -eq 0 ] && [ -z "$output" ] || { echo "$output"; false; }
  run "$KIT/toolbelt/lint-servlet.sh" "$U/src" --strict
  [ "$status" -eq 0 ] || { echo "$output"; false; }
  run "$KIT/toolbelt/lint-structure.sh" "$R"
  [ "$status" -eq 0 ] && [[ "$output" != *"L3"* ]] || { echo "$output"; false; }
  command -v node >/dev/null 2>&1 || skip "no node on PATH"
  run node "$U/srcTest/rc/spec/ModuleAContractSpec.js"
  [ "$status" -eq 0 ] || { echo "$output"; false; }
  # positive control: the contract loses the field the consumer reads -> the JS side fails
  sed -i 's/"status": "string"/"state": "string"/' "$U/src/rc/js/contract.json"
  run node "$U/srcTest/rc/spec/ModuleAContractSpec.js"
  [ "$status" -ne 0 ]
  [[ "$output" == *"field status"* ]]
  command -v javac >/dev/null 2>&1 || skip "no javac on PATH"
  [ -n "$(find "$HOME/.gradle" -name 'junit-4.13.2.jar' 2>/dev/null | head -1)" ] || skip "junit-4.13.2.jar not in ~/.gradle"
  # the servlet side reads the same file: its handler no longer produces the contract's field -> RED
  run "$KIT/toolbelt/run-pure-test.sh" "$U" com.example.ModuleA.ux.route.StatusHandlerTest
  [ "$status" -eq 1 ]
  sed -i 's/"state": "string"/"status": "string"/' "$U/src/rc/js/contract.json"
  run "$KIT/toolbelt/run-pure-test.sh" "$U" com.example.ModuleA.ux.route.StatusHandlerTest
  [ "$status" -eq 0 ] || { echo "$output"; false; }
}

