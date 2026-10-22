<!-- review-status: pending -->
# 2026-10-22 · kit · awk-lint-line-loops

**Session**: niagara-research quality round 8, work unit T8b2, branch `r8/t8b2-line-lints` from origin/main 3a58392
**Delta count**: 1

## What happened
This unit lands Δ2 of retro `2026-10-22-awk-lint-skeleton`. Nine shell line-loop lints (agent-on-shape,
null-context-write, se-display, clock-zero-floor, bql-string-concat, no-system-out, uberjar-api-conflict,
arbitrary-ord, bundled-jar-class-version) repeated usage / not-a-directory / temp dir / `scan_files` /
`had_err` with no `--strict`. `toolbelt/lib/awk-lint.sh` gained a flagless arity `raw`, `awk_lint_list`
(the fail-closed walk alone, shared with `awk_lint_run`) and `awk_lint_exit <code>`; the nine lints use them, and
lint-jasmine-ux uses `raw` for its parse only (it scans a sub-root and exits 3 at once). The lib's empty
`AL_AWK_ARGS` expansion now uses the `${a[@]+"${a[@]}"}` idiom, safe under `set -u` on bash < 4.4.

## Evidence
- RED: `bats tests/awk-lint.bats` 12/16 (AL13-AL16 failing) before the lib change; GREEN 16/16 after `[ev: tests/awk-lint.bats]`
- Byte-identity: 10 lints x 171 argument shapes (none, `-h`, `--`, `-`, `--strict <root>`, extra args, file root,
  missing root, unreadable file + sub-directory, three ux shapes, the repo root and every fixture dir holding
  `*.java/*.js/*.jar/*.gradle.kts/module-include.xml`), origin/main vs branch: 10260 files; the only `diff -r`
  lines are bash's own `line N: Permission denied` for an unreadable file (script line numbers moved), empty
  after normalizing `line N:` `[ev: toolbelt/lib/awk-lint.sh]`
- `bats -j 3 tests/lint-*.bats` 427 ok `[ev: tests/awk-lint.bats]`
- Scripts net: callers +52/-246, lib +47/-14, so 161 fewer script lines `[ev: toolbelt/lib/awk-lint.sh]`

## Proposed kit deltas (propose-never-apply)
| Δ | Delta | Target file / § | Token |
|---|---|---|---|
| Δ1 | **A new flagless line-loop lint opens with `awk_lint_parse <name> raw`, `awk_lint_list <find-tests>` and ends with `awk_lint_exit <code>`.** It never pastes the usage / not-a-directory / `scan_files` / `had_err` opening. Why: nine copies of that opening existed for one fail-closed walk. Reviewer check: a lint with its own `scan_files` call over its root is a finding. | `BUILD-LOOP.md` § Lint doctrine (with retro awk-lint-skeleton Δ1) | `[ev: toolbelt/lib/awk-lint.sh]` |

## Lessons
- A refactor that removes lines cannot keep bash's own diagnostics byte-identical: `line N:` in a redirect error
  names the script line, so the snapshot diff must normalize it and say so.

---
**Status**: PENDING — INDEX row appended: `| 2026-10-22-awk-lint-line-loops.md | kit | 2026-10-22 | pending | 1 |`
