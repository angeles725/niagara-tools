<!-- ledger: kit -->
<!-- date: 2026-11-13 -->
<!-- last-change-note: lint-vendor-floor is a Go command driving node + acorn; CI Go job runs its acorn tests -->
log: 2026-11-13 · all-Go pass, unit 3 → `toolbelt/lint-vendor-floor.sh` is a thin wrapper over `go/cmd/lint-vendor-floor` (`go/internal/vendorfloor`: arguments, tool resolution, walk, API table, classification, rows, exit codes; an embedded acorn driver that only parses and tokenizes). No Go dependency added. Differential 52/53 identical; a node that exits without the driver's final record is now exit 3. Bats `VF-disjoint` (it edited the bash source) is replaced by Go tests of the same rule. The CI Go job installs the pinned npm tools and sets `KIT_REQUIRE_JS_TOOLS`, so the acorn tests run there and a missing tool fails.
open_issue: retro go-lint-vendor-floor Δ1-Δ2 pending (walk fails open on a symlinked vendor/ and an unreadable sub-directory; source-mutating tests)
