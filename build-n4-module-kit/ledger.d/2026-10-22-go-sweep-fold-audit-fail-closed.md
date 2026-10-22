<!-- ledger: kit -->
<!-- date: 2026-10-22 -->
<!-- last-change-note: quality round 8 T17: sweep-fold-audit fail-open cases closed -->
log: 2026-10-22 · quality round 8 T17 follow-up → `go/internal/foldaudit` closes the four bash fail-open cases kept byte-identical in the port: a leading-zero deltas cell is decimal (bash read `010` as 8 and aborted the Δ loop on `08`/`09`), an overflowing cell is `deltas cell unreadable`, an unterminated last INDEX row is audited, an unreadable INDEX exits 3; `tests/sweep-fold-audit.bats` F12-F14 added (14/14), real-tree output unchanged. [ev: retro go-sweep-fold-audit Δ2]
