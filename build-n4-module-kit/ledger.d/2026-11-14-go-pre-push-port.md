<!-- ledger: kit -->
<!-- date: 2026-11-14 -->
<!-- last-change-note: .githooks/pre-push is a launcher over go/cmd/pre-push -->
log: 2026-11-14 · all-Go pass, wave 2 → `.githooks/pre-push` is a launcher over `go/cmd/pre-push` (`go/internal/prepush`): client guard, live-doctrine citation gate, changelog gate (in process), retro and ledger gate with its trailers, content sweep. Differential against the bash hook over 83 rows: 80 identical, 3 differ in one diagnostic line with the same decision; one mutation proof per gate. No Go toolchain or a failed build refuses the push. A client denylist that cannot be examined now refuses (the bash hook went on unchecked).
open_issue: retro go-pre-push-port Δ1-Δ4 pending (rev-parse failure exits 0; promotion flip under rename detection; live-doctrine-set.sh without a consumer; comments that still describe a bash hook)
