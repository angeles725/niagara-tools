<!-- ledger: kit -->
<!-- date: 2026-10-25 -->
<!-- last-change-note: quality round 10: slot-coverage per-slot reads OPERATOR from code, not comments (#459) -->
log: 2026-10-25 · quality round 10 → kit issue #459 (refs): slot-coverage per-slot tested OPERATOR on the raw `@NiagaraProperty` text, so a READONLY slot whose annotation comment says "never OPERATOR-writable" was reported MISSING (client station A condenser1AutoLocked); the test now reads the comment-stripped annotation (javasrc.Strip), names and STALE unchanged. rc-scan host-literal in an unselected mode's config stays FAIL (decided, e991a8d7) and browser-floor under `--profile unknown` stays WARN (§ Profiles): no change. Retro: retros/2026-10-25-r10-lint-fp-459-rcscan-slotcov.md.
