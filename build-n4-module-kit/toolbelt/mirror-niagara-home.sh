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
#        <mirror_dir> is judged where it leads (every symlink followed where components exist,
#        also after a ".." that climbs back into them; the rest by its text, as realpath -m). That
#        judged path is the one G-B checks, the wipe removes and the mirror is written to, never the
#        spelling; nothing is created until every check has passed, so a refusal leaves nothing
#        behind. One that cannot be resolved (a link loop, an unsearchable directory) is refused.
#   G-B  an existing non-empty <mirror_dir> is wiped ONLY if it holds the .niagara-mirror marker
#        this script writes (source path + date); anything else is refused. A symlinked
#        <mirror_dir> is refused whatever its spelling (link, link/, link/., missing/../link). -> exit 20
# An argument <source_niagara_home> or <mirror_dir> that starts with '-' ('-' alone too) is a usage
# error (exit 1): spell such a path ./-name.
# Exit: 0 mirror ready · 1 usage / not a niagara_home / a step failed · 3 no Go toolchain · 20 guard refused
#
# Truth fixes of the Go port (each was a "mirror ready" or a bare shell error in the bash tool):
#   a relative <source_niagara_home> is refused (exit 1) before anything is wiped: the links hold the source
#     as given, so every one of them dangled from inside the mirror;
#   a source, or its modules/, that cannot be listed is exit 1 (the bash globs stayed literal: 0 jars, "ready");
#   a <mirror_dir> that exists and is not a directory gets the G-B refusal (exit 20) the bash tool never
#     reached (its `cd` failed first, exit 1).
# The source is listed before the old mirror is wiped, so a refusal leaves the old mirror whole.
#
# Implementation: the Go module at <repo>/go (go/cmd/mirror-niagara-home, go/internal/mirrorhome). This script
# is only the launcher: go/go-tool.sh `go_tool_bin` builds go/bin/mirror-niagara-home when the binary is missing
# or older than a Go source, then this script execs it. No Go toolchain or a failed build is exit 3.
set -u

# readlink -f: resolve a symlinked copy of this file to the real kit, so ../../go exists.
here="$(cd "$(dirname "$(readlink -f "${BASH_SOURCE[0]}")")" && pwd)"
GO_DIR="$(cd "$here/../.." && pwd)/go"
[ -f "$GO_DIR/go-tool.sh" ] || { printf 'mirror-niagara-home: no Go module at %s (go-tool.sh missing)\n' "$GO_DIR" >&2; exit 3; }
# shellcheck disable=SC1091  # go/ helper, resolved at runtime via GO_DIR
. "$GO_DIR/go-tool.sh"
BIN="$(go_tool_bin "$GO_DIR" mirror-niagara-home)" || exit 3
exec "$BIN" "$@"
