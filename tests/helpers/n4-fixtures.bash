#!/usr/bin/env bash
# n4-fixtures.bash — shared bats helpers that GENERATE every Niagara-module fixture at test time.
# Zero committed binaries: a fake .class is 8 bytes of printf, a jar is `zip` over a temp dir.
# Loaded with `load helpers/n4-fixtures` from tests/*.bats (never from tests/ng-deploy.bats).

# make_class_file <path> <major>   — CAFEBABE + minor 0000 + major (offset 6-7, what `od -j6 -N2` reads)
make_class_file() {
  local path="$1" major="$2"
  mkdir -p "$(dirname "$path")"
  # octal escapes only: POSIX printf has no \xNN. The major byte is rendered to an escape first
  # ('\064' for 52) because a bare '\%o' prints a literal backslash followed by digits.
  local esc; esc=$(printf '\\%03o' "$major")
  printf '%b' "\\312\\376\\272\\276\\000\\000\\000$esc" > "$path"
}

# make_jar <jar> <dir>             — deflated (zip default), like a gradle artifact
make_jar()        { (cd "$2" && zip -q -r "$1" .); }
# make_stored_jar <jar> <dir>      — every entry STORED, like stored-repack.sh output
make_stored_jar() { (cd "$2" && zip -q -0 -X -D -r "$1" .); }

# add_manifest <dir>               — META-INF/MANIFEST.MF (a jar without one is not a jar)
add_manifest()  { mkdir -p "$1/META-INF"; printf 'Manifest-Version: 1.0\n' > "$1/META-INF/MANIFEST.MF"; }
# add_signature <dir>              — NIAGARA4.SF + NIAGARA4.RSA with dummy content
add_signature() {
  mkdir -p "$1/META-INF"
  printf 'Signature-Version: 1.0\nName: com/x/A.class\nSHA-256-Digest: AAAA\n' > "$1/META-INF/NIAGARA4.SF"
  printf 'RSA' > "$1/META-INF/NIAGARA4.RSA"
}

# make_module_xml <dir> <vendorVersion> <type-class>...  — META-INF/module.xml with a baja dependency
make_module_xml() {
  local dir="$1" ver="$2"; shift 2
  mkdir -p "$dir/META-INF"
  {
    printf '<module name="Foo" vendor="Angeles">\n<dependencies><dependency name="baja" vendor="Tridium" vendorVersion="%s"/></dependencies>\n<types>\n' "$ver"
    for c in "$@"; do printf '<type name="%s" class="%s"/>\n' "${c##*.}" "$c"; done
    printf '</types>\n</module>\n'
  } > "$dir/META-INF/module.xml"
}

# make_module_include <profile-dir> <type-class>...   — <profile>/module-include.xml (where gradle reads it)
make_module_include() {
  local dir="$1"; shift
  mkdir -p "$dir"
  { printf '<types>\n'; for c in "$@"; do printf '<type name="%s" class="%s"/>\n' "${c##*.}" "$c"; done; printf '</types>\n'; } > "$dir/module-include.xml"
}

# make_niagara_home <dir> <plugin-version>   — the etc/m2 discriminator build.sh/mirror check for
make_niagara_home() {
  mkdir -p "$1/etc/m2/repository/com/tridium/niagara-module/com.tridium.niagara-module.gradle.plugin/$2" "$1/bin" "$1/modules"
}

# make_fake_gradlew <module-root>  — records every invocation's args in $TMPDIR_T/gradlew.calls.log,
#                                    exits with ${FAKE_GRADLEW_EXIT:-0}
make_fake_gradlew() {
  # shellcheck disable=SC2016
  # why: the single-quoted $* and ${FAKE_GRADLEW_EXIT} must reach the generated script unexpanded
  printf '#!/usr/bin/env bash\nprintf "%%s\\n" "$*" >> "%s/gradlew.calls.log"\nexit "${FAKE_GRADLEW_EXIT:-0}"\n' "$TMPDIR_T" > "$1/gradlew"
  chmod +x "$1/gradlew"
}

# make_profile <module-root> <MOD> <p> [with-sources 1|0]  — <root>/<MOD>/<MOD>-<p> with a gradle file
make_profile() {
  local d="$1/$2/$2-$3"
  mkdir -p "$d/src"
  : > "$d/$2-$3.gradle.kts"
  if [ "${4:-1}" = 1 ]; then mkdir -p "$d/src/com/x"; : > "$d/src/com/x/B$2.java"; fi
}
