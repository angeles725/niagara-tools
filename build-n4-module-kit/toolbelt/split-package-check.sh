#!/usr/bin/env bash
# split-package-check.sh — project-level check: one Java package declared in two N4 modules of a project.
#
# N4 is not OSGi. ModuleClassLoader.nfind() asks the parent loader, then the module's own jar, then its
# dependencies — a sticky last-hit cache, the deps that declare the package (in module.xml dependency order),
# then every other dep — and the FIRST hit wins. A same-named class in the same package of a second module is
# silently shadowed: no error, and behavior that changes with the dependency order. Rule: one package belongs to
# exactly one module; shared code goes in its own module listed as a dependency. See
# types/issues-and-gotchas.md §D4c. [ev: retro module-hardening-failure-modes-deltas Δ12] [ev: corpus B1125]
# [ev: code com/tridium/sys/module/ModuleClassLoader.java:186-294] [ev: code com/tridium/sys/module/NModule.java:333-335]
#
# Cross-module by nature, so it is not a per-module lint and report-module.sh (one module) does not run it. Its
# natural home is the client repository's CI (issue #142); until that is wired, run it by hand on the project
# root before a release. Advisory by default (WARN, exit 0); --strict makes a split package exit 1 for a CI gate.
# <project-root> is ONE checkout: a directory that holds several checkouts or worktrees of the same project
# reports every copied package as a split.
#
# Structural rules:
#   Module  — a directory holding `module-include.xml`, `build.gradle.kts` or `build.gradle` AND a `src/` directory
#             (each -rt / -ux / -wb / -se artifact is its own N4 module). Dot-dirs, `build/`, `node_modules/`,
#             `src/` and `srcTest/` are pruned from the module search (sources are listed per module below).
#   Package — the `package <name>;` declaration of every *.java under the module's `src/` (srcTest/ is not a
#             module's runtime classpath). A file with no declaration is the default package, `(default)`.
#   Split   — the same package in two or more modules: one WARN row naming every module and one file in each.
#   Fail closed — a directory find cannot enter or a source file awk cannot read is exit 3, never a clean pass.
#
# Usage:  split-package-check.sh [--strict] <project-root>
#   Row:  WARN  split-package-check  <package>  <detail>
#   Exit: 0  no split package (or WARN without --strict) · 1  a split package under --strict · 3  usage/env, an unreadable file or directory, or no module found
# VCS-free by design (kit-links L2).
# Mutation: SPC2 -- counting files instead of distinct modules WARNs a package spread over two files of ONE module
# Mutation: SPC4 -- scanning srcTest/ as well WARNs a test fixture that mirrors a runtime package
# Mutation: SPC-finderr -- ignoring the per-module listing status skips the unreadable directory and reports clean
# Mutation: SPC-findtree -- ignoring the module-search status skips a module under an unreadable directory
set -u
# shellcheck disable=SC1091  # sibling lib, resolved at runtime via BASH_SOURCE
. "$(cd "${BASH_SOURCE[0]%/*}" && pwd)/lib/scan-files.sh"
LC_ALL=C
export LC_ALL

USAGE='usage: split-package-check.sh [--strict] <project-root>'
STRICT=0
while [ $# -gt 0 ]; do
  case "$1" in
    --strict) STRICT=1; shift ;;
    --) shift; break ;;
    -*) printf 'split-package-check: unknown flag: %s\n%s\n' "$1" "$USAGE" >&2; exit 3 ;;
    *) break ;;
  esac
done
if [ $# -ne 1 ]; then
  printf '%s\n' "$USAGE" >&2
  exit 3
fi
ROOT="${1%/}"
if [ ! -d "$ROOT" ]; then
  printf 'split-package-check: not a directory: %s\n' "$ROOT" >&2
  exit 3
fi

_TMP=$(mktemp -d)
trap 'rm -rf "$_TMP"' EXIT
had_err=0

# 1. Module directories: a build marker next to a src/ directory.
if ! find "$ROOT" -mindepth 1 \( -type d \( -name '.*' -o -name build -o -name node_modules -o -name src -o -name srcTest \) -prune \) -o \
     -type f \( -name module-include.xml -o -name build.gradle.kts -o -name build.gradle \) -print \
     > "$_TMP/markers" 2> "$_TMP/find.err"; then
  printf 'split-package-check: cannot list every file under %s: %s\n' "$ROOT" "$(head -n 1 "$_TMP/find.err")" >&2
  had_err=1
fi
: > "$_TMP/modules"
while IFS= read -r m; do
  d="${m%/*}"
  [ -d "$d/src" ] && printf '%s\n' "$d" >> "$_TMP/modules"
done < "$_TMP/markers"
LC_ALL=C sort -u "$_TMP/modules" -o "$_TMP/modules"
if [ ! -s "$_TMP/modules" ]; then
  printf 'split-package-check: no module (a module-include.xml / build.gradle[.kts] next to src/) under %s\n' "$ROOT" >&2
  exit 3
fi

# 2. Package declarations per module: <package> TAB <module> TAB <file>
: > "$_TMP/pkgs"
while IFS= read -r d; do
  if ! scan_files "$_TMP/files" "$_TMP/find.err" "$d/src" -name '*.java'; then
    printf 'split-package-check: cannot list every file under %s: %s\n' "$d/src" "$(head -n 1 "$_TMP/find.err")" >&2
    had_err=1
  fi
  mod="${d#"$ROOT"/}"
  while IFS= read -r f; do
    if ! p=$(awk '
        { sub(/\/\/.*/, "") }
        match($0, /^[[:space:]]*package[[:space:]]+[A-Za-z_][A-Za-z0-9_.]*[[:space:]]*;/) {
          s = substr($0, RSTART, RLENGTH); sub(/^[[:space:]]*package[[:space:]]+/, "", s); sub(/[[:space:]]*;$/, "", s)
          print s; exit
        }' "$f" 2>"$_TMP/awk.err"); then
      printf 'split-package-check: cannot scan %s: %s\n' "$f" "$(head -n 1 "$_TMP/awk.err")" >&2
      had_err=1
      continue
    fi
    printf '%s\t%s\t%s\n' "${p:-(default)}" "$mod" "${f#"$ROOT"/}" >> "$_TMP/pkgs"
  done < "$_TMP/files"
done < "$_TMP/modules"

# 3. A package in more than one module (distinct modules, first file of each named).
LC_ALL=C sort -t "$(printf '\t')" -k1,1 -k2,2 -k3,3 "$_TMP/pkgs" | awk -F '\t' '
  function flush() {
    if (nm > 1) printf "WARN  split-package-check  %s  declared in %d modules: %s -- first-dependency-wins shadows one copy; one package belongs to one module (types/issues-and-gotchas.md §D4c)\n", cur, nm, list
  }
  $1 != cur { flush(); cur = $1; nm = 0; list = ""; last = "" }
  $2 != last { nm++; list = list (list == "" ? "" : ", ") $2 " (" $3 ")"; last = $2 }
  END { flush() }
' > "$_TMP/rows"

had_warn=0
if [ -s "$_TMP/rows" ]; then
  cat "$_TMP/rows"
  had_warn=1
fi
[ "$had_err" -eq 0 ] || exit 3
if [ "$had_warn" -eq 1 ] && [ "$STRICT" -eq 1 ]; then
  exit 1
fi
exit 0
