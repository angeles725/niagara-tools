#!/usr/bin/env bash
# build.sh — full automatic chain: preflight → gradle (Java 8 + clean + slotomatic + jar) → verify gate → report-module.
# A `gradle :jar` with the default JDK is NOT a build (wrong bytecode major, slotomatic skipped).
# Deploying to a station is ng-deploy.sh's job (backup -> build -> copy -> type-count verify).
#
# Usage: build.sh [--profiles rt,ux,wb] [--target-version X.Y] [--plugin-version V] [--ui-profile P] [--legacy]
#                 [--gradle-prop k=v]... [--werror] [--no-preflight] [--no-report] [--no-drift-check]
#                 [--install-into-home] [--first-deploy | --schema-baseline-ref <deployed-ref> | --target <name>]
#                 <module-root> <MOD> [niagara_home]
#        build.sh --restore-mirror <mirror>/.mirror-replaced/<MOD>-<profile>.jar.<sha12>
#   <module-root>   the dir holding ./gradlew and <MOD>/<MOD>-{rt,ux,wb}/ (the GROUP dir). When neither it nor
#                   an ancestor holds ./gradlew (the REPO root was passed), the error lists every gradle root
#                   found up to 3 levels BELOW it — descend into one. [ev: retro comppan-fase2-amps-alarms Δ2]
#                   (report-module.sh's <module-root> is one level down, <module-root>/<MOD>: the module directory.)
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
#   --no-preflight  skip environment preflight (useful for inner rebuild loops when env is known-good), the acorn
#                   preflight below included
#   --no-report     skip report-module punch-list at the end (useful for quick inner rebuild loops)
#   --first-deploy  forwarded to report-module.sh: a never-deployed module has no baseline for schema-risk
#                   (otherwise a missing <artifact>/.deploy-baseline is INCOMPLETE, exit 53; its schema-risk SKIP row
#                   names the archive command that creates it from the deployed ref) [ev: audit-2026-10-03 A2]
#   --schema-baseline-ref <deployed-ref>  that archive command, done for the run: before gradle, every artifact
#                   under <module-root>/<MOD> is extracted as it is at <deployed-ref> (a tag or commit) into a temporary
#                   directory and <artifact>/.deploy-baseline links to it, so report-module's schema-risk floor check runs;
#                   the link and the extraction are removed when the run ends, whatever the exit (a link left by a killed
#                   run dangles and reads as no baseline; the next run with the option removes such a link, INFO line; an operator link is kept and refused). One "==> schema-risk baseline: <artifact> from <ref>" line per
#                   artifact; an artifact absent at the ref gets an INFO line and no baseline (its row stays SKIP). An
#                   artifact is a directory with module-include.xml, or a <MOD>-rt/-ux/-wb/-se/-doc part with its build file
#                   (report-module's set): a docs/ folder beside the parts gets an INFO line, no baseline and no report rows,
#                   never a schema-risk ERROR (exit 10) or INCOMPLETE (exit 53). [ev: retro client-2.3.0 K4] Exit 2
#                   before gradle when <deployed-ref> is not a commit of the module's repository, <MOD> does not exist at
#                   it, an artifact already has a .deploy-baseline (remove it, or drop the option to use it), or with
#                   --first-deploy; a missing version-control tool is exit 10, a failed extraction exit 1.
#                   [ev: retro mcpn4-history-batch-tools Δ4]
#   --target <name>  the deployed ref comes from a record, not from the mirror (<niagara_home>/modules is a build sandbox, never
#                   the deployed state): the line "<name> <MOD>@<vendorVersion>=<commit>" of <module-root>/.deploy-targets
#                   (one per target and module; "#" starts a comment) names the commit the station runs, used as
#                   --schema-baseline-ref would be; "==> deploy target <name>: ..." says what was read, and a vendorVersion
#                   that differs from the defaultModuleVersion at that commit is a WARN. No line for the module, or one
#                   that does not read, is exit 2 before gradle; it excludes --schema-baseline-ref and --first-deploy.
#                   Without it, --schema-baseline-ref WARNs when the mirror holds another version, worded "the mirror
#                   holds <MOD>-rt X (a build, not necessarily what a station runs); deployed version unknown".
#   --restore-mirror <kept-jar>  builds nothing: copies a jar a mirror build kept (below) back into that mirror's
#                   modules/, leaves the kept copy and ends the cycle (below). Refuses a path not shaped
#                   <mirror>/.mirror-replaced/<name>.<sha12> (exit 2); a missing kept jar, one whose sha256 prefix is
#                   not <sha12>, or a home without .niagara-mirror (exit 10, nothing written); a failed copy is exit 1.
#   --no-drift-check  skip the deployed-baseline drift gate below (Δ2) — fix the version bump, don't skip, unless
#                   you have a real reason (e.g. this niagara_home was never the deploy target)
#                   The mirror is a build sandbox, not a station: a jar it holds is not a release. After a passing mirror
#                   build (the gate passed, the jar stays in modules/) build.sh writes <module-root>/.build/mirror-built,
#                   one line "<jar file name> <sha256 of the jar bytes>" per jar it installed (created if needed; the lines
#                   of jars this run did not build are kept; nothing is written into modules/, which Niagara reads, nor
#                   for a niagara_home that is no mirror). When a rebuild finds the same vendorVersion with other bytes
#                   and the jar the mirror held has the sha256 recorded for its name, it is this checkout's own earlier
#                   build: one "INFO — the mirror's <jar> is this checkout's own build of <version>, not a release: drift
#                   check skipped" line, no exit 51. No record, an unreadable one or another digest: exit 51 as before.
#                   Gitignore .build/ in the module repository (the kit fixtures' .gitignore ignore build/ only).
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
#   again. A report-module punch-list with FAILs (exit 50) restores it the same way, so a fix rebuilt at the same
#   version is not drift (exit 51) and needs no --no-drift-check; INCOMPLETE (53) leaves the new jar.
#   [ev: retro client-persist-writes P4] Under --no-drift-check there is no backup: nothing is restored and the deployed-jar signer is unavailable.
#   First deploy: when modules/ held no jar of the profile before the build there is nothing to restore, so a gate
#   FAIL removes the jar gradle just installed (one "Removed the newly-installed" line, exit 50 as before); left there,
#   a later run without the alias would take its signer as the expected one and pass. A gate PASS leaves it. Under
#   --no-drift-check nothing is removed either. The development signer is recognised in any case (niagara4.SF).
#   (source: delta B5 of an external session retro, 2026-11-13)
# Signing block precheck (#1121): with a niagaraSigningAlias configured (as above) and a root build.gradle.kts that
#   applies com.tridium.niagara-signing without a niagaraSigning {} block (the lint-structure L15 detection), the
#   plugin ignores the alias and the profile and signs with the throwaway key, so build.sh exits 10 BEFORE gradle
#   (also under --no-preflight) and prints the command that appends templates/signing-root.gradle.kts to the root.
#   A root that does not apply the plugin, or that it cannot scan, is not refused.
# No private key: when gradle ends with "No private key for <alias> found", build.sh prints the signing profile it
#   used (niagaraSigningProfile: the last --gradle-prop, else $ORG_GRADLE_PROJECT_niagaraSigningProfile, else
#   gradle.properties), the keystore the profile names and the aliases that keystore holds (keytool -list, the store
#   password handed over in its environment, never printed). Without a usable password it prints the keytool command
#   to run. The exit code stays 30.
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
#   The content hash covers each entry outside META-INF/ in name order (name, size, bytes); it is taken and compared
#   within one run and is never stored.
#   Retry-safe: gradle's :jar step already overwrote the deployed jar by the time this FAILs, so the gate
#   backs up the pre-build jar and restores it into modules/ on FAIL — a bare re-run then still sees the
#   OLD baseline and catches the same drift again, instead of silently "Up to Date"-passing on the retry.
# Mirror jar kept (retro row B6): a mirror is also the drift baseline of what the station runs, and gradle's :jar
#   step replaces its modules/<MOD>-<p>.jar. When a run ends (any exit) with a modules/ jar that differs from the
#   drift gate's pre-build snapshot, the snapshot is kept as <mirror>/.mirror-replaced/<MOD>-<p>.jar.<sha12> (sha256
#   of the replaced jar; a kept jar is never overwritten or deleted) and stdout ends with one "==> mirror: …
#   replaced (vendorVersion <old> sha256:<old> -> <new> sha256:<new>); first replaced this cycle: vendorVersion <v>
#   sha256:<s>; restore with: <toolbelt>/build.sh --restore-mirror <kept>" line (paths shell-quoted). "First replaced
#   this cycle" is the first jar replaced since the last --restore-mirror, recorded in
#   <mirror>/.mirror-replaced/<MOD>-<p>.jar.first; nothing else ends the cycle, so AFTER DEPLOYING a build to the
#   station delete that .first file (the mirror then holds what the station runs), or the next build still names
#   the pre-deploy jar. A pointer that cannot be written is one WARN, and that line names no first jar (list
#   .mirror-replaced/ and pick the one the station runs). A jar that cannot be kept is one WARN ("was NOT kept").
#   The exit code never changes. Nothing is kept for a
#   real niagara_home, under --no-drift-check (no snapshot), on a first deploy, or when the run restored modules/.
# Mirror source line (Δ2): a mirror is NOT the installation it was made from, so a passing mirror build ends (after the
#   line above, when there is one) with one "==> mirror: <mirror> was made from <installation> (Niagara <version>); the
#   jars built this run were NOT installed in <installation>/modules — to install them, stop the station and close
#   Workbench of that installation, then run: <toolbelt>/check-target-version.sh <jar> <installation> && cp <jars>
#   <installation>/modules/" line (paths shell-quoted). The installation is the marker's "source=" line, its version
#   that of its modules/baja.jar ("unknown" when unreadable). Nothing is printed for a real niagara_home, a marker
#   with no source, or a build that did not pass; the exit code never changes. check-target-version.sh refuses an
#   installation of another Niagara major.minor than the jar was compiled against (exit 1).
#   [ev: retro deploy-target-version Δ2]
#   An INCOMPLETE report ends with a pointer to the schema-risk SKIP row only when an artifact lacks .deploy-baseline;
#   the INCOMPLETE line names --schema-baseline-ref <deployed-ref> as the way to supply it.
# Acorn preflight (exit 10, #1116): when a <mod>-ux/src/rc/vendor holds a .js/.mjs/.cjs file, report-module's
#   vendor-floor check parses it with node + acorn. Right after the preflight, before gradle, node (KIT_NODE, else
#   on PATH) and acorn (require() of KIT_ACORN, else toolbelt/eslint/node_modules/acorn, else "acorn") must resolve;
#   if not the build ends exit 10 printing the one command (`npm install --prefix <toolbelt>/eslint`, or set
#   KIT_ACORN / KIT_NODE), instead of a vendor-floor SKIP at the end of a finished build. Nothing is installed.
#   A module with no vendored script needs neither.
# Post-jar copy lock (exit 32): the plugin's last step copies the jar into <niagara_home>/modules; a Windows
#   process (Workbench, station) holding the old jar open on /mnt/c makes it fail with FileAlreadyExistsException.
#   build.sh names the build/libs jar(s) rebuilt THIS run as the artifact of record (a stale one is flagged
#   "not rebuilt"). [ev: retro panccadia-restart-seq-comp-lockout-hours Δ3]
#   The line after them says to stop the station and close Workbench of that installation (on Windows the station is
#   a service or station.exe) before re-running. [ev: retro deploy-target-version Δ3]
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
# Interrupt: SIGINT, SIGTERM or SIGHUP from the baseline snapshot until the chain passed puts the pre-build
#   modules/ jars back (a first-deploy jar is removed), then removes the temporary directory; exit 128 + the signal.
#   Before the restore the signal is passed on to the gradle the run started, which is waited for (killed after
#   10 s), and no gradle starts after it. A gradle daemon (outside a mirror home) cancels on its own, so prefer
#   interrupting the process group (Ctrl-C, kill -- -<pgid>). [ev: retro p1-build-deploy-safety Δ4]
# Kit freshness (retro R9 ΔB5): a stale kit runs old report-module and build-module gates silently (a rebuild at an
#   old kit read "docs ERROR schema-risk env fault"; after kit-resolve it was CLEAN). Before anything else this
#   launcher runs ../../scripts/kit-resolve.sh (version control lives there, never in toolbelt/) on THIS kit with
#   --offline-ok and prints one first stdout line "==> kit: KIT=<dir> SHA=<sha> (origin/main, clean)". A clean
#   kit-main worktree strictly behind origin/main is fast-forwarded first, as kit-resolve does ("kit-resolve:
#   refreshed <old>→<new>" on stderr, #1122). Any other kit behind origin/main ("stale") or diverged from it is
#   "build.sh: FAIL — the kit is behind origin/main (...): <why>" and exit 10 before gradle. Any other answer
#   (dirty, ahead of origin/main, not a work tree, no resolver) is not a stale kit: the line is "==> kit: <dir>
#   freshness unchecked — <why>" and the run goes on. $BUILD_N4_KIT_CHECK=0 skips the check (inner rebuild loops,
#   CI); $BUILD_N4_KIT_RESOLVE names another resolver (tests). -h and --restore-mirror skip it. The Go binary never sees these lines.
# Exit: 0 chain passed (or mirror restored) · 1 a copy the run needs failed (temporary dir, baseline, --restore-mirror,
#   --schema-baseline-ref extraction) · 2 usage (and the --schema-baseline-ref refusals above) · 10 environment (preflight FAIL, no JDK 8, not a niagara_home, a real niagara_home
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
# Kit freshness check (header above): the first stdout line names the kit and its revision.
kit_check() {
  local a kit resolver out rc line why
  [ "${BUILD_N4_KIT_CHECK:-1}" = 0 ] && return 0
  for a in "$@"; do case "$a" in -h | --help | --restore-mirror) return 0 ;; esac; done
  kit="$(cd "$here/.." && pwd)"
  resolver="${BUILD_N4_KIT_RESOLVE:-$here/../../scripts/kit-resolve.sh}"
  if [ ! -x "$resolver" ]; then
    printf '==> kit: %s freshness unchecked — no kit-resolve.sh beside this kit\n' "$kit"
    return 0
  fi
  out="$(BUILD_N4_KIT="$kit" "$resolver" --offline-ok 2>&1)"; rc=$?
  line="$(printf '%s\n' "$out" | grep '^KIT=' | head -1)"
  if [ "$rc" -eq 0 ] && [ -n "$line" ]; then
    printf '%s\n' "$out" | grep -v '^KIT=' >&2
    printf '==> kit: %s\n' "$line"
    return 0
  fi
  why="$(printf '%s\n' "$out" | grep 'REFUSED' | head -1)"
  [ -n "$why" ] || why="$(printf '%s\n' "$out" | head -1)"
  case "$why" in
    *"REFUSED stale:"* | *"REFUSED diverged:"*)
      printf 'build.sh: FAIL — the kit is behind origin/main (a stale kit runs old gates): %s\n' "$why" >&2
      return 10 ;;
  esac
  printf '==> kit: %s freshness unchecked — %s\n' "$kit" "$why"
}
kit_check "$@" || exit $?
exec "$BIN" "$@"
