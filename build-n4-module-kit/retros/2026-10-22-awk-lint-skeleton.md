<!-- review-status: pending -->
# 2026-10-22 · kit · awk-lint-skeleton

**Session**: niagara-research quality round 8, work unit T8b, branch `r8/t8b-awk-lint` from origin/main 1197614
**Delta count**: 2

## What happened
Quality round 8 applies the modular, line-efficient rule (retro `2026-10-21-modular-line-efficient-quality` Δ1)
to the kit itself. Ten per-file awk lints each carried the same ~50-line skeleton: `--strict`/usage/not-a-directory
parsing, a temp dir with its cleanup trap, the fail-closed `scan_files` walk, one `awk -f main.awk` run per file
with "cannot scan", and the exit-code tail. The issue #226 fail-closed fix had to land in every copy. This unit
moved the skeleton into `toolbelt/lib/awk-lint.sh` (`awk_lint_parse` + `awk_lint_run`), tested it first, and
migrated all ten callers with stdout, stderr and exit codes unchanged.

## Evidence
- RED: `bats tests/awk-lint.bats` 0/12 before the lib existed; GREEN 12/12 after `[ev: tests/awk-lint.bats]`
- Byte-identity: 10 lints x 28 argument shapes (usage, unknown flag, `-`, `--`, extra roots, file root, missing root,
  unreadable file, unreadable sub-directory, `--max-class`/`--max-method` errors, the fixtures and every repo
  `*.java/*.html/*.js` with and without `--strict`) captured from origin/main and from the branch: 840 files, `diff -r`
  empty `[ev: tests/awk-lint.bats]`
- `bats -j 6 tests/lint-*.bats` 427 ok before and after `[ev: tests/awk-lint.bats]`
- The three variants the lib had to keep: root arity (`one` rejects extra roots: set-null-ord, license-isoperational-gate,
  size; `first` ignores them: six lints; `each` scans them all: ws-placement), extra awk inputs (method-boundary,
  lint-size `MAXC`/`MAXM`) and lint-size's own flags (`AL_FLAG_HOOK`) `[ev: toolbelt/lib/awk-lint.sh]`
- Scripts net: callers +63/-544, lib +108, so 373 fewer script lines `[ev: toolbelt/lib/awk-lint.sh]`

## Proposed kit deltas (propose-never-apply)
| Δ | Delta | Target file / § | Token |
|---|---|---|---|
| Δ1 | **A new per-file awk lint sources `lib/awk-lint.sh`; it never pastes the parse/walk/exit skeleton.** It writes only its header, `awk_lint_parse`, its `main.awk` and `awk_lint_run <find-tests>`. Why: ten copies meant ten landings for one fail-closed fix. Reviewer check: a new lint with its own `scan_files` loop or `had_warn`/`had_err` tail is a finding. | `BUILD-LOOP.md` § Lint doctrine (next to the method-boundary shared-parser rule) | `[ev: toolbelt/lib/awk-lint.sh]` |
| Δ2 | **Fold the line-loop lints' shared opening into the same lib.** Nine shell line-loop lints (agent-on-shape, null-context-write, se-display, clock-zero-floor, bql-string-concat, no-system-out, uberjar-api-conflict, arbitrary-ord, bundled-jar-class-version) repeat usage / not-a-directory / temp dir / `scan_files` / `had_err` with no `--strict`; jasmine-ux differs (exits 0 before the walk, scans a sub-root). A flagless parse mode plus a list-only helper would cover the nine. Left out of T8b to keep the unit near its planned size. | `toolbelt/lib/awk-lint.sh`; the nine lints | `[ev: toolbelt/lib/awk-lint.sh]` |

## Lessons
- A shared skeleton must carry every real variant (arity, extra awk inputs, caller flags) as an explicit parameter;
  "almost identical" copies differ in exactly the edge cases a snapshot must cover.
- Byte-identity is proven by snapshotting the old scripts from `git archive origin/main`, not by re-running the
  suites alone: the suites rarely cover `-`, extra roots or flag-argument errors.

---
**Status**: PENDING — INDEX row appended: `| 2026-10-22-awk-lint-skeleton.md | kit | 2026-10-22 | pending | 2 |`
