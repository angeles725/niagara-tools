# Spec: build-n4-module-kit-v0.2

**Change**: `build-n4-module-kit-v0.2`
**Status**: spec
**Source version**: niagara-tools `v0.3.0` · kit `v0.1`
**Target version**: niagara-tools `v0.4.0` · kit `v0.2`
**Capabilities**: module-verify-gate · kit-build-toolbelt · kit-regression-tests · build-n4-module-kit-doctrine
**Modified Capabilities**: none — `ng-deploy.sh` contract is untouched.

Coordinator resolutions encoded as requirements:
- Q1: success = all fold-in items present + bats suites green; no post-ship build validation in scope.
- Q2: hard doctrine rule — a jar that has not passed `verify-module.sh` MUST NOT go to a station.
- Q3: conservative default check set; `--stored` and `--target-version` are opt-in.
- Q4: `[CERT-live]` markers (B6, G2, G3); A1 is folded as `[CERT]` (Clock/BAbsTime API) + `[INFER · pending station smoke-test]` (restart re-arm behavior, never smoke-tested live per bitácora rt-hardening §3) MUST be preserved after fold-in.
- Q5: one-time catch-up; `review-status` markers on retros are the only sweep hook.
- R2: `CONTRIBUTING.md` bats-core install step lands in PR2 (QA slice), not PR3.

---

## 1. New Capability: module-verify-gate

### Requirement: verify-module.sh CLI contract

`toolbelt/verify-module.sh` MUST accept:

```
verify-module.sh [--target-version X.Y] [--stored] [--src <module-dir>] <jar>...
```

Exit codes: 0 all checks pass; 1 any FAIL; 2 usage error; 3 required POSIX tool missing. Output MUST be English, one line per check per jar: `PASS|FAIL|SKIP  <check>  <jar>  <detail>` (no colon; tab-separated fields). Opt-in checks not requested MUST emit `SKIP` rows. MUST run on bare POSIX tools (unzip, od, grep, awk) — no JDK required.

**Check matrix**:

| Check | Default | Opt-in flag |
|---|---|---|
| Every `.class` in jar has bytecode major == 52 | yes | — |
| `META-INF/NIAGARA4.SF` present | yes | — |
| Every `<type class=...>` in `META-INF/module.xml` resolves to a `.class` in the jar | yes | — |
| `baja vendorVersion` in `module.xml` <= declared target | no | `--target-version X.Y` |
| Zero `Defl:` entries in `unzip -v` output | no | `--stored` |
| For each jar `X-p.jar`: packaged type count in `module.xml` == count in `<module-dir>/X-p/module-include.xml`; AND grep of `<module-dir>/X-p/src/` for `BFacets\.make\(BFacets\.(MIN\|MAX), *-?[0-9]` is empty | no | `--src <module-dir>` |

#### Scenario: non-first class at major 65 fails

- GIVEN a jar where the first `.class` is major 52 and a subsequent `.class` is major 65
- WHEN `verify-module.sh <jar>` runs (no flags)
- THEN exit 1 with a FAIL line for the bytecode check naming that jar

#### Scenario: missing NIAGARA4.SF fails

- GIVEN a jar with no `META-INF/NIAGARA4.SF` entry
- WHEN `verify-module.sh <jar>` runs
- THEN exit 1 with a FAIL line for the signature-presence check

#### Scenario: module.xml type without matching .class fails

- GIVEN a jar whose `module.xml` lists a `<type class="com.ex.Foo">` with no `com/ex/Foo.class` in the jar
- WHEN `verify-module.sh <jar>` runs
- THEN exit 1 with a FAIL line for the type-resolution check

#### Scenario: vendorVersion vs --target-version

- GIVEN a jar with `baja vendorVersion="4.15"` and flag `--target-version 4.14`
- WHEN `verify-module.sh --target-version 4.14 <jar>` runs
- THEN exit 1 FAIL; same jar with `--target-version 4.15` exits 0 PASS

#### Scenario: deflated jar with --stored fails; STORED passes

- GIVEN a jar containing `Defl:` entries
- WHEN `verify-module.sh --stored <jar>` runs
- THEN exit 1 FAIL; a fully STORED equivalent jar exits 0 PASS

#### Scenario: --src type-count drift or raw facet literal fails

