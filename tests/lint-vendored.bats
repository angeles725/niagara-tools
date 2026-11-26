#!/usr/bin/env bats
# Tests for build-n4-module-kit/toolbelt/lint-vendored.sh (issue #1123): rc/vendor/VENDORED.md lists each vendored
# file (file, library, version, source URL); a listed file whose version is in the kit's known-defects table
# (data/vendored-defects.md) and a vendor file no row lists are WARN. These run the wrapper with the kit's real table.

setup() {
  KIT="$(cd "$BATS_TEST_DIRNAME/.." && pwd)/build-n4-module-kit"
  LINT="$KIT/toolbelt/lint-vendored.sh"
  RC="$BATS_TEST_TMPDIR/rc"; V="$RC/vendor"; mkdir -p "$V"
}

# ledger <row>...: VENDORED.md with the given `| file | library | version | url |` rows.
ledger() {
  { printf '| File | Library | Version | Source URL |\n|---|---|---|---|\n'; printf '%s\n' "$@"; } > "$V/VENDORED.md"
}

@test "VN-usage: no argument exits 3 with the usage line" {
  run "$LINT"
  [ "$status" -eq 3 ]
  [[ "$output" == *"usage: lint-vendored.sh"* ]]
}

@test "VN0: a listed three.js r160 OrbitControls WARNs with the defect and the fix, exit 0; --strict exits 1" {
  ledger '| OrbitControls.js | three.js | r160 | https://example.org/three/r160/OrbitControls.js |'
  : > "$V/OrbitControls.js"
  run "$LINT" "$RC"
  [ "$status" -eq 0 ]
  [[ "$output" == "WARN  lint-vendored  $V/OrbitControls.js  three.js r160 (VENDORED.md:3) has a known defect: "* ]]
  [[ "$output" == *"devicePixelRatio < 1"* ]]
  [[ "$output" == *"fix: upgrade to >= r161" ]]
  run "$LINT" --strict "$RC"
  [ "$status" -eq 1 ]
}

@test "VN1: r-numbers compare as numbers: r99 is below r161 (flagged), r161 and r185 are clean" {
  # Mutation: VN1 -- read the version as text so r99 sorts above r161 and r160 stops being flagged
  ledger '| a/OrbitControls.js | three.js | r99 | https://example.org/three |' \
    '| b/OrbitControls.js | three.js | r161 | https://example.org/three |' \
    '| c/OrbitControls.js | three.js | r185 | https://example.org/three |'
  mkdir -p "$V/a" "$V/b" "$V/c"; : > "$V/a/OrbitControls.js"; : > "$V/b/OrbitControls.js"; : > "$V/c/OrbitControls.js"
  run "$LINT" "$RC"
  [ "$status" -eq 0 ]
  [ "$(printf '%s\n' "$output" | wc -l)" -eq 1 ]
  [[ "$output" == "WARN  lint-vendored  $V/a/OrbitControls.js  three.js r99 "* ]]
}

@test "VN2: THIRD-PARTY.md and LICENSE files are notes, not unlisted vendor code; an unlisted file and a missing ledger are WARN" {
  # Mutation: VN2 -- count THIRD-PARTY.md and LICENSE files as vendored code and flag them unlisted
  ledger '| listed.min.js | other-lib | 2.0.0 | https://example.org/other |'
  : > "$V/listed.min.js"; : > "$V/THIRD-PARTY.md"; : > "$V/LICENSE"; : > "$V/LICENSE-other.txt"
  run "$LINT" "$RC"
  [ "$status" -eq 0 ]
  [ -z "$output" ]
  : > "$V/extra.min.js"
  run "$LINT" "$RC"
  [ "$status" -eq 0 ]
  [[ "$output" == "WARN  lint-vendored  $V/extra.min.js  not listed in rc/vendor/VENDORED.md; add a row: "* ]]
  rm "$V/VENDORED.md"
  run "$LINT" "$RC"
  [ "$status" -eq 0 ]
  [ "$(printf '%s\n' "$output" | wc -l)" -eq 1 ]
  [[ "$output" == "WARN  lint-vendored  $V  has vendored files but no VENDORED.md; create it with the table | File | Library | Version | Source URL |"* ]]
}

@test "VN3: an rc dir with no vendor/ is clean, exit 0, no output" {
  mkdir -p "$BATS_TEST_TMPDIR/rc2"; : > "$BATS_TEST_TMPDIR/rc2/app.js"
  run "$LINT" "$BATS_TEST_TMPDIR/rc2"
  [ "$status" -eq 0 ]
  [ -z "$output" ]
}
