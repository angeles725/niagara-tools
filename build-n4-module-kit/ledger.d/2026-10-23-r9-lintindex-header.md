<!-- ledger: kit -->
<!-- date: 2026-10-23 -->
<!-- last-change-note: quality round 9: one lint-header helper for gen-lint-index and lint-guard-pins -->
log: 2026-10-23 · quality round 9 → new `go/internal/lintheader` `Lines` (leading run of `#` lines, no line cap) is the one header definition for `go/internal/lintindex` (gen-lint-index) and `go/internal/guardpins` (lint-guard-pins); lintindex drops its lines 1-60 window (`HeaderLines`), so `toolbelt/INDEX.md` gains the 6 `[ev:]` tags on `lint-timers.sh` header lines 61-128; contract-violation stderr says `(the leading comment block)`; a blank line now ends the guard-pins header too (no current lint affected). Retro retros/2026-10-23-r9-lintindex-header.md (1Δ, pending). [ev: retro r9-lintindex-header Δ1]
