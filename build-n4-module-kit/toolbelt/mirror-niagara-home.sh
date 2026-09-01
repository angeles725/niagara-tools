#!/usr/bin/env bash
# mirror-niagara-home.sh — build a READ-ONLY mirror of a Niagara install with a WRITABLE modules/,
# so a module compiles against a SPECIFIC niagara_home WITHOUT touching a running station.
#
# WHY: a running station (station.exe / niagarad.exe) LOCKS <niagara_home>/modules/<module>.jar.
# The niagara-module gradle plugin's clean/jar tries to delete+recopy that jar -> build fails
# ("Unable to delete file ... modules/<module>.jar"). A mirror whose modules/ is a real writable dir
# (symlinks to every real module jar EXCEPT the one being built) lets the plugin copy freely while the
# live install is never written. The manifest's baja dependency version comes from the mirrored
# install, so this also pins cross-version builds (mirror 4.14 -> jar stamps baja 4.14).
#
# Usage:  mirror-niagara-home.sh <source_niagara_home> <mirror_dir> [exclude-jar ...]
# Example (ColdRoomPan against Honeywell 4.14, station running):
#   mirror-niagara-home.sh /mnt/c/Honeywell/OptimizerSupervisor-N4.14.0.162 "$HOME/niagara-mirror-hon414" ColdRoomPan-rt.jar
#   ./gradlew :ColdRoomPan-rt:{clean,slotomatic,jar} -Pniagara_home="$HOME/niagara-mirror-hon414" \
#       -Porg.gradle.java.installations.paths=/usr/lib/jvm/java-8-openjdk-amd64 -PniagaraPluginVersion=7.6.17
#   (absolute path: `-Pniagara_home=~/x` is NOT tilde-expanded by bash, gradle would get a literal "~")
#
# SAFETY (both guards run before any rm; there is no --force):
#   G-A  <mirror_dir> must not be the source install, live inside it, or contain it.        -> exit 20
#   G-B  an existing non-empty <mirror_dir> is wiped ONLY if it holds the .niagara-mirror marker
#        this script writes (source path + date); anything else is refused.                  -> exit 20
# Exit: 0 mirror ready · 1 usage / not a niagara_home · 20 guard refused
set -euo pipefail
MARK=.niagara-mirror
usage() { echo "usage: mirror-niagara-home.sh <source_niagara_home> <mirror_dir> [exclude-jar ...]" >&2; exit 1; }
[ $# -ge 2 ] || usage
src="$1"; mir="$2"; shift 2
[ -d "$src/etc/m2/repository" ] || { echo "not a niagara_home (no etc/m2/repository): $src" >&2; exit 1; }
src_abs="$(cd "$src" && pwd -P)"
mir_parent="$(dirname "$mir")"
if [ -e "$mir" ]; then mir_abs="$(cd "$mir" && pwd -P)"
else mkdir -p "$mir_parent"; mir_abs="$(cd "$mir_parent" && pwd -P)/$(basename "$mir")"; fi
# G-A: source protection (equal, inside, or containing)
case "$mir_abs/" in "$src_abs/"*) echo "refusing: mirror dir is the source install or inside it: $mir" >&2; exit 20 ;; esac
case "$src_abs/" in "$mir_abs/"*) echo "refusing: mirror dir contains the source install: $mir" >&2; exit 20 ;; esac
# G-B: wipe protection
if [ -e "$mir" ]; then
  [ -d "$mir" ] || { echo "refusing: $mir exists and is not a directory" >&2; exit 20; }
  if [ -n "$(ls -A "$mir")" ] && [ ! -f "$mir/$MARK" ]; then
    echo "refusing to wipe $mir: non-empty and not a mirror made by this script (no $MARK)" >&2; exit 20
  fi
  rm -rf "$mir"
fi
mkdir -p "$mir/modules"
printf 'source=%s\ncreated=%s\n' "$src_abs" "$(date +%Y-%m-%dT%H:%M:%S)" > "$mir/$MARK"
for x in "$src"/*; do b=$(basename "$x"); [ "$b" = modules ] && continue; ln -s "$x" "$mir/$b"; done
n=0
for j in "$src"/modules/*.jar; do
  [ -e "$j" ] || continue          # empty modules/ -> the glob stays literal
  b=$(basename "$j"); skip=0
  for e in "$@"; do [ "$b" = "$e" ] && skip=1; done
  [ "$skip" = 1 ] && continue
  ln -s "$j" "$mir/modules/$b"; n=$((n+1))
done
echo "mirror ready: $mir  ($n jars linked, modules/ writable, excluded: ${*:-none})"
