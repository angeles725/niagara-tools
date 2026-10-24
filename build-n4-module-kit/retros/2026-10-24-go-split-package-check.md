<!-- review-status: pending -->
<!-- module: kit -->
<!-- deltas: 3 -->
# 2026-10-24 · kit · go-split-package-check

**Session**: niagara-research quality round 10, branch `r10/split-package-check-go` from origin/main 53bfe0c
**Delta count**: 3

## What happened
`toolbelt/split-package-check.sh` (find, a read loop forking awk per source file, sort and an awk aggregation) is now
the RootRule `go/internal/lint/rules/splitpackage` and a thin wrapper that execs `n4lint split-package-check`. The
args, exit codes and row shape the client pre-push hook, the CI template and install-client-hooks rely on are
unchanged. The brief named this the first `xmlsrc` consumer (R8-D12); it is not one: the script never reads a
module-include.xml or build file, it only looks for them by name beside a `src/` directory, and the bats fixtures
create them empty. Parsing them would turn an empty or malformed marker into exit 3 for no rule the lint has, so the
port keeps them as markers. Byte-identical first (the bash's stderr `no module` contract is kept: `rules_test` pins it),
then two truth fixes with RED first.

## Evidence
- Goldens recorded from the original bash: 6 bats runs and 22 edge trees (argument shapes, rt/ux of one module vs two
  modules, three modules, every marker kind, declaration shapes, default package, the root as a module, nested
  modules, pruned search directories, no final newline, no module). The symroot run was left out (its root sat outside
  the tree); `TestUnreadable` pins the three failure texts recorded from the bash. RED: 19/28 cases failed against a
  stub `CheckRoot` `[ev: go/internal/lint/rules/splitpackage/splitpackage_test.go]`
- Byte identity old vs new (stdout, stderr, exit): Cliente (8 rows, exit 0; 1 under `--strict`), panccadia-leon,
  Leon-Guanjuato, Honeywell (4 rows), ford, sources/decompiled (exit 3, no module) and the kit repository (1 row).
  Timing: Cliente 1574 → 52 ms `[ev: toolbelt/split-package-check.sh]`
- Declared port differences: SPC-aggfail is removed with its header pin (the aggregation runs in process: there is no
  external sort to fail); a missing root given with a trailing slash is named with it (engine argument check)
- Rows: every Cliente row is false as a split package. Cliente is not one checkout: 4 of the 8 bash rows came from
  `*-worktrees` checkouts (Leon-Guanjuato-worktrees, chihuahua-worktrees), and the 4 left pair the stale Leon-Guanjuato
  checkout with the live panccadia-leon one. Each live checkout alone has 0 rows. The kit repository's row (`com.x` in
  21 test fixtures) is the same class
- Truth fix 1 (RED: `truth-worktrees`, bats SPC-worktrees): `*-worktrees` directories are pruned from the module
  search, as the shared walker prunes them everywhere else. Cliente 8 → 4, Honeywell 4 → 0
- Truth fix 2 (RED: `truth-block-comment`, bats SPC-block): the package declaration is read on code, so a `package`
  line inside a `/* */` comment no longer wins over the real one. Real trees unchanged
  `[ev: go/internal/lint/rules/splitpackage/splitpackage.go]`

## Proposed kit deltas (propose-never-apply)
| Δ | Delta | Target file / § | Token |
|---|---|---|---|
| Δ1 | **split-package-check is not an xmlsrc consumer; name a real one.** It never reads its build markers. R8-D12's first xmlsrc consumer should be a lint that reads module.xml / module-include.xml content (e.g. a dependency or `<def>` check), where a malformed file is a real error. | ledger R8-D12 item | `[ev: toolbelt/split-package-check.sh]` |
| Δ2 | **Run it on one checkout, never on a client folder.** A client directory holding several checkouts of one project (live + stale + salvage) reports every package as a split; the pre-push hook and CI template already run it on one checkout. A sweep over Cliente should loop over live checkout roots. | `toolbelt/split-package-check.sh` header (root scope) | `[ev: tests/split-package-check.bats]` |
| Δ3 | **The root as a module prints its full path.** A build marker and `src/` directly in the root is a module named by the root as given (`${d#"$ROOT"/}` does not strip the root itself); `.` would read better. Kept for byte identity; no real tree has it. | `go/internal/lint/rules/splitpackage` | `[ev: go/internal/lint/rules/splitpackage/testdata/edge-root-module-1.txt]` |

## Lessons
- A brief's premise about what a script reads is checked against the script first: "reads module-include.xml" was a
  file-name test here, and parsing would have broken the bats fixtures' empty markers.
- An in-process aggregation removes a failure mode; its pinned test goes with it as a declared difference, never left
  passing for the wrong reason.

---
**Status**: PENDING — INDEX row generated from the header: `| 2026-10-24-go-split-package-check.md | kit | 2026-10-24 | pending | 3 |`
