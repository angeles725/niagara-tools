#!/usr/bin/env bats
# Kit doc pins for packaging and authoring files (register C23, C24, C25, C26, C29 — WU-P0-7). Each pin bans a
# statement the source contradicts, so a later edit cannot bring it back. retros/, odd/ and BUILD-STATE.md are
# evidence and are not scanned; types/ and METHODOLOGY.md are.
#   KP1  dist.xml noStation=true means "install only with every app stopped", never "without stopping"
#        (DistributionManifest.java:87, PlatformInstallManager.java:353-356; corpus B1318 §1318.2 C1, B1345)
#   KP2  -se means "depends on Java SE APIs" and runs on Windows-based hosts only, not "server-only" or a JACE
#        profile (devguide modules.txt:50; aPlat4PlatformAdminConfigureRuntimeProfiles.txt:30; corpus B1313 §1313.3 C3,
#        B1341 §1341.2 C1)
#   KP3  a doc PART of a code module is the devguide's form (deployingHelp.txt:8; corpus B1313 §1313.3 C1)
#   KP4  help.guide.base is optional; a vendor module never points it at docUser (deployingHelp.txt:28)
#   KP5  module-include.xml holds only <types>; <moduleParts> is generated from niagara-module.xml
#        (upgradingBuild.txt:95,115; corpus B1313 §1313.3 C2)
#   KP6  no kit doc links a §Templates section (none exists; corpus B1325 §1325.2 C1)
# [ev: retro devguide415-r2-packaging-platform-signing Δ1] [ev: retro devguide415-r3-packaging-extension-control Δ1]
# [ev: retro devguide415-horizons-authoring-security Δ6] [ev: retro devguide415-horizons-authoring-security Δ7]
# [ev: retro devguide415-horizons-authoring-security Δ8] [ev: retro devguide415-r2-packaging-platform-signing Δ4]

# PINS: "<id>|<ERE that must match nothing in SCAN>|<sample of the banned phrasing it must match>". Each pin is
# checked both ways: the pattern matches its sample (positive control: a mis-escaped regex cannot pass silently)
# and matches nothing in the scanned docs. grep exit 2 (missing scan root) fails the test, never passes it.
# KP4 matches the key anywhere on a line (indented code block, inline code span), not only at line start.
# shellcheck disable=SC2016  # the backticks are Markdown code spans in the patterns, not command substitution
PINS=(
  'KP1|installs without stopping|a noStation dist installs without stopping the station'
  'KP2|`-se` server-only|-se` \(server\)|JACE \*\*and\*\* Supervisor|`-se` modules load on JACE|the `-se` server-only profile'
  'KP3|never a part of a code module|help is never a part of a code module'
  'KP4|help\.guide\.base=module://docUser|    `help.guide.base=module://docUser/doc`'
  'KP5|lists its siblings in `<moduleParts>`|module-include.xml` — add the `-se` part|module-include.xml lists its siblings in `<moduleParts>`'
  'KP6|§ ?Templates|see §Templates for the palette'
)

setup() {
  KIT="$(cd "$BATS_TEST_DIRNAME/.." && pwd)/build-n4-module-kit"
  SCAN=("$KIT/types" "$KIT/METHODOLOGY.md")
}

# pin_parts <id> — sets RE and SAMPLE for one pin (the pattern may itself contain '|').
pin_parts() {
  local p
  for p in "${PINS[@]}"; do
    [ "${p%%|*}" = "$1" ] || continue
    p=${p#*|}; RE=${p%|*}; SAMPLE=${p##*|}; return 0
  done
  return 1
}

check_pin() {
  pin_parts "$1"
  run grep -qE -- "$RE" <<< "$SAMPLE"
  [ "$status" -eq 0 ]   # positive control: the pattern catches the phrasing it bans
  run grep -rniE -- "$RE" "${SCAN[@]}"
  [ "$status" -eq 1 ]   # and no kit doc carries it (2 = scan root missing → fail)
}

@test "KP1: no kit doc says noStation=true installs without stopping the station" { check_pin KP1; }
@test "KP2: no kit doc calls -se server-only or lists a JACE as an -se host" { check_pin KP2; }
@test "KP3: no kit doc forbids a doc part of a code module" { check_pin KP3; }
@test "KP4: no kit doc tells a vendor module to set help.guide.base to docUser" { check_pin KP4; }
@test "KP5: no kit doc puts <moduleParts> in module-include.xml" { check_pin KP5; }
@test "KP6: no kit doc links a §Templates section" { check_pin KP6; }
@test "KP-all: every pin id resolves to a pattern and a sample" {
  local id
  for id in KP1 KP2 KP3 KP4 KP5 KP6; do
    pin_parts "$id"
    [ -n "$RE" ]
    [ -n "$SAMPLE" ]
  done
}
