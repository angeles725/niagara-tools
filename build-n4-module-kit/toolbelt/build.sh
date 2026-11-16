#!/usr/bin/env bash
# build.sh — full automatic chain: preflight → gradle (Java 8 + clean + slotomatic + jar) → verify gate → report-module.
# A `gradle :jar` with the default JDK is NOT a build (wrong bytecode major, slotomatic skipped).
# Deploying to a station is ng-deploy.sh's job (backup -> build -> copy -> type-count verify).
#
# Usage: build.sh [--profiles rt,ux,wb] [--target-version X.Y] [--plugin-version V] [--ui-profile P] [--legacy]
#                 [--gradle-prop k=v]... [--werror] [--no-preflight] [--no-report] [--no-drift-check]
#                 [--install-into-home]
#                 <module-root> <MOD> [niagara_home]
#   <module-root>   the dir holding ./gradlew and <MOD>/<MOD>-{rt,ux,wb}/ (the GROUP dir). When neither it nor
#                   an ancestor holds ./gradlew (the REPO root was passed), the error lists every gradle root
#                   found up to 3 levels BELOW it — descend into one. [ev: retro comppan-fase2-amps-alarms Δ2]
#   niagara_home    arg 3, else $niagara_home. It must be a mirror (mirror-niagara-home.sh, holds .niagara-mirror):
#                   gradle's :jar step copies the jar into <niagara_home>/modules, so a marker-less home exits 10.
#   --install-into-home  opt in to building into a real (marker-less) niagara_home; its modules/ jar is replaced.
#                   [ev: retro r10-buildsh-mirror-guard]
#   --plugin-version / $NIAGARA_PLUGIN_VERSION   forwarded as -PniagaraPluginVersion (each install ships ONE
#                   niagara-module plugin: 4.13.2 -> 7.3.40, 4.14 -> 7.6.17, 4.15.3 -> 7.6.22); also forwarded to
#                   preflight.sh --plugin-version so its plugin-pin check tests the version gradle will actually
#                   use, not a drifted settings.gradle.kts default [ev: retro continuous-fan-post-defrost-delay Δ1]
#   --ui-profile P  hmi|lan|both|unknown — the module's BUILD-STATE ui_profile, forwarded to report-module.sh
#                   --profile (selects rc-scan's browser floor); --legacy forwards report-module.sh --legacy
#   --gradle-prop k=v  repeatable; forwarded to gradlew as -Pk=v (e.g. niagaraSigningAlias=..., niagaraSigningProfile=...
#                   for the signing block); a missing k=v or an empty key exits 2 before gradle. The env route
#                   ORG_GRADLE_PROJECT_<k>=<v> reaches providers.gradleProperty(...) too. [ev: retro r8-buildsh-316-308]
#   --werror        opt in: every JavaCompile runs with -Xlint:all -Werror (gradle --init-script lib/werror.init.gradle),
#                   as vendors that build with -Werror do; a warning fails the build (exit 30, remedy printed: fix it,
#                   or a scoped @SuppressWarnings). Off by default. [ev: retro course-coverage Δ22]
#   --no-preflight  skip environment preflight (useful for inner rebuild loops when env is known-good)
#   --no-report     skip report-module punch-list at the end (useful for quick inner rebuild loops)
#   --first-deploy  forwarded to report-module.sh: a never-deployed module has no baseline for schema-risk
#                   (otherwise a missing <artifact>/.deploy-baseline is INCOMPLETE, exit 53) [ev: audit-2026-10-03 A2]
#   --no-drift-check  skip the deployed-baseline drift gate below (Δ2) — fix the version bump, don't skip, unless
#                   you have a real reason (e.g. this niagara_home was never the deploy target)
#   $JAVA8          JDK 8 path (default /usr/lib/jvm/java-8-openjdk-amd64)
# Profiles: by default every <MOD>-<p> dir that has a gradle file AND sources under src/ is built; a scaffold
#   (gradle file, no sources) is reported "skipped". --profiles replaces auto-detection entirely.
# After gradle, verify-module.sh (same dir) runs on every produced jar with --src <module-root>/<MOD>.
# Expected signer: the gate also gets --expected-signer <alias>, so a jar signed with another key (the signing
#   plugin's default development key) FAILs its `signed` row (exit 50). The alias is the niagaraSigningAlias gradle
#   property (the last --gradle-prop, else $ORG_GRADLE_PROJECT_niagaraSigningAlias, else
#   $GRADLE_USER_HOME/gradle.properties, else the gradle root's gradle.properties); else the one signer of the deployed
#   jars the drift gate snapshots (none under --no-drift-check, or when they disagree; never the plugin's default
#   development signer NIAGARA4, which a deployed jar may carry: the WARN then says so). With neither, the gate accepts
#   any signature and build.sh prints one "WARN — no niagaraSigningAlias is configured" line after the gate's rows: the
#   jar may carry the development signer, which a station's Software Manager shows as self-signed. The exit code does
#   not change. A configured alias is passed as given, whatever its value.
#   Retry-safe: when the gate FAILs, the pre-build modules/ jar the drift gate backed up is restored (one "Restored
#   the previously-deployed" line, exit 50 as before), so a bare re-run still expects the deployed signer and FAILs
#   again. Under --no-drift-check there is no backup: nothing is restored and the deployed-jar signer is unavailable.
#   [ev: external session retro, delta B5]
# Build-location precheck (Δ7): WARNs (non-fatal) when <module-root>'s gradle root or niagara_home sits on a
#   WSL 9p/drvfs mount (/mnt/<drive>) — Gradle's per-file work is far slower there than on ext4.
# Deployed-baseline drift gate (Δ2): before gradle's :jar task overwrites <niagara_home>/modules/<jar>, this
#   snapshots what is CURRENTLY installed there; after the build, if the new jar's shipped bytes differ from
#   that snapshot but the module's own vendorVersion did not change, the build FAILs (exit 51) — Software
#   Manager compares versions, not bytes, and would silently report "Up to Date" and skip installing the fix.
#   The version may be X.Y or X.Y.Z and compares numerically per component, missing = 0 (1.3 == 1.3.0), via
#   go/internal/vendorversion (shared with scripts/ng-deploy.sh). A LOWER version than the deployed jar WARNs
#   ("vendorVersion went down") and passes: a deliberate rollback is legitimate.
#   A jar the gate cannot read (not a zip, no entry, a duplicate entry name, an entry failing its CRC) or a changed jar whose own vendorVersion cannot be
#   read FAILs closed with exit 52 ("cannot verify"), never a silent pass.
#   Retry-safe: gradle's :jar step already overwrote the deployed jar by the time this FAILs, so the gate
#   backs up the pre-build jar and restores it into modules/ on FAIL — a bare re-run then still sees the
#   OLD baseline and catches the same drift again, instead of silently "Up to Date"-passing on the retry.
# Post-jar copy lock (exit 32): the plugin's last step copies the jar into <niagara_home>/modules; a Windows
#   process (Workbench, station) holding the old jar open on /mnt/c makes it fail with FileAlreadyExistsException.
#   build.sh names the build/libs jar(s) rebuilt THIS run as the artifact of record (a stale one is flagged
#   "not rebuilt"). [ev: retro panccadia-restart-seq-comp-lockout-hours Δ3]
# Independent gradle groups (different gradle roots) may be built CONCURRENTLY: one build.sh per group in the
#   background, then wait — never two runs on the SAME gradle root (BUILD-LOOP.md §4).
#   [ev: retro change-tier-time-budgets Δ5]
# Mirror user.home isolation: when niagara_home holds the .niagara-mirror marker (mirror-niagara-home.sh), gradle
#   runs with JAVA_TOOL_OPTIONS += -Duser.home=<mirror>/.mirror-user-home, GRADLE_USER_HOME kept (default
#   ~/.gradle), -Dmaven.repo.local=<real ~/.m2/repository> (unless already set) and --no-daemon, so the signing
#   plugin cannot read the real ~/.tridium keystore. A mirror path, or a $HOME used for that maven.repo.local, with
#   whitespace is refused (exit 33); a signing profile in the mirror user home (always plugin-created, so a throwaway
#   dev key) is a WARN on every build. [ev: retro r8-mirror-userhome Δ1, retro r8-buildsh-316-308]
# Slot-o-Matic skip (exit 34): Slot-o-Matic logs a file it could not process ("Could not process <file>.java: …")
#   at WARNING level and gradle still ends BUILD SUCCESSFUL; the jar then carries stale generated code. A passing
#   run whose log names a skipped file (go/internal/slotomaticskip) FAILs, the pre-build modules/ jar is restored
#   like a drift FAIL, and the gate is not run (METHODOLOGY K15). [ev: retro r10-slotomatic-skip-fail-open Δ1]
# Exit: 0 chain passed · 2 usage · 10 environment (preflight FAIL, no JDK 8, not a niagara_home, a real niagara_home
#   without --install-into-home, no profile) ·
#   30 gradle failed · 31 :clean locked · 32 post-jar modules/ copy locked · 33 mirror path or $HOME has whitespace ·
#   34 Slot-o-Matic skipped a file in a passing gradle run · 50 gate or report-module FAIL ·
#   51 deployed-baseline drift (Δ2) · 52 drift gate cannot verify · 53 report-module INCOMPLETE (a floor check SKIPped)
set -u

