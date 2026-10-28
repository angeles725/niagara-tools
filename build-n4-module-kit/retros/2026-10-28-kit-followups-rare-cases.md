<!-- review-status: pending -->
<!-- module: kit -->
<!-- deltas: 4 -->
# 2026-10-28 · kit · kit-followups-rare-cases

**Session**: niagara-tools kit, branch `chore/kit-followups-rare-cases` from 8040cce9
(`odd/tasks/kit-followups-rare-cases.md`, F1-F11)
**Delta count**: 4

## What happened
The advisory § Follow-ups of `odd/tasks/kit-backlog-2026-10-27.md` were worked as rare-case tests. Five of the new
tests went RED on real lint defects (lifecycleguard LG2 key, intrinsiclock IL2 and IL3, charsetdefault receiver,
structure L17 commented `moduleName`), which were fixed; the rest pinned existing behaviour and were shown to have
teeth by mutation. A real `toolbelt/build.sh --werror` run on a client A rt module closed the werror evidence gap.

## Evidence
- Defect fixes with RED first: dcc9b1dd (LG2), dcb947af (IL2, IL3), 6a3c4d5e (charset receiver), 0f0d0066 (L17)
  `[ev: dcc9b1dd]` `[ev: dcb947af]` `[ev: 6a3c4d5e]` `[ev: 0f0d0066]`
- Real `--werror` run: kit 8040cce9, Gradle 7.6 + JDK 8, baseline exit 0, `--werror` exit 30 on 3 javac `[cast]`
  warnings `[ev: 8b5422b5]`
- Native review approved with 3 advisory findings, recorded in the backlog `[ev: 8e775d4e]`

## Proposed kit deltas (propose-never-apply)
| Δ | Delta | Target file / § | Token |
|---|---|---|---|
| Δ1 | Read `moduleName` with comments blanked (`javasrc.Strip`) as L17 now does | `go/internal/lint/rules/loggername/loggername.go:54` | `[ev: 0f0d0066]` |
| Δ2 | Name the IL3 lambda / anonymous-method exception in the wrapper header | `toolbelt/lint-intrinsic-lock.sh` header | `[ev: dcb947af]` |
| Δ3 | version-drift: do not treat sibling `<repo>-worktrees/` checkouts (or the repo root) as drifted sibling modules | preflight version-drift check | `[ev: 8b5422b5]` |
| Δ4 | Groovy `*.gradle` golden with a single-quoted string holding `/*`, and surface the ignored `Strip` error | `go/internal/lint/rules/structure/palette.go:37-39` | `[ev: 8e775d4e]` |

## Lessons
- A rule that keys findings by a type or simple name merges anonymous bodies; key by the body's opening brace.
- A mutation that leaves a Go variable unused fails the build, not the test; force the path with `(true || cond)`.
- A fresh client worktree stores `gradlew` as 100644; `build.sh` exits 10 until it is made executable.

---
**Status**: PENDING — INDEX row generated from the header: `| 2026-10-28-kit-followups-rare-cases.md | kit | 2026-10-28 | pending | 4 |`
