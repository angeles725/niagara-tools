<!-- ledger: kit -->
<!-- date: 2026-11-13 -->
<!-- last-change-note: dead bash lint libs removed; lint doctrine points at the Go engine -->
log: 2026-11-13 · all-Go pass, unit 1 → `toolbelt/lib/awk-lint.sh` and `toolbelt/lib/scan-files.sh` removed with `tests/awk-lint.bats`, `tests/scan-files.bats` and ER3/ER4 of `tests/lint-empty-root.bats` (no script sourced them; every lint is an `n4lint` wrapper). `BUILD-LOOP.md` § 5 and `skill/SKILL.md` point at `go/internal/lint` (`lint.Walk`, the shared runner). The owed NUL-separated listing closes with the removal. `ORCHESTRATION.md` loses an `[ev:]` pointer into the home directory that failed the pre-push cite check.
open_issue: retro remove-dead-bash-lint-libs Δ1-Δ2 pending (kit-cite-resolve: backticked paths in live prose; no evidence token outside the repository)
