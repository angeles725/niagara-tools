<!-- review-status: folded -->
<!-- module: kit -->
<!-- deltas: 1 -->
# 2026-10-22 · kit · go-lint-driver-scaffold

**Session**: niagara-research quality round 8, work unit T8k-2b, branch `r8/t8k2b-driver-scaffold` from origin/main 627b396
**Delta count**: 1

## What happened
`toolbelt/lint-driver-scaffold.sh` (bash + one awk per file over `lib/method-boundary.sh`) moved onto the shared Go
lint engine as the per-file rule package `go/internal/lint/rules/driverscaffold`, registered in `rules.All()` after
nullcontext; the wrapper keeps its header and execs `n4lint lint-driver-scaffold` (Raw mode, usage exit 3). Issue
#282 is closed: (1) an annotation is dropped from the member header as its name plus its balanced argument list, so
`@Ann(value = f(x), y = g(z)) public void doPing()` no longer names `g` and fails open; (2) a multi-line annotation
argument list no longer leaves its `(` in the header, so the row line is the method header's line, not the
annotation's; (3) `new pkg.Type() {` and `new Outer . Inner() {` at member level open an anonymous class, not a
method, so the members inside are checked; (4) LDS1-LDS10 pass against the wrapper and the LDS6 mutation comment
names the literal-blanking step. An annotation argument list still open at end of file is a new "not trusted"
reason, so that shape WARNs instead of being swallowed silently.

## Evidence
- RED: 11 golden cases recorded from the origin/main script by pointing the bats `L` at a recording wrapper (every
  bats run on a tree), plus 7 #282 table cases with corrected rows, failed against an empty rule (16 subtests) before
  the scanner existed `[ev: go/internal/lint/rules/driverscaffold/driverscaffold_test.go]`
- Byte identity, origin/main script + libs vs wrapper, stdout + stderr + exit: same on tests/fixtures (13 rows),
  Cliente/Honeywell, Cliente/panccadia-leon, Cliente/Leon-Guanjuato (0 rows each), no args, `-h`, `--strict`, a
  missing root, a file root, two roots, an unreadable sub-directory and an unreadable file `[ev: go/internal/lint/run.go]`
- A 1,187-file decompiled driver sample differs only by 229 old "braces do not balance" WARN rows that the new
  rule no longer prints; blanking every annotation line makes the old script drop all 229, so each one came from a
  nested annotation argument (`facets={@Facet(...)}`) cut at its first `)` (#282 item 1) `[ev: issue #282]`
- Timing old -> new, one run each: tests/fixtures 0.68 s -> 0.03 s, Honeywell 1.77 s -> 0.05 s, panccadia-leon
  1.96 s -> 0.06 s, Leon-Guanjuato 0.47 s -> 0.04 s, decompiled sample 45.1 s -> 1.9 s `[ev: go/internal/lint/run.go]`

## Proposed kit deltas (propose-never-apply)
| Δ | Delta | Target file / § | Token |
|---|---|---|---|
| Δ1 | **A header-based method finder drops annotations by balanced parentheses and reads `new <qualified>(` as an anonymous class.** A one-level `\([^)]*\)` annotation regex leaves junk for nested arguments (wrong name, unbalanced braces) and a `new` check that only looks one identifier back reads `new pkg.Type() {` as a method; `rules/driverscaffold/scan.go` does both right, and `javasrc.Methods` (open issue: `new X() {` field named as a method) should take the same two rules when it is next changed. Reviewer check: a Java scanner that strips annotations with a regex, or tests `new` only right before the call name, is a finding. | `go/internal/javasrc/javasrc.go` § Methods | `[ev: go/internal/lint/rules/driverscaffold/scan.go]` |

## Lessons
- Record goldens by swapping the bats lint path for a recording wrapper: every bats run becomes a Go case with its exact bytes.
- Real trees with zero rows prove little; add a corpus that exercises the rule (here a decompiled driver sample).
- When a diff is large, find the one source edit that makes the old tool agree (blanking annotation lines) before calling it intended.

---
**Status**: FOLDED (quality round 8 W23 U0, main e66da0a) — Δ1 `toolbelt/lint-driver-scaffold.sh:27` header + `go/internal/lint/rules/driverscaffold/scan.go:11-25`; the `javasrc.Methods` half rides with the owed `new X() {` item [ev: retro go-javasrc-scanner Δ2] in `ledger.d/2026-10-22-go-lint-engine.md`. INDEX row generated from the header: `| 2026-10-22-go-lint-driver-scaffold.md | kit | 2026-10-22 | folded | 1 |`
