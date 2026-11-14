<!-- ledger: kit -->
<!-- date: 2026-11-13 -->
<!-- last-change-note: check-client-source, check-installed-line and check-bats-skips are Go commands -->
log: 2026-11-13 · all-Go pass, unit 2 → `scripts/check-client-source.sh`, `scripts/check-installed-line.sh` and `scripts/check-bats-skips.sh` are thin wrappers over `go/cmd/check-client-source`, `check-installed-line` and `check-bats-skips` (`go/internal/checkclientsource`, `checkinstalledline`, `checkbatsskips`); one git runner in `go/internal/gitcmd`. Differential against the bash scripts 94/95 identical (SK6: the old script leaked a `grep:` line on stderr). The three scripts now need `go` on PATH.
open_issue: retro go-check-scripts-ports Δ1-Δ3 pending (quoted git path fail-open, unreadable BUILD-STATE soft, allowlist dialect)
