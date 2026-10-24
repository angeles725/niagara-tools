<!-- review-status: folded -->
<!-- module: kit -->
<!-- deltas: 2 -->
# 2026-10-24 · kit · r10-buildsh-mirror-guard

**Session**: niagara-research quality round 10 (Ayudante1), branch `fix/r10-buildsh-mirror-guard` from origin/main 5fb3e454
**Delta count**: 2

## What happened
A client build ran `toolbelt/build.sh` with `niagara_home` pointed at a real Windows install. The niagara-module
plugin's `:jar` step copies the built jar into `<niagara_home>/modules` as its last action, so the install's jars
were replaced without any prompt. The drift gate keeps a copy of the pre-build jar, but in a `mktemp -d` dir that
an EXIT trap deletes, so on a passing build no copy survives. `build.sh` already knew what a mirror is (the
`.niagara-mirror` marker drives the user.home isolation), yet it built into any home.

`build.sh` now fails closed: a `niagara_home` without `.niagara-mirror` exits 10 before preflight and gradle, and
the message names `mirror-niagara-home.sh` and the opt-in `--install-into-home`. With the flag the build runs as
before.

## Evidence
- `toolbelt/build.sh` drift gate: `DRIFT_DIR="$(mktemp -d)"` with `trap 'rm -rf "$DRIFT_DIR"' EXIT` (pre-change
  lines 314-315)
- RED on origin/main 5fb3e454: `BS-home-guard` (marker-less home, no flag, expected exit 10) got exit 0;
  `BS-home-guard-optin` got exit 2 (unknown flag). GREEN after the guard: `tests/build-sh.bats` +
  `tests/mirror-niagara-home.bats` 92/92
- The RED case's gradlew stub writes `modules/Foo-rt.jar` the way the plugin does, so "nothing written to
  modules/" is observed, not assumed; the opt-in positive control sees the jar appear
- Callers: `tests/build-sh.bats` keeps its fake home as a stand-in for a real install through a wrapper that adds
  `--install-into-home`; `tests/scaffold-module.bats` TC4 now builds into a throwaway mirror of `$NIAGARA_HOME`
  instead of the install itself

## Proposed kit deltas (propose-never-apply)
| Δ | Delta | Target file / § | Token |
|---|---|---|---|
| Δ1 | **A build tool that writes into an install must be opt-in.** The default target is a marked throwaway copy (a mirror); writing the real install takes an explicit flag, and the refusal names both routes. | `METHODOLOGY.md` (fail-closed rules); `build-verify.md` § Building against a running station | `[ev: tests/build-sh.bats BS-home-guard]` |
| Δ2 | **A backup deleted on success is not a backup.** A temp copy that an EXIT trap removes protects only the FAIL path; a tool that overwrites user data either keeps the copy past a pass or refuses the overwrite by default. | `METHODOLOGY.md`; `toolbelt/build.sh` drift-gate header | `[ev: retro r10-buildsh-mirror-guard]` |

## Fold
- Δ1 → `METHODOLOGY.md` § Build, a tool that overwrites user data is opt-in; the instance is the existing `build-verify.md` § Building against a running station line, now cited by Δ.
- Δ2 → the same `METHODOLOGY.md` § Build line (a backup an EXIT trap deletes protects only the FAIL path) and the `build-verify.md` line; the `toolbelt/build.sh` drift-gate header is code, left as is.

## Lessons
- A marker the tool already reads for one purpose is the cheapest guard for another: no new state was needed.
- A test fixture that stands for a real install should say so with the opt-in flag rather than borrow the
  mirror marker, which would switch on unrelated mirror behavior.

---
**Status**: FOLDED 2026-10-24 (branch `docs/r10-fold-late`) — INDEX row: `| 2026-10-24-r10-buildsh-mirror-guard.md | kit | 2026-10-24 | folded | 2 |`