- GIVEN `--src <module-dir>` where the jar type count differs from `<module-dir>/X-p/module-include.xml`, or a `BFacets.make(BFacets.MIN, -40)` literal is found in `<module-dir>/X-p/src/`
- WHEN `verify-module.sh --src <module-dir> <jar>` runs
- THEN exit 1 FAIL for the offending check; the other opt-in checks not requested emit SKIP rows

#### Scenario: no jar argument exits 2

- GIVEN `verify-module.sh` called with no positional arguments
- WHEN the script runs
- THEN exit 2 (usage error, not check failure)

---

## 2. New Capability: kit-build-toolbelt

### Requirement: build.sh contract

Usage: `build.sh [--profiles rt,ux,wb] [--target-version X.Y] [--plugin-version V] <module-root> <MOD> [niagara_home]`

`niagara_home` is positional arg 3 or the `$niagara_home` environment variable. Exit codes: 2 usage error; 10 env/path error (missing `etc/m2/repository` or missing `$JAVA8`); 30 gradle task failed; 50 verify-module gate failed. Profile selection: a profile directory is included only when it has a gradle build file AND contains source files; a stub directory (gradle file, no sources) MUST be reported "skipped" and excluded. MUST run `:MOD-p:clean :MOD-p:slotomatic :MOD-p:jar` with `-Pniagara_home` and `-Porg.gradle.java.installations.paths=$JAVA8` per selected profile. MUST forward `-PniagaraPluginVersion` when `NIAGARA_PLUGIN_VERSION` env or `--plugin-version` is given. After gradle, MUST invoke `verify-module.sh` on every produced jar; exit 50 when `verify-module.sh` returns exit 1. Output MUST be English.

#### Scenario: stub -wb profile skipped

- GIVEN `MOD-wb/` has a `build.gradle` but no `.java` source files; `MOD-rt/` and `MOD-ux/` have sources
- WHEN `build.sh <root> MOD <nh>` runs
- THEN `-rt` and `-ux` tasks execute; wb is logged "skipped"; `verify-module.sh` is called on produced jars

#### Scenario: non-niagara_home refused with exit 10

- GIVEN a directory path that lacks `etc/m2/repository`
- WHEN `build.sh <root> MOD <that-path>` runs
- THEN exit 10 before any gradle task with an English error

#### Scenario: no args exits 2

- GIVEN `build.sh` called with no arguments
- WHEN the script runs
- THEN exit 2 and usage text printed to stderr

#### Scenario: --profiles override honored

- GIVEN `--profiles rt` and all profile dirs have sources
- WHEN `build.sh --profiles rt <root> MOD <nh>` runs
- THEN only the `-rt` tasks execute; `-ux` and `-wb` are not invoked

#### Scenario: verify-module.sh invoked after gradle; gate failure exits 50

- GIVEN a successful gradle build producing jars in `build/libs`
- WHEN `build.sh` completes the gradle phase
- THEN `verify-module.sh` is called with those jars; if `verify-module.sh` exits 1, `build.sh` exits 50

### Requirement: mirror-niagara-home.sh contract

Usage: `<source_niagara_home> <mirror_dir> [exclude-jar ...]`

MUST exit 20 when mirror equals or is inside source. MUST exit 20 (without deleting the directory) when an existing mirror dir lacks the `.niagara-mirror` marker file. MUST write `.niagara-mirror` on creation. MUST tolerate an empty source `modules/`. MUST symlink every top-level entry except `modules/`, and every `modules/*.jar` except excluded names (exact-name match). MUST print the linked count.

#### Scenario: mirror equals source refused with exit 20

- GIVEN mirror path == source or is a subdirectory of source
- WHEN `mirror-niagara-home.sh <src> <src>` runs
- THEN exit 20 with no filesystem modifications

#### Scenario: existing dir without .niagara-mirror marker refused with exit 20 and NOT wiped

- GIVEN a pre-existing directory at `<mirror_dir>` lacking `.niagara-mirror`
- WHEN `mirror-niagara-home.sh` is invoked with that path
- THEN exit 20; the directory MUST NOT be deleted or emptied

#### Scenario: happy path links and excludes

- GIVEN a valid source, a fresh `<mirror_dir>`, and one named jar to exclude
- WHEN `mirror-niagara-home.sh <src> <dst> excluded.jar` runs
- THEN `.niagara-mirror` written; excluded.jar is not linked; linked count printed to stdout

#### Scenario: empty modules/ tolerated

- GIVEN `<source>/modules/` exists but is empty
- WHEN `mirror-niagara-home.sh` runs
- THEN exits 0; linked count printed (may be 0 for modules)

