<!-- ledger: kit -->
<!-- date: 2026-11-14 -->
<!-- last-change-note: toolbelt/scaffold-module.sh is a launcher over go/cmd/scaffold-module -->
log: 2026-11-14 · all-Go pass, wave 2 → `toolbelt/scaffold-module.sh` is a launcher over `go/cmd/scaffold-module` (`go/internal/scaffoldmodule`); differential against the bash original over 114 rows: 106 identical, 8 declared (an option value sed rejected and a write fault mid-copy now exit 2 / 3 instead of 0 over a broken tree; `add-concern` without `module-map.sh` words its exit-127 line differently). `add-concern` dispatch runs in Go.
open_issue: retro go-scaffold-module-port Δ1-Δ4 pending (option values as literal text; PF-jarlock cases under a parallel bats run; --vendor validated as a package segment; a failed emit removes its partial tree)
