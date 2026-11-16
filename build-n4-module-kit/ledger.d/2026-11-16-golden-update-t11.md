<!-- ledger: kit -->
<!-- date: 2026-11-16 -->
<!-- last-change-note: the shared golden runner takes -update to re-record the cases that differ -->
log: 2026-11-16 · golden re-record mode → `go test ./internal/<pkg> -update` rewrites the `want` and `stderr` sections of every golden case whose exit code, stdout or stderr differs (other bytes kept, `(noeol)` when needed), with the case dir resolved to an absolute path so `CHDIR=1` cases work; nothing is written without the flag; documented in `go/internal/lint/rules/golden/record/README.md`; the runner was split by responsibility first and left the size baseline. Closes retro ntpl-lint-walk-k-rows Δ6. Retro golden-update-t11 (Δ1-Δ2 applied, pending review). [ev: retro golden-update-t11 Δ1] [ev: retro golden-update-t11 Δ2]
