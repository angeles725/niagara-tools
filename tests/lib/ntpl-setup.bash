# tests/lib/ntpl-setup.bash — the setup every tests/lint-ntpl-manifest-deps*.bats suite shares: the lint path,
# a module ModA with one -rt part and its build file, an installation of empty stock jars, and the _ntpl helper.
# Load at file scope with:   load lib/ntpl-setup
# shellcheck shell=bash

setup() {
  KIT="$(cd "$BATS_TEST_DIRNAME/../build-n4-module-kit" && pwd)"
  # shellcheck disable=SC2034  # read by the suites that load this file
  LINT="$KIT/toolbelt/lint-ntpl-manifest-deps.sh"
  M="$BATS_TEST_TMPDIR/ModA"
  mkdir -p "$M/ModA-rt/src/templates"
  printf 'dependencies {\n  nre(":nre")\n  api(":baja")\n}\n' > "$M/ModA-rt/ModA-rt.gradle.kts"
  NH="$BATS_TEST_TMPDIR/niagara"
  mkdir -p "$NH/modules"
  local j
  for j in baja fox-rt jetty-rt hx-wb web-rt control-rt bajaui-wb; do : > "$NH/modules/$j.jar"; done
  unset NIAGARA_HOME   # the machine running the test may name a real installation
}

# _ntpl <out.ntpl> <manifest-root-attrs> <dependency>... — a template archive with both entries
_ntpl() {
  local out="$1" attrs="$2" d stage="$BATS_TEST_TMPDIR/stage.$RANDOM"
  shift 2
  mkdir -p "$stage" "$(dirname "$out")"
  printf '<bajaObjectGraph version="4.0"/>\n' > "$stage/template.bog"
  {
    printf '<template id="t1" version="1.0"%s>\n  <dependencies>\n' "$attrs"
    for d in "$@"; do printf '    <dependency name="%s" vendor="Acme" vendorVersion="1.0"/>\n' "$d"; done
    printf '  </dependencies>\n</template>\n'
  } > "$stage/template-manifest.xml"
  (cd "$stage" && zip -q -X "$out" template.bog template-manifest.xml)
}
