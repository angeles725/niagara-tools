<!-- review-status: folded -->
<!-- module: kit -->
<!-- deltas: 25 -->
# 2026-11-15 · kit · build-kit-bench-session

**Session**: bench run of the build kit as a newcomer: `skill/SKILL.md` and `BUILD-LOOP.md` followed literally, from an empty directory to a verified, deployed jar, for every type `scaffold-module.sh` offers (`logic`, `dashboard`, `driver`).
**Delta count**: 25

## What happened
The build ran for real: a 4.15 installation on a Windows-backed mount, read through a mirror made by
`toolbelt/mirror-niagara-home.sh`, JDK 8, Gradle 7.6, the niagara-module plugin 7.6.22 that installation ships. All
output went to one scratch directory outside the repository. Each type was scaffolded, built with
`toolbelt/build.sh`, gated, reported and deployed with `scripts/ng-deploy.sh` into a throwaway modules directory, with
and without a backup. A second pass on the `logic` type repeated the steps with newcomer mistakes. One live pass,
authorized during the session, deployed a scaffolded `dashboard` module (`-rt` and `-ux` jars) into the `modules/`
directory of a laboratory installation with a running station; the station was not started, stopped or contacted, so
whether the module loads is not known from this session.

Three things stopped a literal reading before the first jar: the scaffold's plugin version is not the one the
installation ships, the scaffold requires a signing profile the kit never shows complete, and the `logic` scaffold
fails the kit's own report. The deploy wrapper then showed four defects of result (rows Δ1 to Δ3 and Δ5).

Timings (seconds, wall clock, one run each):

| Step | logic | dashboard | driver |
|---|---|---|---|
| scaffold | 0.04 | 0.05 | 0.05 |
| preflight alone | 27.9 | not run alone | not run alone |
| `build.sh` (gradle + gate + report) | 58 with preflight (gradle 20) | 55 without preflight | 62 without preflight |
| `verify-module.sh` alone | 0.04 | 0.11 | 0.06 |
| `report-module.sh` alone | 2.0 | 1.4 | 1.0 |
| `ng-deploy.sh`, backup | 5 | 6 | 12 |
| `ng-deploy.sh --no-backup` | 5 (refused, exit 51) | 6 (refused, exit 51) | 7 (refused, exit 51) |

Mirror creation: 3.9 s for 738 jars. Live pass: `build.sh` with preflight 123 s (gradle 88 s), `ng-deploy.sh` 17.6 s
with the copy onto the Windows-backed mount.

## Evidence
- `ng-deploy.sh` with a wrong `EXPECTED_RT_TYPES` exits 50 after "build ok", and the jars of that build are already in
  `<NIAGARA_HOME>/modules` (removed before the run, present after it with the time of the run): the plugin's `:jar`
  step installs them before any precheck. `build.sh` refuses a home without the `.niagara-mirror` marker (exit 10);
  the deploy wrapper has no such check (no match for the marker in `go/internal/ngdeploy`).
- The same jar is reported by `build.sh` as `signed by alias BENCHSIG, the expected signer benchsigner` and by the
  deploy gate as `signed by alias BENCHSIG (META-INF/<ALIAS>.SF + signature block)`: the deploy does not pass
  `--expected-signer` (the option is named only under `go/internal/buildmodule`).
- A second `ng-deploy.sh` run with no source change exits 51, `bytes changed but vendorVersion is still 1.0.0`, for all
  three types. Two builds of the same sources differ in `buildMillis` of `META-INF/module.xml`, the manifest digests
  and the signature files, and in nothing else (entry-by-entry CRC listing of the two jars).
- `build.sh` on the untouched `logic` scaffold: `FAIL lint-delays … facet-min-zero slot interval has no MIN facet`,
  `report-module: … 1 FAIL … ISSUES`, exit 50; two `WARN facets-req` rows on the same file. The `dashboard` scaffold
  gets `WARN palette-root` on both jars. The `driver` scaffold is clean apart from `slot-coverage 0.0%`.
- The `driver` scaffold builds `-rt` and `-wb`. `ng-deploy.sh --mode C` copies `-rt` only; `--mode A` fails with
  `project 'BenchDriver-ux' not found`. No mode copies a `-wb` jar.
