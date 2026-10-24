<!-- review-status: folded -->
<!-- module: kit -->
<!-- deltas: 3 -->
# 2026-10-24 · kit · go-lint-inline-site-literal

**Session**: niagara-research quality round 10, branch `r10/inline-site-literal-go` from origin/main d051831
**Delta count**: 3

## What happened
The owed advisory lint G42 (retro modular-line-efficient-quality Δ2, `METHODOLOGY.md` § Domain correctness "Site
and station data live in one data table") shipped to the deferred spec in
ledger.d/2026-10-22-modular-line-efficient-quality.md as the TreeRule `go/internal/lint/rules/inlinesiteliteral`
and the wrapper `toolbelt/lint-inline-site-literal.sh`. It reuses the shared scanners (javasrc for Java literals,
jssrc for JavaScript, the new `javasrc.Unescape` for literal text). The engine support it needs was not on main yet,
so the three engine commits from the duplicate-helper and link-target-flags branches (TreeRule `ErrEmptyRoot`,
`Spec.ModuleRoot`, `Mode Valued` + `Configurer`) were cherry-picked unchanged (one design, identical changes) and
extended in one commit: `Spec.Options` (repeatable value options in `Args.Values`) and `Spec.ValuedStrict`, with
`ParseOption` delegating to the one `ParseValued` parser so the link-target-flags rule keeps its API. Every commit is
green on its own with RED quoted in its body.

## Evidence
- RED: 13 table/option checks and the golden test on a stub `CheckTree`; `TestParseValuedRepeat` on bare
  declarations; `TestUnescape` 3 cases on an identity stub; 7/7 bats with the wrapper moved away
  `[ev: go/internal/lint/rules/inlinesiteliteral/inlinesiteliteral_test.go]`
- Cliente, live module roots only (retro go-lint-link-target-flags Δ1: `_salvage/` and the stale Leon-Guanjuato
  checkout left out): chihuahua, UmbrellaDashboard, Apillm, DashboardFord and the three panccadia-leon modules,
  15-200 ms each. Before tuning 12 rows, 7 false: query ORDs that name no slot below the station root
  (`station:|alarm:|bql:...` ×6 in chihuahua and DashboardPan, `station:|slot:/|bql:... from schedule:...` ×1).
  After the query-ORD tuning 5 rows, true by the spec text: `station:|slot:/Services/<Module>Service` inline (chihuahua 3 copies across
  BChiServlet, ChiEquipmentReader and rc AuditPage.js; UmbrellaReader; DashboardReader), the weakest by intent since
  the path is the same on every site; the Δ2 decision exempts them: 0 rows. No IPv4 row. No file carries a `// site-table` marker yet
- With `--site-pattern 'Cuarto ?[0-9]+'` on DashboardPan: 7 rows, true by the doctrine (the five rooms are baked into
  `BDashboardService` as slot declarations, and `DashboardReader.ROOMS` is the de-facto table: passing it as
  `--site-table` clears its own row)
- Mutation spot checks: dropping the site-table check, the part or rc filters, the first-line marker rule, the
  one-row-per-line rule, the handle shape, the char-literal filter or the 72-byte cut each turns 1-3 cases red (the
  char-literal filter needed a new case: commit d65a41b)

## Proposed kit deltas (propose-never-apply)
| Δ | Delta | Target file / § | Token |
|---|---|---|---|
| Δ1 | **G42 reads site ORDs, not query ORDs; declared differences from the spec.** An ORD is a row only when it names a slot below the station root or a handle (`station:|alarm:|bql:...` and `station:|slot:/|bql:...` read the same on every station: 7/12 Cliente rows were those). Also declared: the check column is the lint ID, the row appends `: "<literal>"`, and `h:` means a handle (`h:<hex>`), never a `h:mm` format. | `toolbelt/lint-inline-site-literal.sh` header; ledger G42 clause | `[ev: go/internal/lint/rules/inlinesiteliteral/inlinesiteliteral.go]` |
| Δ2 | **DECIDED 2026-10-24: a service ORD is not site data.** `station:|slot:/Services/<Module>Service` (any path at or under /Services) is module structure identical on every site, so it is exempt like a query ORD; `station:|slot:/Rooms/Room1` and a `/ServicesX` folder still warn. Cliente live module roots 5 → 0 rows without `--site-pattern` (commit 663b7ac). A reader that pastes the service ORD per file is still a G41-style duplication for review; `Sys.getService(Type)` is the better lookup. | `toolbelt/lint-inline-site-literal.sh` header | `[ev: go/internal/lint/rules/inlinesiteliteral/inlinesiteliteral.go]` |
| Δ3 | **`machinedefaultpath` keeps a private copy of the literal unescape (owed).** `javasrc.Unescape` is now the shared helper; lint-machine-default-path still has its own `unescape` (same body) and moves to the helper after Ayudante1's in-flight G43 change there lands (G41 doctrine: one helper per concern). | `go/internal/lint/rules/machinedefaultpath` | `[ev: go/internal/javasrc/unescape.go]` |

## Lessons
- When the engine support a new rule needs sits on unlanded sibling branches, cherry-pick those commits unchanged and
  extend them additively: one design, and identical changes merge cleanly.
- A `;` between a failing `go vet` and `git commit` commits a broken tree: chain the gate with `&&`, or check before
  every commit.

---
**Status**: FOLDED 2026-10-24 (branch `docs/r10-fold-lints-a`) — Δ1 (already in the wrapper header and `toolbelt/INDEX.md`) and Δ2 `METHODOLOGY.md` § Domain correctness site-data rule; Δ3 parked as an owed `open_issue:` in its `ledger.d/` fragment, `[deferred:]` in `BUILD-LOOP.md` § 5 kit tooling doctrine (needs code).
