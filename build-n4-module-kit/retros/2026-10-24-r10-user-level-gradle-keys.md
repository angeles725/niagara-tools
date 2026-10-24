<!-- review-status: folded -->
<!-- module: kit -->
<!-- deltas: 2 -->
# 2026-10-24 · kit · r10-user-level-gradle-keys

**Session**: niagara-research quality round 10 (`odd/tasks/quality-round10-ayudante1.md`), decision R10-D8,
branch `feat/r10-d8-user-level-gradle-keys` from origin/main 53bfe0c
**Delta count**: 2

## What happened
The kit gave two answers for the host build-location keys (`niagara_home`, `niagara_user_home`, `nodeHome`,
`org.gradle.java.installations.paths`). `BUILD-LOOP.md` bring-up step 2 and `types/structure.md` point 7 said to
copy them from a sibling into the repository's `gradle.properties`. lint-structure L10 FAILed exactly those host
paths, L12 FAILed a module-root `gradle.properties` without the `paths` key, and G43 (#408) reported the keys as
INFO because the docs prescribed them. R10-D8 settles it: the keys live in the per-user Gradle file and the
repository stays machine-independent.

Reading the code before writing found that the brief did not match the lints. L10 scanned `user_home` but not
`org.gradle.java.installations.paths`, and L12 would FAIL once the migration removed `paths`. The architect then
decided: L12 asks only for `org.gradle.java.installations.auto-detect=false`, L10 owns all four keys through one
shared set (`go/internal/props` `BuildLocationKeys`), and `user_home` stays a FAIL.

## Evidence
- Gradle precedence: command line (`-P` for project properties) wins, then `GRADLE_USER_HOME/gradle.properties`,
  then the project and build-root `gradle.properties`, then `GRADLE_HOME`
  `[ev: https://docs.gradle.org/current/userguide/build_environment.html, read 2026-10-24 via context7]`
- Before (origin/main 53bfe0c), over the 10 module roots under `/home/cristian/modulos_niagara_n4/Cliente`:
  L10 24 FAIL rows (`niagara_home` 9, `niagara_user_home` 9, `nodeHome` 6), no other L10 row, L12 0;
  G43 9 INFO rows and nothing else. The brief expected only the `niagara_user_home` lines to change
- After: L10 31 WARN rows, 0 FAIL (`niagara_home` 9, `niagara_user_home` 9, `nodeHome` 6, and 7 new
  `org.gradle.java.installations.paths` rows); L12 0; G43 0 rows
- L12 never fires in Cliente: every `gradle.properties` there sits in the group directory, above the module
  root that L12 reads. The conflict was in the doctrine (`types/structure.md` JDK-pinning section) and in any
  module that keeps the file at its root
- Pins: `tests/lint-structure.bats` LS10-keys and LS10-userhome (each made RED by its named mutation),
  `go/internal/lint/rules/structure` TestHostBuildKeys and TestJdkPin, `tests/lint-machine-default-path.bats`
  MDP6/MDP7, `go/internal/props` TestBuildLocationKey. RM19-line keeps its FAIL relay on `user_home`

## Proposed kit deltas (propose-never-apply)
| Δ | Delta | Target file / § | Token |
|---|---|---|---|
| 1 | **Host build-location keys go in the user-level `gradle.properties`, once per machine.** `~/.gradle/gradle.properties` (WSL/Linux) or `%USERPROFILE%\.gradle\gradle.properties` (Windows), or `$GRADLE_USER_HOME/gradle.properties`; the repository keeps only `org.gradle.java.installations.auto-detect=false`. The old "copy from a sibling" advice is quoted under a dated Superseded note. | `BUILD-LOOP.md` bring-up step 2; `types/structure.md` L10 doctrine points 2, 4, 5, 7 and the JDK-pinning section; `build-verify.md` JDK split | `[ev: https://docs.gradle.org/current/userguide/build_environment.html]` |
| 2 | **One lint owns one key set.** L10 reads `props.BuildLocationKeys`: WARN to migrate in a client tree, FAIL in `build-n4-module-kit/fixtures/` or `templates/` (K16) and for `user_home`; L12 pins only the machine-independent `auto-detect=false`; G43 prints no row for the four keys and dropped its kit-fixture exception. | `toolbelt/lint-structure.sh` header L10/L12; `toolbelt/lint-machine-default-path.sh` header; `METHODOLOGY.md` K16 | `[ev: tests/lint-structure.bats LS10-keys]` |

## User migration checklist (per machine; never edit a client repo first)
1. Create the user-level `gradle.properties` (path above) with the four keys and this machine's values.
2. Build once with `gradlew` from the client root and confirm it picks up the keys (a jar built, major 52).
3. Only then remove the four keys from each client repo's `gradle.properties`.
   `org.gradle.java.installations.auto-detect=false` STAYS in the repo (L12).
4. WSL `toolbelt/build.sh` is unaffected: its `-P` arguments win over both files.

## Lessons
- Read the lint before rewriting its doctrine: the key list in the brief, in the docs and in the regex were three
  different lists.
- Count the real tree before predicting a delta: the predicted change was a third of the actual one.

---
**Status**: FOLDED 2026-10-24 (branch `feat/r10-d8-user-level-gradle-keys`) — Δ1 `BUILD-LOOP.md` bring-up step 2,
`types/structure.md` L10 doctrine points 2/4/5/7 + JDK-pinning section, `build-verify.md` JDK split; Δ2
lint-structure L10/L12 and lint-machine-default-path headers and code, `METHODOLOGY.md` K16. INDEX row generated
from the header: `| 2026-10-24-r10-user-level-gradle-keys.md | kit | 2026-10-24 | folded | 2 |`
