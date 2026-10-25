#!/usr/bin/env bash
# run-niagara-test.sh — run Niagara module station tests from WSL, and decide PASS or FAIL from their output.
#
# WHY: `bin\test.exe <module> -v:2` and gradle `niagaraTest` both report success (exit 0, BUILD SUCCESSFUL) on
# runs that ran no test or whose test module the test NRE rejected (types/moduleTest.md § Known gotchas;
# retro testing-parity-btestng). This tool reads the output instead (kit spec C40, Go tool G33).
#
# Usage: run-niagara-test.sh verdict [<log>|-]     (no <log> or - reads standard input)
#        run-niagara-test.sh verdict --testng <testng-results.xml> [--allow-ignored-count <n> --allow-ignored-reason <why>]
#                                    [<log>|-]
#        run-niagara-test.sh run --install <dir> --gradle-root <dir> --part <gradle-part> --module <test-module>
#                                [--timeout <dur, default 30m>] [--keep-home] [--scratch <dir>]
#                                [--jdk8 <WSL JDK 8 dir, default /usr/lib/jvm/java-8-openjdk-amd64>]
#                                [--gradle-user-home <dir, default ~/.gradle>] [--module-verification low]
#                                [--exclude-jar <glob>]... [--restore-gradle-root] [--log-dir <dir>]
#                                [--allow-ignored <Class.method>... | --allow-ignored-count <n>] [--allow-ignored-reason <why>]
# Contract: PASS only when the log holds at least one `Total tests run: N` line and every one has N > 0 and
# `Failures: 0`. FAIL on a summary with 0 tests, failures or no Failures count; a `Failures:` key outside a
# summary with a count above 0 or a value that is no count; `invalid test target(s)`; `failed signature
# validation`; a malformed summary; no summary at all; or a line longer than 1 MiB (the rest is unread). One `PASS|FAIL  run-niagara-test  <detail>` row per
# summary line and per other reason, then the verdict row.
# --testng: `test.exe <module> -v:2 -output:<dir>` prints its `Total tests run` summaries on STDERR; its stdout
# holds only shutdown thread dumps. <dir>/testng-results.xml is the authoritative verdict; a console log
# (capture stderr, or 2>&1, never stdout alone) is a cross-check. PASS only when its root has total > 0,
# failed = 0 and skipped = 0, no test-method (config methods included: a failed setupStation skips its tests) has status
# FAIL, the root counts match the test-methods, and its ignored count is 0. Ignored methods (@Test(enabled = false)
# or excluded) are only a count in the file, never named, so `--allow-ignored-count <n> --allow-ignored-reason <why>`
# (both or neither, else exit 2; n from 1; the reason one printable line) waives exactly n and prints the reason; more
# ignored than n (a new disable) or fewer (a stale allowance) is FAIL. run names them from the source instead: each
# method-level @Test(enabled = false) a type registered in <part>/moduleTest-include.xml declares or inherits from a
# srcTest/ superclass is <SimpleName>.<method>; their count must equal the root ignored ("source and TestNG disagree"
# is FAIL) and `--allow-ignored <Class.method>` (repeatable, with --allow-ignored-reason) waives exactly that set: a
# new disabled method or a stale name is FAIL, so a swapped disable fails although the count holds. A class-level
# disable, a non-literal enabled value or an unreadable registered source is exit 3 before the build; run's count
# pair is the fallback that skips the source, and mixing it with --allow-ignored is exit 2. A missing, malformed, truncated or over-64 MiB file is
# FAIL (exit 1), with a row per failed test-method (20 at most, then "and N more"). A <log>, when given, still
# fails the run on its rejections and failure counts; its missing summary line is no reason.
# Exits: 0 PASS; 1 FAIL; 2 usage; 3 unreadable log, or the Go tool cannot be built.
# run: copies the install (X:\… or /mnt/x/…, never under a user home, never one a running station, NRE or
# Workbench uses: process command lines and a live stations\*\config.bog.lock are probed; a service's station.exe
# whose path a non-elevated probe cannot read is attributed through its niagarad service, or refused) into a new dir under
# %TEMP% with robocopy (bin etc jre lib security defaults, then the top-level modules\*.jar but dotted-name
# strays), builds `:<part>:jar :<part>:moduleTestJar` in WSL against the copy (scratch niagara_user_home and
# user.home; JAVA_HOME and -Porg.gradle.java.installations.paths the --jdk8, GRADLE_USER_HOME the caller's cache), runs `bin\test.exe <module> -v:2 -output:` on it, and judges with `verdict --testng` (stderr as the
# console log; a non-zero exit with green results is FAIL). Cleanup on every path: the Windows processes of the
# run are killed by PID (taskkill /PID /T /F), the install is compared with its snapshot (any change is FAIL),
# and the dir the run created is deleted unless --keep-home. Every real C:\Users\<you>\Niagara4.*\<brand>\registry\
# and security\ is snapshotted before the copy and compared after cleanup: the NRE writing a real user home, or
# leaving no etc\ and security\ in the scratch niagara_user_home, is FAIL (the runner fails closed). test.exe runs straight from WSL
# (cwd the copy) and gets the scratch home as niagara_user_home=<win path> with WSLENV niagara_user_home/w; a
# cmd.exe `set "…"` does not reach it, as WSL re-quotes its inner quotes. Exits: 0 PASS; 1 FAIL; 2 usage or refused;
# 3 environment (cmd.exe, worktree status, robocopy, gradle, cleanup); 4 --timeout reached, the run killed. The tool opens no
# port: a test that needs Fox on loopback only turns Fox off (types/driver-authoring.md § 12.1).
# --module-verification low (TEST ONLY, off by default, `low` its only value): after robocopy, append
# niagara.moduleVerificationMode=low (the file's own line ending) to the COPY's defaults\system.properties, never the
# install's: 4.15 leaves the line commented out, so the code default medium rejects a self-signed test jar ("failed
# signature validation", then "invalid test target(s)", exit 0); 4.14 ships low. A copy whose file resolves inside
# --install or outside the run's scratch dir is refused (exit 2); the install's file is hashed before and after (a
# change is FAIL). low still rejects a module requesting a signature-required permission group. Never use it on a
# commissioning or production home.
# --exclude-jar <glob> (repeatable): the top-level modules\*.jar it matches stay out of the copy (a stale
# <module>-ux.jar of the module under test must not sit next to the freshly built jars); a row per glob lists
# what it left out (10 names, then "and N more"). The glob is a case-insensitive jar basename of letters, digits,
# _ . - * ?; a path separator, .., or a cmd/PowerShell metacharacter is exit 2, and a glob matching no jar is FAIL,
# exit 2, before anything is copied. The install is never written.
# Signing-profile gate: a LocalSigningProfile named by niagaraSigningProfile (GRADLE_USER_HOME/gradle.properties,
# else the gradle root's) with validity - refresh < 32 days is refused (exit 2, both values named, before any copy):
# the plugin sets a new key's notAfter to day <validity> of the current month and re-mints it when fewer than
# <refresh> days are left, so every signing re-mints it and the jars fail signature verification (types/moduleTest.md).
# Gradle-root gate (the Go tool, not this script, reads the worktree; kit-links L2 keeps .sh files git-free):
# --gradle-root must be a version-controlled worktree (else a FAIL row says it cannot verify it, exit 3, before
# any copy). Its porcelain status (-z, every untracked file listed), each listed file sha256-hashed
# (a content change to an already-dirty file counts), is taken before the copy and after cleanup; any new,
# modified, deleted or rewritten path is FAIL (`gradle root changed by the build: <10 paths, then "and N more">`),
# else `gradle root unchanged (<n> tracked changes before, same after)`. Niagara gradle plugin 7.3.40 (4.13) makes
# :<part>:compileModuleTestJava append every @NiagaraType test class to the PRODUCTION <part>/module-include.xml.
# --restore-gradle-root then checks out (literal pathspecs) only the files that were clean before and are now
# modified or deleted, never touches a file already dirty, never deletes an untracked file (it lists them),
# re-reads the status and prints `gradle root restored: <paths>`; the run stays FAIL. It never cleans, resets or
# checks out the whole tree (types/moduleTest.md § Running from WSL).
# Run log: a failed run (exit 1, 3 or 4, once its scratch dir exists) reads, before cleanup deletes the dir, the
# last 200 stderr and 50 stdout lines of test.exe (each cut at 4 KiB) and the -output: dir listing, and writes them
# with the runner's rows and the exit status to <UTC timestamp>-<module>.log (0600) under --log-dir (default
# $XDG_STATE_HOME/run-niagara-test, else ~/.local/state/run-niagara-test, 0700): `diagnostics kept: <path>`. The
# scratch dir and its copy read <scratch> and <copy> (Windows and /mnt forms, any case), %USERPROFILE% and $HOME
# <you>; install paths stay. A green run writes one only with --log-dir or --keep-home. Attach the log, not the home.
# A failed run also keeps <stem>.testng-results.xml (a copy) and <stem>.traces.txt (the whole full-stacktrace of
# every FAIL test-method) beside the log, redacted the same way, each cut at a line end under 4 MiB with a
# `(truncated: ...)` row; a missing or unreadable results file is a `testng-results.xml unavailable` row instead.
#
# Implementation: go/internal/niagaratest (goldens in its testdata/, hand-written: no bash tool to record from).
# go/go-tool.sh `go_tool_bin` builds go/bin/run-niagara-test when it is missing or older than a Go source, then
# this script execs it; no Go toolchain or a failed build is exit 3.

set -u
# Resolve symlinks: a stubbed toolbelt (tests/helpers/stub-toolbelt.bash) links this script from another dir.
here="$(cd "$(dirname "$(readlink -f "${BASH_SOURCE[0]}")")" && pwd)"
GO_DIR="$(cd "$here/../.." && pwd)/go"
[ -f "$GO_DIR/go-tool.sh" ] || { printf 'run-niagara-test: no Go module at %s (go-tool.sh missing)\n' "$GO_DIR" >&2; exit 3; }
# shellcheck disable=SC1091  # go/ helper, resolved at runtime via GO_DIR
. "$GO_DIR/go-tool.sh"
BIN="$(go_tool_bin "$GO_DIR" run-niagara-test)" || exit 3
exec "$BIN" "$@"