- A signing profile written from the keys `build-verify.md` § Signing per deploy target lists, with
  `niagara.signing.profileType` set to the short class name, fails `:jar` with `Could not determine class for profile`
  (exit 30). The fully qualified class name works; no kit file holds it, and the key usage a throwaway key needs is in
  `types/driver-authoring.md` only.
- The scaffold's `settings.gradle.kts` falls back to plugin 7.6.17 whatever `--target-version` says. Preflight on the
  mirror: `FAIL plugin-pin plugin 7.6.17 missing from …/etc/m2 — copy from Niagara install`, while that directory
  holds 7.6.22. `ng-deploy.sh` has no plugin-version option and fails with gradle's
  `Plugin [id: 'com.tridium.niagara', version: '7.6.17'] was not found`.
- First deploy of a scaffold that is not a git work tree: `copy ok`, then
  `ERROR jars copied, but the deployed commit was not recorded in .last-deploy-sha: git rev-parse HEAD failed`,
  exit 41, after `NOTICE: not a git repository; skipping annotation change detection` earlier in the same run. After
  `git init` and a commit the live pass exits 0. The first build adds 108 generated lines to four scaffolded sources
  and leaves `.gradle/` and `build/` untracked; the scaffold emits no `.gitignore`.
- `build.sh … ../mirror-415` (a relative home): gradle fails with
  `Plugin [id: 'com.tridium.settings.multi-project' …] was not found … Searched in … maven(file:/)`, exit 30.
- `report-module.sh <GROUP dir>`: `4 artifacts · 6 PASS · … 121 SKIP … INCOMPLETE`, exit 3; with the module directory
  the same module reports 35 PASS. Both usages call the argument `<module-root>`.
- `scaffold-module.sh M9 <dir>` with no `--vendor` exits 0 and emits the constant `defaultVendor` of
  `go/internal/scaffoldmodule` as vendor and Java package. `--target-version 4.99 --plugin-version banana` exits 0
  and writes `getOrElse("banana")`.
- First deploy: `backup: _backups/<module>-pre-<stamp>.tar.gz` and then `nothing to archive`; the directory is empty.
  The usage text never says the backups go to `_backups/` under the working directory.
- Live pass output ends at `copy ok` and `station restart required (Java classes deployed)`. Nothing is said about
  the running station, about the signer (a throwaway self-signed key passed the gate), or about what to look at after
  the restart. BUILD-LOOP § 6 has no removal step.
- A module root under a directory with a space builds, but its report rows split the jar path:
  `M6-rt WARN facets-req space/M6/M6/M6-rt/build/libs/M6-rt.jar /…/BM6.java:33 …`.
- `verify-module.sh` with no argument prints `shellcheck disable=SC2329,SC2317` and three lines of its rationale
  before the first line of usage.
- Preflight takes 26 to 28 s on a gradle root two levels below a temporary directory and prints
  `find: '…': Permission denied` for system directories beside it: `versionDrift` in `go/internal/preflight` walks the
  siblings of three ancestor levels to depth 3.
