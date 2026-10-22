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

setup() {
  KIT="$(cd "$BATS_TEST_DIRNAME/.." && pwd)/build-n4-module-kit"
  SCAN=("$KIT/types" "$KIT/METHODOLOGY.md")
}

@test "KP1: no kit doc says noStation=true installs without stopping the station" {
  run grep -rniE 'installs without stopping' "${SCAN[@]}"
  [ "$status" -eq 1 ]
}

@test "KP2: no kit doc calls -se server-only or lists a JACE as an -se host" {
  # shellcheck disable=SC2016  # the backticks are Markdown code spans in the pattern, not command substitution
  run grep -rniE '`-se` server-only|-se` \(server\)|JACE \*\*and\*\* Supervisor|`-se` modules load on JACE' "${SCAN[@]}"
  [ "$status" -eq 1 ]
}

@test "KP3: no kit doc forbids a doc part of a code module" {
  run grep -rniE 'never a part of a code module' "${SCAN[@]}"
  [ "$status" -eq 1 ]
}

@test "KP4: no kit doc tells a vendor module to set help.guide.base to docUser" {
  run grep -rnE '^help\.guide\.base=module://docUser' "${SCAN[@]}"
  [ "$status" -eq 1 ]
}

@test "KP5: no kit doc puts <moduleParts> in module-include.xml" {
  # shellcheck disable=SC2016  # the backticks are Markdown code spans in the pattern, not command substitution
  run grep -rnE 'lists its siblings in `<moduleParts>`|module-include.xml` — add the `-se` part' "${SCAN[@]}"
  [ "$status" -eq 1 ]
}

@test "KP6: no kit doc links a §Templates section" {
  run grep -rnE '§ ?Templates' "${SCAN[@]}"
  [ "$status" -eq 1 ]
}
