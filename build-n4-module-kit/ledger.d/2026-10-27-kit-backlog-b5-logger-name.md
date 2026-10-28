<!-- ledger: kit -->
<!-- date: 2026-10-27 -->
<!-- last-change-note: kit-backlog B5.4: lint-logger-name (G40) warns a logger not named after the module -->
log: 2026-10-27 · kit-backlog B5.4 (G40, C52 kit half) → new lint `lint-logger-name.sh <module-root>...` (`go/internal/lint/rules/loggername`): WARN a `Logger.getLogger("<x>")` that is neither the part's `moduleName` (part gradle, else group gradle, else the part directory name) nor `<moduleName>.<child>`, and a `Logger.getLogger(X.class.getName())`. `types/observability.md` § 1 points to it; `tests/lint-logger-name.bats` LN1-LN2. [ev: retro n4-course-parity-observability-horizons Δ3]