# Implementation: the Go module at <repo>/go (go/cmd/build-module, go/internal/buildmodule). This script is only
# the launcher: go/go-tool.sh `go_tool_bin` builds go/bin/build-module when the binary is missing or older than a
# Go source, then this script execs it. Needs `go` on PATH to build; no Go toolchain or a failed build is exit 3.
# The header above is the usage text: the binary reads it from this file when it prints it (go/internal/buildmodule/usage.go).
# readlink -f: a toolbelt copy that links this file (tests/build-sh.bats) still finds ../../go, while the gate,
# preflight, report-module and lib/ are taken from the directory of the INVOKED path, so its stubs run.
here="$(cd "$(dirname "$(readlink -f "${BASH_SOURCE[0]}")")" && pwd)"
GO_DIR="$(cd "$here/../.." && pwd)/go"
[ -f "$GO_DIR/go-tool.sh" ] || { printf 'build.sh: no Go module at %s (go-tool.sh missing)\n' "$GO_DIR" >&2; exit 3; }
# shellcheck disable=SC1091  # go/ helper, resolved at runtime via GO_DIR
. "$GO_DIR/go-tool.sh"
BIN="$(go_tool_bin "$GO_DIR" build-module)" || exit 3
BUILD_MODULE_TOOLBELT="$(CDPATH='' cd -- "$(dirname "$0")" >/dev/null && pwd)"
export BUILD_MODULE_TOOLBELT
exec "$BIN" "$@"
