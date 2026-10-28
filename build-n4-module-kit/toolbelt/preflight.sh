#!/usr/bin/env bash
# preflight.sh — BUILD-LOOP §0.b environment preflight for Niagara N4 module builds.
#
# Automates the four pre-build checks so a broken environment is caught before
# gradle fails with a cryptic error. A FAIL row identifies the problem; the script
# continues checking the rest so all issues are visible in one run.
#
# Usage: preflight.sh [--jvm-dir <d>] [--plugin-version <v>] [--sample] [--toolchains] <niagara_home> <gradle-root>
#   --jvm-dir <d>   scan this directory for a JDK 8 release (default /usr/lib/jvm, read-only)
#                   JAVA_HOME is also checked (fallback) — never $HOME
#   --plugin-version <v>  the niagara plugin version the build will actually use (build.sh forwards its
#                   --plugin-version / $NIAGARA_PLUGIN_VERSION here); plugin-pin checks <v> instead of
#                   re-deriving the settings.gradle.kts default, which may have drifted from niagara_home.
#                   Falls back to $NIAGARA_PLUGIN_VERSION. [ev: retro continuous-fan-post-defrost-delay Δ1]
#   --sample        the gradle root is a project you did NOT generate (course zip, vendor sample): also run
#                   the sample-* retarget checks below. Opt-in: client trees keep their own gradle.properties.
#   --toolchains    the raw POSIX gradlew route (not build.sh, which pins -P...paths=$JAVA8 itself, checked by
#                   jdk8): also run the toolchains row below. Opt-in: it starts gradle. The Windows gradlew.bat
#                   route is not checked (a SKIP row says so). [ev: retro course-coverage Δ25]
#   <niagara_home>  Niagara 4 installation root (must not be a Windows-style path)
#   <gradle-root>   gradle project root containing settings.gradle.kts
#
# Checks (order: win-path, jdk8, plugin-pin, jar-lock):
#   win-path    niagara_home or gradle-root contains C: or backslash
#               -> FAIL + "use /mnt/c/... in WSL" remedy
#   jdk8        JDK 8 found under --jvm-dir or JAVA_HOME (never $HOME)
#               -> PASS|FAIL row; detail names the found path or search dir
#   plugin-pin  plugin version present in <niagara_home>/etc/m2: --plugin-version / $NIAGARA_PLUGIN_VERSION
#               when given, else the first "x.y.z" quoted string in settings.gradle.kts -> PASS|FAIL
#   jar-lock    ONE filtered `lsof -Fn` pass over <niagara_home>/modules/*.jar
#               -> PASS|WARN(locked)|SKIP(lsof absent or returning an empty listing, or niagara_home is
#               9p/drvfs) — never a false PASS
#   sample-*    [--sample] gradle.properties niagara_home exists (C:\\… read as /mnt/c/…) and carries the
#               <niagara_home> version (sample-home; WARN when either path has no X.Y.Z to compare);
#               niagara_user_home is set (WARN when missing) and has no <…> placeholder (sample-user);
#               the ROOT build.gradle.kts code outside comments and strings has allowDefaultProfile.set(false)
#               or `allowDefaultProfile = false` plus niagaraSigning { aliases…; signingProfileFile… }
#               (sample-sign); no shipped build/ directory (sample-build, WARN: delete it).
#               [ev: retro n4-course-signed-jars-gate-fix Δ6] [ev: retro n4-course-sample-check-precision Δ1]
#   toolchains  [--toolchains] a 4.15 build needs a full JDK 8 (corpus B1235 §1235.2): gradle.properties
#               org.gradle.java.installations.paths pinned to one (bin/javac + Java 8) -> PASS without gradle;
#               else `gradlew -q -p <gradle-root> javaToolchains -Porg.gradle.java.installations.auto-detect=true`
#               -> PASS on a listed Language Version 8 with Is JDK true, FAIL when none (a JRE 8 or a JDK 17
#               alone); SKIP when gradlew is not executable or the listing names no toolchain (never a false PASS).
#               WSL/POSIX route only: a root with a gradlew.bat gets a second SKIP row (Windows route not checked)
#
#   Truth fixes of the Go port: plugin-pin matches the settings version as an exact etc/m2 directory name
#   (the bash `-path "*V*"` PASSed on any path holding V, niagara_home included), and version-drift reads each
#   tree in sorted order (find's readdir order made the rows host-dependent). [ev: retro r10-preflight-go Δ3]
#   [ev: retro r10-preflight-go Δ4]
#
# Implementation: the Go module at <repo>/go (go/internal/preflight; lsof, java -version and df run through its
# injectable Host). This wrapper is the stable entry point: go/go-tool.sh `go_tool_bin` builds go/bin/preflight
# when the binary is missing or older than a Go source, then this script execs it. [ev: retro r10-preflight-go Δ1]
#
# Row format: PASS|FAIL|WARN|SKIP  <check>  <detail>
# Exit: 0 all PASS/WARN/SKIP · 1 any FAIL · 2 usage · 3 env (path not found, or no Go toolchain / a failed build)
# This script is VCS-free by design. version control is never invoked.
# kit-links.bats L2 enforces the no-version-control rule on all toolbelt scripts.
# Source-of-truth check (is the gradle root the declared checkout, not detached/stale?) needs version
# control, so it lives OUTSIDE the toolbelt: run scripts/check-client-source.sh next to this preflight.
# [ev: retro client-source-of-truth Δ1]
set -u

# readlink -f: resolve a symlinked copy of this file (report-module test toolbelts) to the real kit, so ../../go exists.
here="$(cd "$(dirname "$(readlink -f "${BASH_SOURCE[0]}")")" && pwd)"
GO_DIR="$(cd "$here/../.." && pwd)/go"
[ -f "$GO_DIR/go-tool.sh" ] || { printf 'preflight: no Go module at %s (go-tool.sh missing)\n' "$GO_DIR" >&2; exit 3; }
# shellcheck disable=SC1091  # go/ helper, resolved at runtime via GO_DIR
. "$GO_DIR/go-tool.sh"
BIN="$(go_tool_bin "$GO_DIR" preflight)" || exit 3
exec "$BIN" "$@"
