<!-- ledger: kit -->
<!-- date: 2026-11-19 -->
<!-- last-change-note: lint-dashboard-write-save Go rule; lint-delays reads a Math.max floor in either argument -->
log: 2026-11-19 · `toolbelt/lint-dashboard-write-save.sh` (Go `go/internal/lint/rules/dashboardsave`): a `-ux` part that writes with the request Context and no station save in the module FAILs, unless MODULE-MAP declares `persistence:`; `types/dashboard.md` § Write path exemplar; `lint-delays` accepts `Math.max(1L, x)`. `toolbelt/lint-dashboard-write-save.sh` is a report-module per-module member. Retro lint-dashboard-write-save (Δ1-Δ4 applied). [ev: retro lint-dashboard-write-save Δ1]
