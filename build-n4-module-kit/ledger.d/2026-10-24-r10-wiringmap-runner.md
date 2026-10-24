<!-- ledger: kit -->
<!-- date: 2026-10-24 -->
<!-- last-change-note: quality round 10: wiringmap goldens on the shared golden Runner -->
log: 2026-10-24 · quality round 10 R10-D14 follow-up → `go/internal/wiringmap` replays its 65 goldens through `golden.Runner` (exact count, `CHDIR=1`, `GOLDEN_ALLOW_ROOT_SKIP`); the 58 recorded cases were re-recorded by `golden/record/record.sh` from the pinned bash at 15e792d1 (`lib/scan-files.sh` as `EXTRA`) byte-identical apart from `@kind`, and its private loop and recorder are gone. `toolbelt/new-retro.sh` names the shared recorder. Retro retros/2026-10-24-r10-wiringmap-runner.md (2Δ, pending). [ev: retro r10-wiringmap-runner Δ1] [ev: retro r10-wiringmap-runner Δ2]
open_issue: `toolbelt/generate-wiring-map.sh` header still names the deleted `go/internal/wiringmap/testdata/record/record.sh` (retro r10-wiringmap-runner, 2026-10-24).
