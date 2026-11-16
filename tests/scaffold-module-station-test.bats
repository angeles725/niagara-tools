#!/usr/bin/env bats
# scaffold-module: the station-test overlay of logic and dashboard modules.
# Shared setup (the Go binary, KIT, SC, the fixture and golden paths, OUT): tests/lib/scaffold-setup.bash.

load lib/scaffold-setup

# ---------------------------------------------------------------------------------------------------------
# Station-test overlay (round 10): logic + dashboard get a BTestNgStation skeleton from templates/station-test
# (Fox-off fixture base, a link-in sample test, moduleTest-include.xml). The driver keeps its own loopback
# test (TC-DRV7) and gets no overlay. Doctrine: types/moduleTest.md, types/driver-authoring.md § 12.1.
# ---------------------------------------------------------------------------------------------------------

# st_golden_diff <type> <Mod>: the emitted overlay files of <Mod>-rt equal the golden for <type>.
st_golden_diff() {
  local t="$1" mod="$2"
  local r="$OUT/$mod/$mod/$mod-rt" g="$ST_GOLDEN/$t"
  diff "$g/moduleTest-include.xml" "$r/moduleTest-include.xml" || return 1
  diff "$g/BModuleAStationTestBase.java" "$r/srcTest/test/com/example/$mod/B${mod}StationTestBase.java" || return 1
  diff "$g/BModuleALinkTest.java" "$r/srcTest/test/com/example/$mod/B${mod}LinkTest.java" || return 1
}

@test "TC-ST1: --type logic emits the station-test overlay equal to its golden, with both test dependencies" {
  # Named mutation: skip the overlay copy for logic in scaffold-module.sh -> the diff finds no files and fails.
  run "$SC" ModuleA "$OUT" --type logic --vendor Example
  [ "$status" -eq 0 ]
  run st_golden_diff logic ModuleA
  [ "$status" -eq 0 ]
  G="$OUT/ModuleA/ModuleA/ModuleA-rt/ModuleA-rt.gradle.kts"
  grep -q 'moduleTestImplementation(":test-wb")' "$G"
  grep -q 'moduleTestImplementation("junit:junit:4.13.2")' "$G"
}

@test "TC-ST2: --type dashboard emits the station-test overlay equal to its golden, with both test dependencies" {
  # Named mutation: drop "status" from the dashboard link-in list -> the golden diff fails.
  run "$SC" ModuleA "$OUT" --type dashboard --vendor Example
  [ "$status" -eq 0 ]
  run st_golden_diff dashboard ModuleA
  [ "$status" -eq 0 ]
  G="$OUT/ModuleA/ModuleA/ModuleA-rt/ModuleA-rt.gradle.kts"
  grep -q 'moduleTestImplementation(":test-wb")' "$G"
  grep -q 'moduleTestImplementation("junit:junit:4.13.2")' "$G"
  # the overlay lands in -rt only, never in -ux
  run find "$OUT/ModuleA/ModuleA/ModuleA-ux" -name moduleTest-include.xml
  [ -z "$output" ]
}

@test "TC-ST3: a renamed overlay leaves no template token and registers its tests for the test jar only" {
  # Named mutation: drop the com.example -> com.<vendor> substitution -> the token grep finds com.example.
  for t in logic dashboard; do
    run "$SC" ColdBox "$OUT/$t" --type "$t" --vendor Acme
    [ "$status" -eq 0 ]
    R="$OUT/$t/ColdBox/ColdBox/ColdBox-rt"
    T="$R/srcTest/test/com/acme/ColdBox"
    [ -f "$T/BColdBoxStationTestBase.java" ]
    [ -f "$T/BColdBoxLinkTest.java" ]
    run grep -rlE 'ModuleA|com\.example|@LINK_IN_SLOTS@|Example' "$R/moduleTest-include.xml" "$T"
    [ "$status" -eq 1 ]
    grep -q 'class="test.com.acme.ColdBox.BColdBoxLinkTest"' "$R/moduleTest-include.xml"
    grep -q 'import com.acme.ColdBox.BColdBox;' "$T/BColdBoxLinkTest.java"
    run grep -c 'LinkTest\|StationTestBase' "$R/module-include.xml"
    [ "$output" = "0" ]
  done
}

