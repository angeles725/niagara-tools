<!-- review-status: pending -->
<!-- module: kit -->
<!-- deltas: 1 -->
# 2026-10-25 · kit · r10-rnt-exclude-jar

**Session**: niagara-tools quality round 10, kit tool G33 `run-niagara-test run`, a repeatable `--exclude-jar
<glob>` for the disposable copy, branch `r10/rnt-exclude-jar` from 1551c6bb (origin/main)
**Delta count**: 1

## What happened
An install's `modules/` can hold a stale jar of the module under test (an old 1.8 `moduleA-ux.jar`) that the
copy then puts next to the freshly built `moduleA-rt.jar`. The runner copied every top-level jar but dotted-name
strays, so the only way out was editing the install. `--exclude-jar <glob>` now leaves the matching jars out of
the copy. Each rule got a fake-runner RED first; no Windows boot was run:
1. The modules robocopy is never handed a matching jar; the others still travel, and the install keeps its jars.
2. One `PASS  run-niagara-test  copy: excluded <n> jar(s) matching <glob>: <names>` row per glob, at most 10
   names, then `and N more`.
3. A glob matching no top-level jar is `FAIL  run-niagara-test  copy: --exclude-jar <glob> matched no jar in
   <install>\modules`, exit 2, checked in `validate` so nothing is probed, copied or built (probably a typo).
4. The glob is a jar basename of letters, digits, `_ . -` and the wildcards `*` `?`, matched case-insensitively as
   Windows names are; a path separator, `..`, a bracket, whitespace or a cmd.exe/PowerShell metacharacter is exit 2
   before any call. The character whitelist also leaves `path.Match` no syntax it could reject.

## Evidence
- the REDs, then green: `TestRunExcludeJar`, `TestRunExcludeJarListsTenNames`, `TestRunExcludeJarMatchingNothingFails`, `TestRunExcludeJarRefusesNonBasenameGlobs` `[ev: go/internal/niagaratest/run_exclude_test.go]`
- the flag, the match and the row `[ev: go/internal/niagaratest/exclude.go]`
- the guard-drop mutation (no `isExcluded` in `copyHome`'s jar selection) turned `TestRunExcludeJar` and `TestRunExcludeJarListsTenNames` red

## Proposed kit deltas (propose-never-apply)
| Δ | Delta | Target file / § | Token |
|---|---|---|---|
| Δ1 | `run-niagara-test run --exclude-jar '<glob>'` (repeatable) leaves matching top-level `modules\*.jar` out of the disposable copy so a stale jar of the module under test never mixes with the freshly built ones; a glob matching nothing is FAIL, exit 2, before the copy | `types/moduleTest.md` § Running from WSL | `[ev: go/internal/niagaratest/exclude.go]` |

## Lessons
- A filter flag whose glob can match nothing must fail loudly: a silent no-op exclusion leaves the stale jar in the copy and the run green for the wrong reason.
