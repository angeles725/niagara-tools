<!-- ledger: kit -->
<!-- date: 2026-10-24 -->
<!-- last-change-note: quality round 10: new-retro runs the Go stub writer -->
log: 2026-10-24 · quality round 10 → `toolbelt/new-retro.sh` is the thin wrapper over `go/cmd/new-retro` (`go/internal/newretro`: stub, retros/INDEX.md regenerated in process through `retroindex`, then the BUILD-STATE.src.md retro_pending flip); 36 goldens recorded from the bash byte-identical, the whole slug is checked, and a `--date` that is no YYYY-MM-DD date or a `--deltas` that is no count is usage, exit 3, before anything lands (bats NRT1, NRT2). [ev: retro go-new-retro]
