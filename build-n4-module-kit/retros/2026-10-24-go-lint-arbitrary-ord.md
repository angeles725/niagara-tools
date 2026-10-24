<!-- review-status: pending -->
<!-- module: kit -->
<!-- deltas: 3 -->
# 2026-10-24 · kit · go-lint-arbitrary-ord

**Session**: niagara-research quality round 10 (Ayudante2, Go lint-engine owner), branch `r10/arbitrary-ord-go` from r10/write-path-go (origin/main merged once, c42eb6bf; r10/write-path-go merged, 8936a15e)
**Delta count**: 3

## What happened
`toolbelt/lint-arbitrary-ord.sh` (an awk_lint_each awk, one awk per file) was ported onto the Go lint engine as
`go/internal/lint/rules/arbitraryord`, byte for byte first, then one RED-first truth fix. Before it, `rules/structure`
dropped its private `dirname`/`isFile`/`isDir` for the shared `lint.Dirname`/`IsFile`/`IsDir` (goldens unchanged).

- **Port.** 26 goldens from the bash: 13 bats runs plus 13 edge trees; `TestUnreadable` holds AO-finderr and
  AO-unreadable. Mode `Raw`, `ContinueOnWalkError`, the K20 `EmptyError` row, and `Spec.ShortScanError` (cherry-picked
  from r10/agent-on-shape-go, 9ee06a58, so AO-unreadable's exact `cannot scan <f>: Permission denied` holds).
  `javasrc.LineCommentAt` is the awk `code_part`. The body comment naming the old lint-write-path awk stripper went
  with the awk body.
- **T1 code.** The call was matched in the raw text before the `//` comment, so a Javadoc or block-comment line, a
  string and a text block holding `BOrd.make(var)` WARNed. It is matched in `linescan.Code` now.

- **T2 guarded prefix (kit #459 a).** A constant-prefixed ord whose client part is rejected by an earlier
  `if (<v>.contains(".."))` in the same scope (through the assignments it is derived from) no longer WARNs.
- **T3 alarm source (kit #459 b).** An ord whose every origin is `BAlarmRecord.getSource()`, followed through
  locals, private-method arguments and private-method returns, no longer WARNs.

## Evidence
- Byte identity before the fix (stdout, stderr, exit, `cmp`): all of Cliente (20 rows) and the decompiled corpus
  (2706 files, 52 rows) `[ev: go/internal/lint/rules/arbitraryord/testdata]`
- Timing: Cliente 1.05 s → 0.06 s; decompiled 9.53 s → 0.25 s `[ev: retro go-lint-arbitrary-ord]`
- T1: Cliente 20 → 18 (chihuahua `ChiLinkHelper.java:643` and `BChiServlet.java:662`, both Javadoc lines); decompiled
  52 → 52; golden edge-code 5 rows gone `[ev: tests/lint-arbitrary-ord.bats]`
- T2: Cliente 18 → 15 (PANCCADIA BDashboardServlet.java:279 and the same shape in Leon-Guanjuato :247 and
  Umbrella :261); T3: 15 → 14 (PANCCADIA :804); decompiled 52 → 52 both times `[ev: tests/lint-arbitrary-ord.bats]`
- The 18 Cliente rows after T1 (14 after T2/T3: chihuahua 12, PANCCADIA 1 (:737), Leon-Guanjuato 1 (:486), Umbrella 0): 12 chihuahua (MX60, live), 1 UmbrellaDashboard (live), 3 panccadia-leon DashboardPan
  (live source), 2 Leon-Guanjuato DashboardPan (the stale local checkout); `*-worktrees` pruned, `_salvage` has none
  `[ev: retro go-lint-arbitrary-ord]`

## Proposed kit deltas (propose-never-apply)
| Δ | Delta | Target file / § | Token |
|---|---|---|---|
| Δ1 | A call split after `BOrd.make(` with the variable on the next line is not matched (no real case today: the one split call in Cliente takes a literal); join code lines as lint-no-system-out does if one appears | `go/internal/lint/rules/arbitraryord` | `[ev: go/internal/lint/rules/arbitraryord/arbitraryord.go]` |
| Δ2 | `golden.Unreadable` asserts awk_lint_run's text only; a `ShortScanError` rule writes its own unreadable test (agent-on-shape, arbitrary-ord). Give `Unreadable` the short form | `go/internal/lint/rules/golden` | `[ev: go/internal/lint/rules/arbitraryord/arbitraryord_test.go]` |
| Δ3 | Review the 16 live Cliente rows and mark the sanitized ones `// lint-arbitrary-ord: reviewed <reason>` | client trees (chihuahua, Umbrella, panccadia-leon) | `[ev: retro go-lint-arbitrary-ord]` |

## Lessons
- A Javadoc line starts with ` * `, not `//`, so a "before the // comment" code view still reads it as code; only a
  stateful stripper (block comments across lines) removes it.
