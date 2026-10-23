#!/usr/bin/env bats
# PR: every module-permissions.xml the kit ships or scaffolds has the root element <permissions>.
# The NRE reads only manifest.elem("permissions") (NModule.java:455, organized/baja/baja/decompiled/
# com/tridium/sys/module/NModule.java) and the build plugin copies the file's root into META-INF/module.xml
# unchanged, so a <modulePermissions/> root makes every permission added later silently ignored.
# The template, the course files and the Workbench wizard all use <permissions>.

setup() {
  REPO="$(cd "$BATS_TEST_DIRNAME/.." && pwd)"
  KIT="$REPO/build-n4-module-kit"
}

# root_of FILE — the first element name of an XML file (after the prolog, comments and blank lines).
root_of() {
  python3 -c 'import sys, xml.etree.ElementTree as ET; print(ET.parse(sys.argv[1]).getroot().tag)' "$1"
}

@test "PR1: every shipped module-permissions.xml (fixtures + templates) has a <permissions> root" {
  # Mutation: PR1-root -- a fixture root renamed back to <modulePermissions/> fails this pin
  cd "$REPO"
  mapfile -t files < <(git ls-files -- 'build-n4-module-kit/**/module-permissions.xml')
  [ "${#files[@]}" -ge 4 ]
  bad=""
  for f in "${files[@]}"; do
    r="$(root_of "$f")" || { bad="$bad $f(unparsable)"; continue; }
    [ "$r" = permissions ] || bad="$bad $f(<$r>)"
  done
  [ -z "$bad" ] || { echo "module-permissions.xml without a <permissions> root:$bad"; false; }
}

@test "PR2: a scaffolded module's permissions files have a <permissions> root (logic and dashboard)" {
  for type in logic dashboard; do
    out="$BATS_TEST_TMPDIR/$type"
    run bash "$KIT/toolbelt/scaffold-module.sh" DemoPerm "$out" --type "$type"
    [ "$status" -eq 0 ] || { echo "$output"; false; }
    mapfile -t files < <(find "$out" -name module-permissions.xml)
    [ "${#files[@]}" -ge 1 ]
    for f in "${files[@]}"; do
      [ "$(root_of "$f")" = permissions ] || { echo "$f has root <$(root_of "$f")>"; false; }
    done
  done
}
