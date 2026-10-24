<!-- review-status: pending -->
<!-- module: kit -->
<!-- deltas: 3 -->
# 2026-10-23 · kit · go-lint-session-store

**Session**: niagara-research quality round 8 (Ayudante2, Go lint-engine owner), truth audit T5, branch `r8/t5-session-store-go` from origin/main 8c49177
**Delta count**: 3

## What happened
`toolbelt/lint-session-store-lazy-evict.sh` was ported onto the Go lint engine (`go/internal/lint/rules/sessionstore`)
byte for byte first, then its false results were fixed (policy R8-D14), one RED-first commit per fix.

- **Port.** 38 goldens recorded from origin/main's script (12 bats runs, two of them the unreadable-file/dir cases
  covered by `golden.Unreadable` instead, plus flag, root, K20, prune-order, CRLF/noeol and the T5 seed trees).
  Mode `First`, `ContinueOnWalkError` (awk_lint_run), the K20 row kept as `EmptyError.Msg`, gawk leftmost-longest
  matching. The wrapper execs `n4lint lint-session-store-lazy-evict`.
- **F1 static fields.** Any line holding `static` and a `Map<...> name` was a field, so a static method's Map
  parameter, a return type and a local were reported, and a repeated name printed twice. A candidate is now a
  class-level declaration outside every `javasrc.Methods` range and every `static { }` initializer (brace depth on
  `BlankLiterals` text), with a `static` token, a (java.util.) Map type with generics nested up to three levels,
  and `=` or `;` after the name; names are deduped; a table filled only in `static { }` is a constant.
- **F2 every put.** Only the file's first `<field>.put(` (a substring: `OLD_SESSIONS.put(` counted for `SESSIONS`)
  was checked. Every method that puts is now checked with an identifier boundary; `removeIf`/`remove` on a view and
  `iterator()` + `.remove()` count as sweeps; the scheduled action resolves by `Method.Name`.
- **F3 blanked code.** The `//` cut was string-unaware (`"http://"` hid a sweep) and a `remove(` in a logged string
  counted. Puts, sweeps and `Clock.schedule` calls now read blanked code.
- **F4 srcTest + header.** The header said `-ux` only; the script scanned every profile and srcTest. srcTest is now
  pruned (production session stores only; a srcTest root is still scanned) and the header says any profile.

## Evidence
- Byte identity before the fixes (stdout, stderr, exit, `cmp`): every Cliente root, all of Cliente, the decompiled
  corpus (2976 files, 53 rows), tests/fixtures, the kit, `--`, `--strict --strict`, two roots, a trailing `/`, `-h`,
  `-x`, no argument and a missing root `[ev: go/internal/lint/rules/sessionstore/testdata]`
- Timing: Cliente 1.17 s → 0.06 s; decompiled corpus 15.46 s → 0.35 s `[ev: retro go-lint-session-store]`
- Real trees (Cliente, `*-worktrees` pruned): 11 rows → 2, the B1160 store `DashboardConfigSession.java:85` and its
  `_salvage` copy; the 9 removed rows (BFordDashboardService:228, InventoryPlanTest:23, FordDispatchTest:22,
  ChiAuditHelper:399, ChiEquipmentReader:591 twice, ChiAlarmHelper:91 and :1857, BChiDashboardService:1408) are each a
  parameter, a local, a static-initializer table or srcTest `[ev: corpus B1160]`
- RED observed per fix on the seeded goldens; mutation pins SSL-param, SSL-everyput, SSL-literal and SSL-srctest
  each fail when run against origin/main's script `[ev: tests/lint-session-store-lazy-evict.bats]`

## Proposed kit deltas (propose-never-apply)
| Δ | Delta | Target file / § | Token |
|---|---|---|---|
| Δ1 | A truth fix that changes a recorded golden puts `declared difference: <why>` above `-- args --`, so the case says it no longer mirrors the bash lint | `go/README.md` § Adding a lint rule | `[ev: go/internal/lint/rules/golden/golden.go]` |
| Δ2 | Class-level field detection (outside `Methods` and `static { }`) lives in the rule for now; a second rule needing it moves `staticInits`/`staticFields` into `javasrc` with a test | `go/internal/javasrc` | `[ev: go/internal/lint/rules/sessionstore/sessionstore.go]` |
| Δ3 | A declaration whose name sits on the line after its type (chihuahua `NOTES_MAP`, an alarm-notes cache) is not a candidate, as before the port; supporting it adds that row, so decide whether a non-session static cache belongs to this lint first | `toolbelt/lint-session-store-lazy-evict.sh` header | `[ev: retro go-lint-session-store]` |

## Lessons
- The golden files turned each truth fix into a reviewable diff of rows: 7, 3, 1 and 3 recorded cases changed (plus 2 new srcTest cases), every
  other case stayed byte-identical to the bash lint.
- A structural rule cannot tell a session store from a registry: on the decompiled corpus the fixed rule still
  reports static caches with puts and no sweep (37 distinct rows), which is the rule's stated shape, not a lie.
