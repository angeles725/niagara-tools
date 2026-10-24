<!-- review-status: pending -->
<!-- module: kit -->
<!-- deltas: 3 -->
# 2026-10-24 · kit · go-lint-duplicate-helper

**Session**: niagara-research quality round 10, branch `r10/duplicate-helper-go` from origin/main cb296a9
**Delta count**: 3

## What happened
The owed advisory lint G41 (retro modular-line-efficient-quality Δ1, `METHODOLOGY.md` § rt "One shared helper per
concern, before a second copy lands") shipped to the deferred spec in
ledger.d/2026-10-22-modular-line-efficient-quality.md as the TreeRule `go/internal/lint/rules/duplicatehelper` and
the wrapper `toolbelt/lint-duplicate-helper.sh`. The token-level Java reading landed in the shared scanner
`go/internal/javasrc`, not in the rule: `Tokens` (a literal or text block one token), `Decls` (the method and
constructor bodies of every class body, with class, parameter types, `@Override`, anonymous, private, static
and the return type's last token), `Statements` and `Shape`. Two engine features landed with their own tests:
a TreeRule may return `lint.ErrEmptyRoot` (a module root with no part `src/` file is the ERROR row, exit 3) and
`Spec.ModuleRoot` keeps a module-root rule out of `n4lint --all`, which runs on source roots. Every commit is
green on its own; RED was observed first and is quoted in the commit bodies (from the anonymous-flag commit on).

## Evidence
- RED: 21 javasrc token/declaration cases on stubs, then 7/15 rule table cases and the golden test on a stub
  `CheckTree`; each tuning's case failed before its change `[ev: go/internal/lint/rules/duplicatehelper/duplicatehelper_test.go]`
- Cliente, 10 module roots, 6-30 ms each: 35 rows before tuning (1 same-body, 34 same-signature), 19 of them false:
  overrides without `@Override` (Apillm TableModel getters and command `doInvoke`, DashboardFord value types'
  `toJson`/`httpStatus`/`truncated`), static factories of two types (`refused(String,String)`), and a delegate to
  the shared helper (`AlarmJson.unavailable` calling `FordJson.unavailable`). After tuning: 16 rows, all private or
  static helper copies (true by the doctrine; `escJson(String)` in two chihuahua ux helpers is a thin wrapper pair,
  the weakest): `logError(String,Throwable)` in ColdRoomPan (both checkouts), `isJsonNumber`/`escape` in CompPan
  backups, `requestConfigSaveIfNeeded`/`configSaveDelay` across DashboardPan and ColdRoomPan panels,
  `readParentPlantaIndex`/`ensureChild`/`getChildComponent` in chihuahua, `statusText` (same body, 9 statements)
  in Apillm `[ev: go/internal/lint/rules/duplicatehelper/duplicatehelper.go]`
- sources/decompiled has no `src/` layout (ERROR row by contract); staged as `<m>/<m>-rt/src/` in a scratch copy:
  32 modules, 713 ms, 639 rows (66 same-body, 573 same-signature). Same-body sample all true (BACnet trend-log
  `clockChanged`/`changed`, NPDU `encode`); same-signature sample 9/14 true (copies or near-copies such as
  `fromJob`, `getTV`, `ordinal`, `hasProperty`, per-type `arrayOf`), 5 false (same name, other logic: `count(int)`,
  `network()`, `isFixed(int)`, `getStatus()`, an obfuscated `b(...)`)
- Mutation spot checks: dropping the generated-region skip, the part filter, the comment-only region marker, the
  8-statement floor, the instance-method exemption, the anonymous or the factory/delegate exemptions each turns
  1-2 cases red; a class-identity check was unreachable (one class cannot declare a signature twice) and was removed

## Proposed kit deltas (propose-never-apply)
| Δ | Delta | Target file / § | Token |
|---|---|---|---|
| Δ1 | **G41's signature rule reads helpers only; declared differences from the spec.** The spec's rule (2) ("two non-overriding methods in two classes share name and parameter types") cannot see overriding without types, and 19/35 Cliente rows were overrides, factories or delegates. Shipped: only private or static methods (they cannot override), minus static factories returning their own class and one-statement delegates to the same-named method. Also declared: the check column is the lint ID (`lint-duplicate-helper`, kit convention) and the row adds a `(<name>: ...)` detail after the spec's text. | `toolbelt/lint-duplicate-helper.sh` header; ledger G41 clause | `[ev: go/internal/lint/rules/duplicatehelper/duplicatehelper.go]` |
| Δ2 | **Same-name helpers in vendor-scale trees are about 2 in 3 true.** On the decompiled corpus the signature rule is noisy (5/14 false: same name, other logic). If a client module of that size shows the same rate, the next step is a body-similarity floor for rule (2), measured with real-tree counts, never a silent narrowing. | `go/internal/lint/rules/duplicatehelper` | `[ev: go/internal/lint/rules/duplicatehelper/duplicatehelper.go]` |
| Δ3 | **Not a report-module member and not in `--all`.** It reads module roots (`Spec.ModuleRoot`); report-module runs members per `<artifact>/src`, so wiring it needs a module-level slot first. ER1 (tests/lint-empty-root.bats) already pins its empty-root ERROR row. Identifiers are kept in the shape, as the spec says, so a copy with renamed locals is not caught. | `toolbelt/report-module.sh` | `[ev: tests/lint-duplicate-helper.bats]` |

## Lessons
- A name-collision rule without type information needs a modifier gate: private and static methods cannot
  override, so they are the only methods whose shared name is evidence of a copy.
- A mutation that does not compile counts as zero failures in a grep-based harness: check the build before
  reading a mutation as toothless.

---
**Status**: PENDING — INDEX row generated from the header: `| 2026-10-24-go-lint-duplicate-helper.md | kit | 2026-10-24 | pending | 3 |`