@test "TC-ST4: the driver gets no station-test overlay (its loopback test stays the only station test)" {
  run "$SC" AcmeLink "$OUT" --type driver --vendor Example
  [ "$status" -eq 0 ]
  run find "$OUT/AcmeLink" -name 'B*StationTestBase.java' -o -name 'B*LinkTest.java'
  [ -z "$output" ]
}

@test "TC-ST5: the overlay fixture turns Fox off and tolerates only the exact Fox-not-serving failure" {
  # Named mutation: catch every Exception in startStation (drop the message check) -> the grep for the
  # rethrow guard fails.
  run "$SC" ModuleA "$OUT" --type logic --vendor Example
  [ "$status" -eq 0 ]
  B="$OUT/ModuleA/ModuleA/ModuleA-rt/srcTest/test/com/example/ModuleA/BModuleAStationTestBase.java"
  for slot in foxEnabled foxsEnabled multicastEnabled enableAnnouncement; do
    grep -q "fox.set(\"$slot\", BBoolean.FALSE);" "$B"
  done
  grep -q 'static final String FOX_NOT_STARTED = "Fox service did not start in time";' "$B"
  grep -q 'if (!FOX_NOT_STARTED.equals(e.getMessage()) || !station.isRunning()) throw e;' "$B"
  grep -q '@AfterMethod(alwaysRun = true)' "$B"
}

# st_link_in_flags <rt-dir> <Mod> <vendor-lc>: one "<slot> <flags>" line per LINK_IN_SLOTS entry of the
# emitted link test, the flags read from that slot's @NiagaraProperty block in the emitted component
# ("MISSING" when the component declares no such property).
st_link_in_flags() {
  local r="$1" mod="$2" v="$3"
  local test="$r/srcTest/test/com/$v/$mod/B${mod}LinkTest.java" comp="$r/src/com/$v/$mod/B${mod}.java"
  local slots
  slots="$(sed -n 's/.*LINK_IN_SLOTS = { \(.*\) };.*/\1/p' "$test" | tr -d '" ' | tr ',' ' ')"
  [ -n "$slots" ] || return 1
  for s in $slots; do
    awk -v want="$s" '
      /@NiagaraProperty\(/ { inblk = 1; name = ""; flags = ""; next }
      inblk && /name *= *"/ { n = $0; sub(/.*name *= *"/, "", n); sub(/".*/, "", n); name = n }
      inblk && /flags *=/ { f = $0; sub(/.*flags *= */, "", f); flags = f }
      inblk && /^\)/ { if (name == want) { print want " " flags; found = 1 } inblk = 0 }
      END { if (!found) print want " MISSING" }
    ' "$comp"
  done
}

@test "TC-ST6: no link-in slot named by the emitted link test is READONLY in the emitted component (source-level guard)" {
  # The station test asserts the same at run time (linkInSlotsAreNotReadonly): LinkCheck refuses a
  # READONLY target (javax.baja.sys.LinkCheck.make, linkcheck.propReadonly), types/dashboard.md link-in rule.
  # Named mutation: add Flags.READONLY to the dashboard fixture's status slot -> this test fails.
  for t in logic dashboard; do
    run "$SC" ModuleA "$OUT/$t" --type "$t" --vendor Example
    [ "$status" -eq 0 ]
    run st_link_in_flags "$OUT/$t/ModuleA/ModuleA/ModuleA-rt" ModuleA example
    [ "$status" -eq 0 ]
    [[ "$output" != *MISSING* ]]
    [[ "$output" != *READONLY* ]]
  done
}