### Requirement: stored-repack.sh contract

Usage: `<in.jar> <out.jar>`

Output MUST contain `MANIFEST.MF` as entry #1; `NIAGARA4.SF` and `NIAGARA4.RSA` when present in input MUST come next; all other entries follow. ALL entries MUST be STORED (zero `Defl:` entries in `unzip -v`). MUST refuse to overwrite an existing `out.jar`. Exit 1 on failure.

#### Scenario: output is fully STORED with correct entry order

- GIVEN a deflated input jar with META-INF/MANIFEST.MF, NIAGARA4.SF, NIAGARA4.RSA, and class entries
- WHEN `stored-repack.sh in.jar out.jar` runs
- THEN `unzip -v out.jar` shows zero `Defl:` entries; MANIFEST.MF is the first listed entry

#### Scenario: refuses to overwrite existing out.jar

- GIVEN `out.jar` already exists at the destination path
- WHEN `stored-repack.sh in.jar out.jar` is invoked
- THEN exit non-zero; `out.jar` is unchanged

---

## 3. New Capability: kit-regression-tests

### Requirement: bats test suite contracts

Each test case MUST name the regression it guards. Fixtures MUST be generated in-test with `printf` + `zip` — zero committed binary files.

| Test file | Mandatory cases | Regression guarded |
|---|---|---|
| `tests/verify-module.bats` | major 65 → FAIL; missing SF → FAIL; type without .class → FAIL; 4.15 vs `--target-version 4.14` → FAIL, 4.14 → PASS; deflated + `--stored` → FAIL; STORED → PASS; `--src` type-count drift → FAIL; raw facet literal → FAIL; no args → exit 2 | All-class bytecode blind spot; unsigned jar gate; type-resolution; version gate; STORED gate |
| `tests/build-sh.bats` | fake gradlew records task sequence; stub `-wb` dir (no sources) skipped; no args → exit 2; non-`niagara_home` → refused; `--profiles` override honored; `verify-module.sh` invoked after gradle | DashboardPan-wb regression; bare `${1:?}` abort; plugin-not-found failure; gate skip |
| `tests/mirror-niagara-home.bats` | mirror == source → exit 20; existing dir without marker → exit 20 AND NOT wiped; happy path links jars, excludes named jar; empty `modules/` tolerated | Destroying a real `niagara_home`; wiping an arbitrary dir |
| `tests/kit-links.bats` | every relative `*.md` / `*.sh` reference inside kit `*.md` resolves against kit root, `types/`, repo root, and `~/.claude/skills/build-n4-module/`; `docs/<f>.md` refs are external pointers and ignored; launcher default kit holds METHODOLOGY.md, BUILD-LOOP.md, `types/`, `toolbelt/` | Dangling `checklist-common.md` / `type-dashboard.md` refs; future renames |
| `tests/stored-repack.bats` | output has zero Defl: entries; MANIFEST.MF is entry #1; refuses overwrite | STORED repack integrity; re-deflation after repack |
| `tests/ng-deploy.bats` | UNCHANGED — all 26 existing cases MUST remain green | Regression guard on ng-deploy surface |

#### Scenario: kit-links catches a dangling relative ref

- GIVEN a kit `*.md` file with a relative link pointing to a path that does not exist at resolution time
- WHEN `tests/kit-links.bats` runs
- THEN the test case for that link FAILS

#### Scenario: ng-deploy tests stay 26/26 after PR2

- GIVEN PR2 toolbelt changes applied with no `ng-deploy.sh` content changes
- WHEN `tests/ng-deploy.bats` runs
- THEN all 26 cases pass

---

## 4. New Capability: build-n4-module-kit-doctrine

### Requirement: documentation fold-in completeness and correctness

Each target file MUST contain every lesson in its assigned group with its evidence marker preserved. Superseded claims MUST NOT appear in any kit file after PR1 merges.

