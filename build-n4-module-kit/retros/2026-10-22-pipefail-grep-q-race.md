<!-- review-status: pending -->
# 2026-10-22 · kit · pipefail-grep-q-race

**Session**: niagara-research quality round 8 (`odd/tasks/quality-round8.md`, task T8f), branch
`r8/t8f-pipefail-grep` from origin/main 3a58392
**Delta count**: 1

## What happened
`toolbelt/verify-module.sh` runs under `set -euo pipefail` and tested presence with
`printf '%s\n' "$LIST" | grep -q PATTERN` at seven sites (signed, types, palette, palette-root, wb-scaffold,
phantom-dep, and the module.xml probe in the jar loop). `grep -q` exits on its first match; if printf has not
finished writing it dies of SIGPIPE (141) and pipefail makes the whole pipeline false, so a pattern that IS
present reads as missing. In tests this showed up as intermittent V4/V8/VMN1/SLOTWALL1/V11 failures in
`tests/verify-module.bats`; on a real jar it would print a spurious FAIL signed/types/typecount.

A sweep of every toolbelt/scripts shell file that enables pipefail found an eighth site with a worse failure
mode: `scripts/ng-deploy.sh` schema-risk precheck pipes the whole `git diff BASELINE..HEAD -- '*/src/com/**/*.java'`
into `grep -qE '^[+-][[:space:]]*@Niagara(Type|Property|...)'`. Once the diff exceeds the 64 KiB pipe buffer
and an annotation change appears early, printf is killed nearly every time and the precheck reports "no
annotation changes": it fails open on exactly the large diffs where a schema change is most likely.

All eight sites now use a here-string, `grep -q… PATTERN <<< "$VAR"`, with the same flags and patterns (a
here-string adds the same single trailing newline as `printf '%s\n'`). `verify-module.sh` check_moduletest_present
`mti=$(find … | head -1)` gains `|| true`: find keeps writing past head's exit when its matches overflow its
stdout buffer (SIGPIPE), and it exits 1 on any unreadable subdirectory even with `2>/dev/null`; either way
pipefail makes the assignment non-zero. The `find -maxdepth 1 … | sort | head -1` in check_phantom_dep stays:
sort writes its few lines in one flush after reading all input, well under the pipe buffer, so neither
producer can be signalled.

## Evidence
- Race reproduction (read-only diagnosis before this unit): `printf '%s\n' "$X" | grep -q P` under
  `set -o pipefail` gave 3-35 false results per 20,000 iterations under load, with `PIPESTATUS=141 0`;
  the here-string form `grep -q P <<< "$X"` gave 0/20,000 `[ev: T8f diagnosis]`
- Large-input reproduction (this unit): a 300,000-byte value whose first line is `+@NiagaraType`, 200 runs:
  pipe form matched 0/200, here-string 200/200 `[ev: T8f session run]`
- New `tests/pipefail-grep.bats` PG1 scans `scripts/*.sh`, `scripts/lib/*.sh`, `toolbelt/*.sh`,
  `toolbelt/lib/*.sh`: RED on 3a58392 with the 8 sites (ng-deploy.sh:479, verify-module.sh
  :214 :223 :419 :424 :447 :487 :642), GREEN after. PG1-neg/PG1-pos pin the scanner; named mutations
  (drop `echo` from the source match; drop the pipefail-file filter) each turn a case red
  `[ev: tests/pipefail-grep.bats]`
- `tests/verify-module.bats` 5 consecutive runs: 44/44 each; `tests/ng-deploy.bats` 43/43 `[ev: T8f session run]`
- About 30 more `printf | grep -q` pipelines live in toolbelt lints that do not enable pipefail; there the
  pipeline status is grep's own, so they are correct and the gate leaves them alone `[ev: T8f sweep]`

## Proposed kit deltas (propose-never-apply)
| Δ | Delta | Target file / § | Token |
|---|---|---|---|
| 1 | **Pipefail scripts test presence with a here-string, never `printf … \| grep -q`.** Any kit or client shell script that sets `-o pipefail` writes `grep -q P <<< "$X"`; a `printf`/`echo` feeding `grep -q` is a false-miss bug (SIGPIPE 141 races on small input, near-certain on input over 64 KiB). A `$(… \| head -1)` assignment in a pipefail script carries `\|\| true` unless the producer provably finishes in one small write. Why: the race silently flipped verify-module checks and made the ng-deploy schema-risk precheck fail open. Reviewer check: `tests/pipefail-grep.bats` PG1 is the gate for the kit; client hook templates get the same rule. | `METHODOLOGY.md` § Conformance rules (shell); `templates/client-pre-push` | `[ev: tests/pipefail-grep.bats PG1]` |

Not applied; a fold review decides it and flips this retro.

## Lessons
- Under pipefail a pipeline's status is the last non-zero stage, so an early-exiting consumer (`grep -q`,
  `head`) turns its producer's SIGPIPE into a failure of the whole test.
- Small-input races hide in CI as flakes; the same bug on a large input is deterministic and fail-open.

---
**Status**: PENDING — INDEX row appended: `| 2026-10-22-pipefail-grep-q-race.md | kit | 2026-10-22 | pending | 1 |`
