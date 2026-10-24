#!/usr/bin/env bats
# Unit tests for build-n4-module-kit/toolbelt/lib/vendor-version.sh — the ONE module vendorVersion reader and
# comparator that toolbelt/build.sh's drift gate and scripts/ng-deploy.sh's version-bump precheck share.
# Versions compare numerically per dotted component, a missing component = 0 (1.3 == 1.3.0), and a non-numeric
# suffix after the dotted number must match exactly or the pair is "incomparable".
# Named mutations: compare components as strings -> VV-numeric goes red (1.10 vs 1.9 reads as -1);
# drop the "a < b" branch -> VV-lower goes red.

setup() {
  TMPDIR_T="$(mktemp -d)"
  # shellcheck disable=SC1091  # kit lib under test, resolved at runtime
  . "$BATS_TEST_DIRNAME/../build-n4-module-kit/toolbelt/lib/vendor-version.sh"
}
teardown() { rm -rf "$TMPDIR_T"; }

# _mkjar <jar> <module.xml text> — a jar holding only META-INF/module.xml.
_mkjar() {
  mkdir -p "$TMPDIR_T/j/META-INF"
  printf '%s\n' "$2" > "$TMPDIR_T/j/META-INF/module.xml"
  (cd "$TMPDIR_T/j" && zip -q -r "$1" .)
  rm -rf "$TMPDIR_T/j"
}

@test "VV-trailing-zero: a missing component is 0 (1.3 vs 1.3.0, 1.03 vs 1.3 -> 0)" {
  run vv_cmp 1.3 1.3.0;  [ "$status" -eq 0 ]; [ "$output" = "0" ]
  run vv_cmp 1.3.0 1.3;  [ "$output" = "0" ]
  run vv_cmp 1.03 1.3;   [ "$output" = "0" ]
}

@test "VV-lower: a lower first version prints -1 (1.3 vs 1.3.1, 1.3 vs 1.4, 2.10.2 vs 2.10.10)" {
  run vv_cmp 1.3 1.3.1;       [ "$status" -eq 0 ]; [ "$output" = "-1" ]
  run vv_cmp 1.3 1.4;         [ "$output" = "-1" ]
  run vv_cmp 2.10.2 2.10.10;  [ "$output" = "-1" ]
}

@test "VV-numeric: components compare as numbers, not text (1.10 vs 1.9 -> 1, 2.10.10 vs 2.10.2 -> 1)" {
  run vv_cmp 1.10 1.9;        [ "$status" -eq 0 ]; [ "$output" = "1" ]
  run vv_cmp 2.10.10 2.10.2;  [ "$output" = "1" ]
  run vv_cmp 1.3.1 1.3;       [ "$output" = "1" ]
}

@test "VV-suffix-same: an identical non-numeric suffix compares on the number (1.3-rc vs 1.3.0-rc -> 0, 1.4-rc vs 1.3-rc -> 1)" {
  run vv_cmp 1.3-rc 1.3.0-rc;  [ "$output" = "0" ]
  run vv_cmp 1.4-rc 1.3-rc;    [ "$output" = "1" ]
  run vv_cmp 1.3-rc 1.4-rc;    [ "$output" = "-1" ]
}

@test "VV-suffix-differ: a differing suffix, an empty side or a non-numeric version is incomparable" {
  run vv_cmp 1.3-rc 1.3;    [ "$status" -eq 0 ]; [ "$output" = "incomparable" ]
  run vv_cmp 1.4 1.3-beta;  [ "$output" = "incomparable" ]
  run vv_cmp "" 1.3;        [ "$output" = "incomparable" ]
  run vv_cmp 1.3 "";        [ "$output" = "incomparable" ]
  run vv_cmp abc abc;       [ "$output" = "incomparable" ]
}

@test "VV-module-version: reads the <module> root tag's own vendorVersion (two- or three-part), never a dependency floor" {
  _mkjar "$TMPDIR_T/a.jar" '<module name="X" vendor="V" vendorVersion="1.3"><dependencies><dependency name="baja" vendorVersion="4.13"/></dependencies></module>'
  run vv_module_version "$TMPDIR_T/a.jar"
  [ "$status" -eq 0 ]; [ "$output" = "1.3" ]
  _mkjar "$TMPDIR_T/b.jar" '<module name="X" vendorVersion="2.10.2-rc"></module>'
  run vv_module_version "$TMPDIR_T/b.jar"
  [ "$output" = "2.10.2-rc" ]
}

@test "VV-module-version-missing: no module version (dependency floor only, or no jar) prints nothing, status 0" {
  _mkjar "$TMPDIR_T/c.jar" '<module name="X"><dependencies><dependency name="baja" vendorVersion="4.13"/></dependencies></module>'
  run vv_module_version "$TMPDIR_T/c.jar"
  [ "$status" -eq 0 ]; [ -z "$output" ]
  run vv_module_version "$TMPDIR_T/no-such.jar"
  [ "$status" -eq 0 ]; [ -z "$output" ]
}

@test "VV-source-version: reads the first uncommented defaultModuleVersion(\"X\") of a build.gradle.kts on stdin" {
  run vv_source_version <<'KTS'
niagaraModule {
  // defaultModuleVersion sets the "vendorVersion" attribute on all modules
  // defaultModuleVersion("9.9.9")
  defaultModuleVersion( "2.10.0" )
  defaultModuleVersion("1.0.0")
}
KTS
  [ "$status" -eq 0 ]; [ "$output" = "2.10.0" ]
}

@test "VV-source-version-missing: no defaultModuleVersion call prints nothing, status 0" {
  run vv_source_version <<<'plugins { id("x") }'
  [ "$status" -eq 0 ]; [ -z "$output" ]
}
