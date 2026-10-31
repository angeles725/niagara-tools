<!-- ledger: kit -->
<!-- date: 2026-10-31 -->
<!-- last-change-note: port-parity gate: a reference frontend port FAILs on any unwaived absence -->
log: 2026-10-31 · port-parity gate → `toolbelt/lint-port-parity.sh` (go/internal/portparity, go/cmd/port-parity) diffs a reference frontend tree against its port (file basenames, DOM ids, named functions) and FAILs every absence not waived in `<module-root>/.port-parity` with `[decision: <ref>]`; stale waivers WARN; `report-module.sh` runs it on modules that have the file; doctrine in BUILD-LOOP §0 and frontend-standard § Prototype → module. Trigger: a client A dashboard port lost its whole equipment view unseen. [ev: retro reference-port-parity]
open_issue: retro reference-port-parity Δ1-Δ3 pending review
