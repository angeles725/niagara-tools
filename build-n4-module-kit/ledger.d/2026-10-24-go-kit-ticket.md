<!-- ledger: kit -->
<!-- date: 2026-10-24 -->
<!-- last-change-note: quality round 10: kit-ticket runs the Go port -->
log: 2026-10-24 · quality round 10 → `toolbelt/kit-ticket.sh` is the thin wrapper over `go/cmd/kit-ticket` (`go/internal/kitticket`: gh-absent SKIP ticket, gh issue create with the retro sections as body, LOCAL exit 4 / FAIL exit 3 keep the ticket); 55 goldens (52 recorded from the bash, 3 pid-only truth cases) replay through golden.Runner with a fake gh in the case tree; retro-loop.bats RL4 builds the binary in setup_file and links find/readlink, gh still absent. [ev: retro go-kit-ticket]
