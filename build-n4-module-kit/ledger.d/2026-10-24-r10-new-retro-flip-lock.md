<!-- ledger: kit -->
<!-- date: 2026-10-24 -->
<!-- last-change-note: quality round 10: new-retro holds a flock on retros/ so concurrent runs keep every flip -->
log: 2026-10-24 · quality round 10 → `go/internal/newretro` held no lock, so two concurrent `new-retro` runs both read BUILD-STATE.src.md and the later rename dropped the other's `retro_pending` flip (and each regenerated INDEX.md from its own scan). `write` now holds an exclusive `flock(2)` on the `retros/` directory from the existence checks through the stub, the INDEX.md regeneration and the envelope rename; no lock file is left behind and the kernel releases the lock on process death. Follow-up W4 of retro `2026-10-24-go-new-retro` done. Retro `2026-10-24-r10-new-retro-flip-lock`.
