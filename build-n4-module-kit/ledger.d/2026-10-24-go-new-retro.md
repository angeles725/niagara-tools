<!-- ledger: kit -->
<!-- date: 2026-10-24 -->
<!-- last-change-note: quality round 10: new-retro runs the Go stub writer -->
log: 2026-10-24 · quality round 10 → `toolbelt/new-retro.sh` is the thin wrapper over `go/cmd/new-retro` (`go/internal/newretro`: stub, retros/INDEX.md regenerated in process through `retroindex`, then the BUILD-STATE.src.md retro_pending flip); 33 goldens recorded from the bash byte-identical; the whole slug is checked, a `--date` that is no YYYY-MM-DD date or a `--deltas` outside 0-99 is usage, exit 3, before anything lands, BUILD-STATE.src.md is replaced atomically with the stub and index rolled back on failure, and a module with no section exits 3 while an already-true one says so (bats NRT1-NRT3). [ev: retro go-new-retro]
