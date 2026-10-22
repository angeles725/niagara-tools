<!-- ledger: kit -->
<!-- date: 2026-10-22 -->
<!-- last-change-note: quality round 8 T17: shared deltas-cell parser go/internal/retrocells -->
log: 2026-10-22 · quality round 8 T17 (#226 triage R2) → new `go/internal/retrocells` (`Delta`, `ParseDeltasCell`: decimal, `*CellError` on an empty/non-numeric/>999 cell) is the shared INDEX deltas-cell parser, imported by `go/internal/foldaudit` and meant for the sweep-build-state Go port; sweep-fold-audit counts an unreadable cell on its own `deltas: N unreadable cells` line instead of as "not cited" (still fails `--strict`); `tests/sweep-fold-audit.bats` F15 added (15/15), mutation R2-unreadable observed; real-tree output unchanged. [ev: retro go-sweep-fold-audit Δ2]