| Target file | Required lesson IDs | Forbidden strings (post-PR1) |
|---|---|---|
| `types/dashboard.md` | C1–C13, G2–G5, G10, J1–J4 | `transient build state`; `BFrozenEnum` as linked-value recommendation |
| `types/logic.md` | A1–A4, G1, G6–G9 (incl. G8) + link renames (checklist-common.md→METHODOLOGY.md; type-dashboard.md→types/dashboard.md) | Same |
| `build-verify.md` | B1–B8 with H1/H2/H3 corrections; deploy-target & signing checklist; STORED recipe referencing `stored-repack.sh`; mandatory `-PniagaraPluginVersion` rule with one-plugin-per-install table (4.13.2→7.3.40, 4.14→7.6.17, 4.15.3→7.6.22); `-ux` slotomatic gap note | `transient build state`; `7.6.1/7.6.3/7.6.5` common-set claim |
| `METHODOLOGY.md` | D1, D2 | — |
| `BUILD-LOOP.md` | Preflight sub-step in §0: JDK 8 present; niagara_home candidate + pinned plugin version in `etc/m2`; running-station lock → mirror; §4 doctrine wording | — |
| `SOURCES.md` | ColdRoomPan path: WSL primary `/home/cristian/modulos_niagara_n4/Cliente/Leon-Guanjuato/Paccadia/ColdRoomPan/`; `/mnt/c/...` as Windows-only fallback | — |
| `types/wb-widgets.md` | Link renames + See-also (no new lessons; stays SEED) | — |

`[CERT-live]` lessons B6, G2, G3 MUST be folded with their `[CERT-live]` marker. `[CERT]` and `[INFER]` markers elsewhere MUST be preserved where the retro carried them. A1 carries `[CERT]` for the API and `[INFER · pending station smoke-test]` for the restart re-arm behavior — NOT `[CERT-live]`.

#### Scenario: superseded strings absent after PR1

- GIVEN PR1 merged into the repo
- WHEN `grep -r "transient build state" build-n4-module-kit/` and related superseded strings are searched
- THEN zero matches across all kit files

#### Scenario: [CERT-live] markers preserved

- GIVEN lessons A1, B6, G2, G3 folded into their respective target files
- WHEN each target file is inspected
- THEN each such lesson's text contains `[CERT-live]`

### Requirement: launcher SKILL.md version and hard rules

`~/.claude/skills/build-n4-module/SKILL.md` (outside git) MUST:
- State version `0.2`
- Hard Rules section MUST state the three-tool doctrine: "`toolbelt/verify-module.sh` is THE verify gate. `toolbelt/build.sh` is the recommended WSL build and runs the gate (slotomatic for every profile with sources). `scripts/ng-deploy.sh` is the station DEPLOY wrapper (backup → build → copy → type-count verify; slotomatic guard rt-only) — run `verify-module.sh` on `build/libs` after it."
- Output Contract MUST include "verify-module.sh result" as a named output item
- Kit resolution path logic MUST remain unchanged

#### Scenario: launcher hard rules carry the three-tool doctrine

- GIVEN the updated SKILL.md v0.2
- WHEN its Hard Rules section is read
- THEN `verify-module.sh` is named THE verify gate; `build.sh` is named the recommended WSL build; `ng-deploy.sh` is named the station DEPLOY wrapper with explicit slotomatic-rt-only caveat; there is no wording calling ng-deploy.sh the primary build or build.sh a fallback

### Requirement: release artifacts state

| Artifact | Required state |
|---|---|
| `VERSION` | `0.4.0` |
| `CHANGELOG.md` | Entry under `[v0.4.0]` referencing slug `build-n4-module-kit-v0.2` |
| Kit `README.md` | States kit status `v0.2` |
| `retros/*.md` (3 files) | Each carries `<!-- review-status: folded v0.2 · 2026-09-01 -->` as line 1; files MUST NOT be deleted |
| `CONTRIBUTING.md` | Contains bats-core install step (lands in PR2) |
| `docs/GOTCHAS.md` | Entries for: STORED repackage (B7 recipe); mirror pattern (B5 safety guard); verify-module gate |

#### Scenario: VERSION and CHANGELOG entry

- GIVEN PR3 release commit applied
- WHEN `cat VERSION` and `grep '\[v0.4.0\]' CHANGELOG.md` run in the repo
- THEN `VERSION` outputs `0.4.0` and CHANGELOG contains a `[v0.4.0]` section

#### Scenario: retros tracked with markers, not deleted

- GIVEN PR3 applied
- WHEN `git ls-files retros/` lists the 3 retro files and line 1 of each is read
- THEN all 3 files are tracked; each has `<!-- review-status: folded v0.2 · 2026-09-01 -->` as its first line

#### Verify-phase check: ng-deploy.sh exec bit

`sdd-verify` MUST confirm `git ls-files -s scripts/ng-deploy.sh` shows mode `100755`. This is not a release artifact to set — the git index is already `100755`; the working-tree drift was a local issue. Verify only.
