<!-- review-status: pending -->
<!-- module: kit -->
<!-- deltas: 3 -->
# 2026-10-24 · kit · go-lint-inert-coordination

**Session**: niagara-research quality round 10 (Ayudante2 lint-engine owner), branch `r10/inert-arbitrary-go` from origin/main 45ddb3b5
**Delta count**: 3

## What happened
`toolbelt/lint-inert-coordination.sh` (one gawk process per file) is now the thin `n4lint lint-inert-coordination`
wrapper over `go/internal/lint/rules/inertcoordination`, header, usage, row, limitations and `# Mutation:` pins kept.
No new engine field (First mode, the awk_lint_run walk contract, the K20 row). Byte-identical first, then three
truth fixes, each RED-first in one green commit: T1 code view (comments blanked, literal contents removed: a
commented-out singletonList and names living only in comments or strings were false WARNs, a field after "http://"
was hidden); T2 Arrays.asList arguments counted by parenthesis depth over lines (a call wrapped before two arguments
was singleton, `asList(pick(a, b))` was not); T3 every Deque/Queue declarator named by its own declaration and
deduped exactly (the first `<name> =` of the line, "(Deque/Queue field)" and substring dedupe hid or misnamed
queues). lint-arbitrary-ord, planned for the same branch, waits: lint-write-path.sh references it and another
writer is porting write-path.

## Evidence
- Goldens: 28 cases recorded from the bash lint (11 bats runs, ICO-awkfail/ICO-finderr pinned by `golden.Unreadable`, 17 edge trees); RED 15/28 and TestUnreadable against a stub Check `[ev: go/internal/lint/rules/inertcoordination/inertcoordination_test.go TestGolden]`
- Byte identity (stdout, stderr, exit), default and `--strict`: Cliente 1 row, decompiled 0, tests/fixtures 0; wrapper identical; 1.13 s → 0.25 s on Cliente, 11.10 s → 1.20 s on decompiled
- The Cliente row is live and real: panccadia-leon `Paccadia/ColdRoomPan/ColdRoomPan-rt/.../BDefrostController.java:1190`, units() returns `Collections.singletonList((BEvaporatorUnit) parent)` while the queue/token/stagger machinery is still declared; the class comment itself says the staggerDelay is inert (the Δ5 shape this lint was written for, still in the live source)
- T1: Cliente 1 → 1 (its name list now code-only: token, staggered, stagger, Stagger came from comments), decompiled 0 → 0 `[ev: go/internal/lint/rules/inertcoordination/testdata/truth-code-view-1.txt]`
- T2, T3: Cliente 1 → 1, decompiled 0 → 0, no row text changed `[ev: go/internal/lint/rules/inertcoordination/testdata/truth-aslist-args-1.txt]` `[ev: go/internal/lint/rules/inertcoordination/testdata/truth-queue-names-1.txt]`

## Proposed kit deltas (propose-never-apply)
| Δ | Delta | Target file / § | Token |
|---|---|---|---|
| Δ1 | Client finding: the live panccadia-leon BDefrostController still coordinates over a singleton units(); coordinate defrost at the parent that aggregates the evaporators, or drop the dead queue/token/stagger machinery (types/issues-and-gotchas.md §J1) | `Cliente/panccadia-leon/.../BDefrostController.java:1190` | `[ev: go/internal/lint/rules/inertcoordination/testdata/ico1-1.txt]` |
| Δ2 | The singleton check still accepts singletonList( anywhere in the body (an unrelated local), and the units() declaration needs a blank before `units()` (`List<B>units()` is missed); route the declaration through javasrc.Methods once it names a generic return type correctly | `go/internal/lint/rules/inertcoordination/inertcoordination.go` | `[ev: go/internal/lint/rules/inertcoordination/testdata/edge-decl-1.txt]` |
| Δ3 | ALL-CAPS constants (`STAGGER_MS`) and method parameters of a Queue type are still read as coordination state by name only | `build-n4-module-kit/toolbelt/lint-inert-coordination.sh` | `[ev: go/internal/lint/rules/inertcoordination/testdata/edge-ids-1.txt]` |

## Lessons
- The one real row of this lint is the defect it was written for, still live: a lint that keeps firing on its own motivating case is a tracked client item, not noise.
- Name-list bugs (first `=` on the line, substring dedupe) never change whether a row fires, so only hand-written edge trees reveal them.

---
**Status**: PENDING — INDEX row generated from the header: `| 2026-10-24-go-lint-inert-coordination.md | kit | 2026-10-24 | pending | 3 |`
