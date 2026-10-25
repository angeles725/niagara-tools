<!-- ledger: kit -->
<!-- date: 2026-10-25 -->
<!-- last-change-note: quality round 10: a failed run-niagara-test run keeps test.exe's whole stderr beside its log -->
log: 2026-10-25 · quality round 10 → a failed `run-niagara-test run` (`go/internal/niagaratest/failevidence.go`) reads `out/stderr.txt` before cleanup and keeps `<stem>.stderr.txt` (test.exe's whole stderr, the station log) beside `<stem>.log`, redacted by the log's own redactor and cut at a line end under 4 MiB with a marker row; a missing or unreadable stderr is a row in the log. A real run's 200-line stderr tail had cut the station-log lines around the failing class. Retro retros/2026-10-25-r10-rnt-full-stderr.md (1Δ, folded). [ev: retro r10-rnt-full-stderr Δ1]
open_issue: proven on the fake runner only; a live failed `run` whose kept `<stem>.stderr.txt` is compared with the console stderr is owed (retro r10-rnt-full-stderr Δ1).
