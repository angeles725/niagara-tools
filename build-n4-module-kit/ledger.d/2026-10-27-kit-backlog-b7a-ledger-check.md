<!-- ledger: kit -->
<!-- date: 2026-10-27 -->
<!-- last-change-note: kit-backlog B7a: ledger-check + scaffolded VERSION-LEDGER (audit A9) -->
log: 2026-10-27 · kit-backlog B7a = audit 2026-10-03 A9, first half (feature doc `odd/tasks/kit-backlog-2026-10-27.md`) → new `go/internal/ledgercheck` + `go/cmd/ledger-check` + `toolbelt/ledger-check.sh` (LC1-LC9; the contract is read from `templates/VERSION-LEDGER.md`, `TestTemplateItself` keeps the two in step; mutations LC1/LC2/LC5/LC6/LC7/LC8/LC9 each red); `toolbelt/scaffold-module.sh` emits `odd/VERSION-LEDGER.md` at the gradle root for every type (bats LCB1-LCB4 in `tests/ledger-check.bats`; the scaffold byte-equality cases exclude `odd/` as they exclude the station-test overlay); BUILD-LOOP §7 names the tool (kit-links L5).
open_issue: `go/README.md` has no `internal/ledgercheck/` + `cmd/ledger-check/` rows (outside the kit-backlog B7 surface).