- `skill/SKILL.md` cites `$KIT/scripts/install-skill.sh`; the kit has no `scripts/` directory (the script is in the
  repository's `scripts/`). Its Toolbelt line describes `scaffold-module.sh` as "run in step 5 before lint-timers".
  Its decision table has thirteen rows; the scaffold offers three types and no document maps one to the other.
- `BUILD-LOOP.md` § 1.a step 1 says the scaffold "does not set the execute bit"; the emitted `gradlew` is mode 0755.
  The section names three steps and none is the signing setup the scaffold's own second output line demands.
- `retros/2026-11-14-double-station-save.md` is `pending` with 3 deltas, and `[ev: retro double-station-save Δ1]`,
  `Δ2` and `Δ3` are all cited in the kit; `sweep-fold-audit.sh retros/INDEX.md .` ends
  `412 folded, 412 cited, 0 uncited` and says nothing about it.
- `ExitFault = 1` in `go/internal/ngdeploy/exit.go` is pinned by `TestABackupDirectoryThatCannotBeMadeExits1`; the
  exit list of `go/internal/ngdeploy/usage.txt` has no code 1. No test compares the constants with that list.
- `count-retro-deltas` on a new retro, with an index generated before the retro existed, prints `index=- MISMATCH`
  and no hint; with the index file absent it names `toolbelt/kit-state.sh --write`. Run over
  `retros/2026-11-14-go-ng-deploy-port.md` it prints `rows=11 count=8 header=8 index=8 MISMATCH`, while
  `gen-retro-index.sh --check` reports the retro headers valid.
- Seen and not a finding: removing `@NiagaraType` from the scaffolded component still builds and passes the gate,
  because the generated region kept the type; an interrupted build (SIGINT at 12 s) is followed by a clean rebuild; a
  second `build.sh` on the same mirror exits 51 as `build-verify.md` § Mirror says. `ledger-check.sh` and its
  argument are named in `BUILD-LOOP.md` § 7, and the same section says the retro index is generated.

## Proposed kit deltas (propose-never-apply)
Ordered by value to a module author. Size: S under about 50 changed lines, M, L. "Changes behaviour" means an existing
command gives another result or exit code for an input it accepted before.

| Δ | Delta | Target file / § | Token |
|---|---|---|---|
| Δ1 | APPLIED 2026-11-15. (M, changes behaviour) The deploy never writes `<NIAGARA_HOME>/modules` before its checks pass: a home without the `.niagara-mirror` marker is refused (exit 10) unless the operator opts in with the option `build.sh` already has, and after a failed check the jars the build installed there are removed or restored; the sentence "a failing check copies nothing" then holds for that directory too | `go/internal/ngdeploy` (build, verify) · `usage.txt` · `build-verify.md` § Station deploy | `[ev: ng-deploy.sh --mode A --first-deploy with EXPECTED_RT_TYPES=3: exit 50 "expected 3 <type entries … found 1", jars of that run present in <NIAGARA_HOME>/modules]` |
| Δ2 | APPLIED 2026-11-15. (S, changes behaviour) The deploy gate gets the same expected signer `build.sh` resolves (the `niagaraSigningAlias` property), so a jar signed with another key fails the deploy with exit 50; with no alias configured it prints the WARN `build.sh` prints | `go/internal/ngdeploy/gate.go` · reuse of `go/internal/buildmodule/signer.go` | `[ev: build.sh "the expected signer benchsigner" vs ng-deploy gate "signed by alias BENCHSIG (META-INF/<ALIAS>.SF + signature block)" on the same module]` |
| Δ3 | APPLIED 2026-11-15. (M, changes behaviour) The version-bump precheck compares jar content without `buildMillis`, the manifest digests and the signature files, so a rebuild of unchanged sources is "same bytes" and a retry of a deploy is not refused; a real content change under the same version still exits 51 | `go/internal/ngdeploy/version.go` · `usage.txt` exit 51 | `[ev: second ng-deploy.sh run, no source change, all three types: exit 51 "bytes changed but vendorVersion is still 1.0.0"; the two jars differ only in module.xml buildMillis, MANIFEST.MF and the .SF/.RSA entries]` |
| Δ4 | APPLIED 2026-11-17 (the logic fixture's `interval` carries `BFacets.MIN` = 1 s and `arm()` a `getInterval().getMillis() > 0` guard, so lint-delays PASSes before and after slotomatic; both `setpoint` slots carry `makeNumeric(celsius, 1)`; the dashboard palettes are rooted in `b:UnrestrictedFolder`; every type key of the three fixtures is `<Type>.displayName`, so slot-coverage is 100 %; `tests/scaffold-module-report.bats` TC-REP1 runs `report-module.sh --lint-only` on each type, TC-REP2 runs verify-module's source and palette checks with a stand-in jar). (S, no behaviour change) Every scaffold type passes the kit's own report with zero FAIL and zero WARN it could avoid: the `interval` slot of the `logic` fixture carries a MIN facet, its OPERATOR numerics carry facets, the `dashboard` fixture's palette uses the unrestricted folder; one test scaffolds each type and runs `report-module.sh --lint-only` on it | `fixtures/MinimalPan` · `fixtures/MinimalDash` · a new case beside the scaffold tests | `[ev: build.sh on the untouched logic scaffold: "FAIL lint-delays … facet-min-zero slot interval has no MIN facet", exit 50]` |
| Δ5 | (M, changes behaviour: additive) A module with a `-wb` jar can be deployed: a mode (or a profile list) that builds, verifies, backs up and copies `-wb` beside `-rt`, with its own expected type count; the `driver` type's guide names it | `go/internal/ngdeploy` (options, build, copy) · `usage.txt` · `types/driver-authoring.md` | `[ev: driver scaffold: --mode C copies BenchDriver-rt.jar only; --mode A exit 30 "project 'BenchDriver-ux' not found"]` |
| Δ6 | (S, no behaviour change) A newcomer can make a throwaway signing profile from one place: a commented template profile with the fully qualified `profileType`, the store keys and the one `keytool` command with the code-signing key usage; the scaffold's signing line and `BUILD-LOOP.md` § 1.a point at it | `templates/` (new profile template) · `BUILD-LOOP.md` § 1.a · scaffold message in `go/internal/scaffoldmodule` | `[ev: profile with the short class name: ":jar FAILED … Could not determine class for profile", exit 30]` |
| Δ7 | (M, changes behaviour) The plugin version stops being a first-build trap: the scaffold takes the version from a given installation (or refuses to guess), the `plugin-pin` FAIL lists the versions found under `etc/m2` and the exact option to pass, and the deploy accepts and forwards the same plugin version as `build.sh` | `go/internal/scaffoldmodule` (args) · `go/internal/preflight` (plugin-pin message) · `go/internal/ngdeploy` (options, build) | `[ev: preflight "FAIL plugin-pin plugin 7.6.17 missing from …/etc/m2 — copy from Niagara install" with 7.6.22 present; ng-deploy exit 30 "Plugin … version: '7.6.17' was not found"]` |
| Δ8 | APPLIED 2026-11-17 (the deploy-record half was already on main: #693, `TestADeployOutsideGitEndsZeroWithANotice`; this change adds the fixtures' `.gitignore`, pinned by TC-GI1, and BUILD-LOOP § 1.a step 4, commit after the first build). (S, changes behaviour) A first deploy from a scaffold ends clean: outside a git work tree the deploy record is skipped with the NOTICE the run already prints, not exit 41 after the copy; the scaffold emits a `.gitignore` (`.gradle/`, `build/`, `_backups/`, `.last-deploy-sha`) and § 1.a says to commit after the first build, which rewrites the scaffolded sources | `go/internal/ngdeploy/record.go` · `fixtures/` · `BUILD-LOOP.md` § 1.a | `[ev: fresh scaffold: "copy ok" then "ERROR jars copied, but the deployed commit was not recorded …", exit 41; first build: 4 files changed, 108 insertions]` |
| Δ9 | APPLIED 2026-11-17 (made absolute after the niagara_home checks; messages still name it as given). (S, changes behaviour) `build.sh` makes a relative `niagara_home` absolute, or refuses it with exit 10 naming the value, before gradle runs | `go/internal/buildmodule` (inputs) | `[ev: build.sh … ../mirror-415: gradle "Searched in the following repositories: maven(file:/)", exit 30]` |
| Δ10 | (S, changes behaviour) `report-module.sh` given a gradle root says so: it descends into the one module directory, or exits with a line naming the directory to pass; the two usages stop calling different directories `<module-root>` | `go/internal/reportmodule` · `toolbelt/report-module.sh` header · `toolbelt/build.sh` header | `[ev: report-module.sh <GROUP dir>: "6 PASS · 0 FAIL · 0 WARN · 121 SKIP … INCOMPLETE", exit 3]` |
| Δ11 | APPLIED 2026-11-17 (no `--vendor` is exit 2 with `--vendor is required` and the usage, checked after --type and the ModuleName; `--target-version` must be `<major>.<minor>` and `--plugin-version` `<major>.<minor>.<patch>`, digits only, each refused by name before anything is written; the newline rule is subsumed; every kit caller, bats case and CI step passes `--vendor`). (S, changes behaviour) The scaffold does not guess identity: `--vendor` is required (exit 2 naming it), and `--target-version` and `--plugin-version` are checked for shape before anything is written | `go/internal/scaffoldmodule` (args, `defaultVendor`) | `[ev: scaffold-module.sh M9 <dir>: exit 0 with the default vendor as package; --target-version 4.99 --plugin-version banana: exit 0, getOrElse("banana")]` |
| Δ12 | APPLIED 2026-11-15. (S, no behaviour change) The backup step names an archive only when it wrote one, and the usage text says where backups are kept and how to restore one | `go/internal/ngdeploy/backup.go` · `messages.go` · `usage.txt` | `[ev: first deploy: "backup: _backups/…tar.gz" then "nothing to archive"; _backups/ empty]` |
| Δ13 | APPLIED 2026-11-15. (M, no behaviour change) The deploy ends with what the author does next: the jars and directory it wrote, the signer and that trust is decided by the station, the restart, the console check to run after it and the command that removes the module again; § 6 gains a short "first deploy of a new module" path and a removal step, and says the deploy does not look at whether a station is running | `go/internal/ngdeploy/messages.go` · `BUILD-LOOP.md` § 6 and § 6.a | `[ev: live pass output ends "copy ok" / "station restart required (Java classes deployed)"]` |
| Δ14 | (S, changes behaviour) Report rows keep a path with a space whole | `go/internal/reportmodule` (row parsing of gate output) | `[ev: "M6-rt WARN facets-req space/M6/M6/M6-rt/build/libs/M6-rt.jar /…/BM6.java:33 …"]` |
| Δ15 | (S, no behaviour change) `verify-module.sh` usage starts at its own first line, without the lint directive and its rationale | `go/internal/verifymodule/usage.txt` or the header reader | `[ev: verify-module.sh: first output line "shellcheck disable=SC2329,SC2317"]` |
| Δ16 | (S, changes behaviour) The preflight version-drift walk stays inside the repository or the client tree of the gradle root, and a directory it cannot read is skipped without a line | `go/internal/preflight/drift.go` (`versionDrift`, `treeVersion`) | `[ev: preflight.sh <mirror> <gradle root under a temporary directory>: 27.9 s, "find: '…': Permission denied" rows for unrelated system directories]` |
| Δ17 | APPLIED 2026-11-17 (the installer is named as the repository's `scripts/install-skill.sh`; per-user paths are `$HOME/…`; the decision table has a Scaffold `--type` column; the Toolbelt entry says what the scaffold does; pinned by `tests/scaffold-module-docs.bats` TC-DOC1–TC-DOC4). (S, no behaviour change) `skill/SKILL.md` points only at things that exist and says them once: the install script by its real location, the scaffold's Toolbelt entry with its own description, a column or line mapping each decision-table row to a scaffold `--type` (or "no scaffold"), and no per-user absolute path | `skill/SKILL.md` § Resolve the kit, § Decision Gates, § References | `[ev: ls build-n4-module-kit/scripts: no such directory; "scaffold-module.sh (skeleton from fixtures/MinimalPan; run in step 5 before lint-timers)"]` |
| Δ18 | APPLIED 2026-11-17 (§ 1.a: the gradle root `<out-dir>/<ModuleName>`, no chmod step, signing, plugin version, first build then commit; § 4 example root generic; pinned by TC-DOC5–TC-DOC7). (S, no behaviour change) `BUILD-LOOP.md` § 1.a matches what the scaffold emits: no `chmod` step for a scaffold, the signing and plugin steps the first build needs, the fact that the gradle root is `<out-dir>/<ModuleName>`, and a generic path in place of a client path in § 4 | `BUILD-LOOP.md` § 1.a and § 4 | `[ev: emitted gradlew mode 0755; scaffold output line 2 "signing: set niagaraSigningAlias + niagaraSigningProfile … before the first build"]` |
| Δ19 | (M, changes behaviour) A pending retro whose every delta is already cited as landed is reported: the fold audit prints a row for it and fails under `--strict` | `toolbelt/sweep-fold-audit.sh` (Go `internal/foldaudit`) | `[ev: retros/2026-11-14-double-station-save.md pending, deltas 3, citations Δ1 Δ2 Δ3 present; sweep-fold-audit.sh: "412 folded, 412 cited, 0 uncited"]` |
| Δ20 | (S, no behaviour change) A Go test fails when an exit constant of the deploy is missing from its usage text, and the same check covers the exit lists of `build-verify.md` for `build.sh` and `ng-deploy.sh` | `go/internal/ngdeploy` (new test over `exit.go` and `usage.txt`) | `[ev: ExitFault = 1, TestABackupDirectoryThatCannotBeMadeExits1; no "1" in the exit list of usage.txt]` |
| Δ21 | (S, changes behaviour) The delta counter tells a stale index from a wrong retro: when the generated index has no row for a retro it says to regenerate it (as it does when the file is missing), and a count mismatch already on the main branch is caught by a check that runs the counter over every retro | `go/internal/retrodeltas` · the retro checks of `tests/` | `[ev: count-retro-deltas on a new retro with an index generated before it: "index=- MISMATCH", no hint; on retros/2026-11-14-go-ng-deploy-port.md: "rows=11 count=8 header=8 index=8 MISMATCH" while gen-retro-index.sh --check says the headers are valid]` |
| Δ22 | APPLIED 2026-11-15. (M, changes behaviour) One deploy at a time per modules directory: a run holds an exclusive lock file in `<NIAGARA_HOME>/modules` and in the station directory from before it looks at them until it has ended, a second run is refused (exit 10) with the holder and its age, and a lock a killed run left is never removed automatically. This answers what no row above covered and an independent verification of Δ1 found: two runs started together on one home each put back or removed the jars the other had moved, and the home ended with none of the module's jars. With it: the move aside never replaces a file already aside, and a stopped deploy removes from the home only the jar it built | `go/internal/ngdeploy` (`lock.go`, `aside.go`, `settle.go`) · `usage.txt` · `BUILD-LOOP.md` § 6 | `[ev: 17 of 30 trials of two failing-gate runs started together ended with no module jar in <home>/modules, both runs exit 50 with the "holds the jars it held before the build" NOTICE]` |
| Δ23 | APPLIED 2026-11-17 (refused up front, exit 10, before the backup and gradle: `msgSameDirNoDeploy`; `TestNoDeployIntoTheStationThatIsTheHomeModulesDirectoryIsRefused`). (S, changes behaviour) `--no-deploy` with `STATION_MODULES_DIR` = `<NIAGARA_HOME>/modules` does not replace the station's jars without the prechecks: the build's jars are taken out again, or the run is refused with a line that says the two settings name one directory | `go/internal/ngdeploy` (`home.go`, `settle.go`) | `[ev: code go/internal/ngdeploy/settle.go]` (`settleHome` keeps what the build installed when the run ends with exit 0, and `--no-deploy` ends there before the prechecks) |
| Δ24 | APPLIED 2026-11-17 (recheck of #787: a station jar whose signer cannot be told now falls back to the home instead of switching the check off; `expectedSigner`, `signers.go`: the station's jars of the mode are read in addition to the home's, never instead of them: the home's give the signer whenever the station's give none (none signed, two signers, or only `NIAGARA4`), the two never mixed in one decision; a station jar whose signer cannot be told (unreadable, not a zip, a `.SF` without its block) does not decide: the home gives the signer with a WARN (#787 first gave no signer there; the recheck found that fail-open and #792 corrected it). First applied as "home only when no station jar is signed" (#781); an independent verification found that dropped a signer the code before enforced, corrected the same day. `TestTheGateIsToldTheSignerBuildShExpects`, `TestTheHomeStillGivesTheSignerWhenTheStationGivesNone`, `TestAStationJarWhoseSignerCannotBeToldFallsBackToTheHome`). (S, changes behaviour) The expected-signer fallback of the deploy gate reads the jars the station holds, not only those of `<NIAGARA_HOME>/modules`, so a station whose jars carry the real signer gives it when no alias is configured | `go/internal/ngdeploy/gate.go` (`expectedSigner`) · `go/internal/modjar` | `[ev: code go/internal/ngdeploy/gate.go]` |
| Δ25 | APPLIED 2026-11-17 (reproduced on main 0aa0276a: a missing home and a file both reached gradle; now refused before the backup and gradle by `homeIsDir`, a link to a directory accepted; `TestANiagaraHomeThatIsNotADirectoryIsRefusedBeforeGradle`, the opt-in home pin renamed `TestAMirrorHomeAndAnOptedInHomeDeployAsBefore`). (S, changes behaviour) A `NIAGARA_HOME` that names no directory, or names a file, is refused with exit 10 before gradle runs, with a line that names the value | `go/internal/ngdeploy/home.go` (`guardHome`) | `[ev: code go/internal/ngdeploy/home.go]` (`guardHome` refuses only a directory; pinned as it is by `TestAMirrorHomeAndAHomeThatIsNotThereDeployAsBefore`) |

## Lessons
- A wrapper that builds and a wrapper that deploys need the same guards; a guard added to one is absent from the other
  until a test names both.
- A scaffold is the kit's first claim to a newcomer: it has to pass every check the kit runs, on the installation the
  newcomer has.
- "Copies nothing on failure" has to be read against every directory the build tool writes, not only the one the
  wrapper copies to.
- A deploy that cannot be repeated unchanged teaches the author to bump versions for nothing or to skip the check.

---
**Status**: PENDING — INDEX row generated from the header: `| 2026-11-15-build-kit-bench-session.md | kit | 2026-11-15 | pending | 21 |`
