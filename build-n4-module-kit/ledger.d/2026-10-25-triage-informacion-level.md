<!-- ledger: kit -->
<!-- date: 2026-10-25 -->
<!-- last-change-note: quality round 10: triage-console INFORMACIÓN is a level (truth fix owed by retro go-triage-console) -->
log: 2026-10-25 · quality round 10 → `go/internal/triageconsole` `isLevel` matches `INFORMACIÓN` (UTF-8 and Latin-1) exactly, so its header closes the open WARNING/SEVERE block and its frames no longer attach to the previous row; the bash compared `substr(s, 1, 8)` with the 9-byte "INFORMACI" and never matched. Declared truth difference `truth-informacion-level`; `INFORMACIONAL` and the unaccented `INFORMACION` stay no level, every recorded golden unchanged. Closes the tail-3 open issue [ev: retro go-triage-console] [ev: retro triage-informacion-level]
