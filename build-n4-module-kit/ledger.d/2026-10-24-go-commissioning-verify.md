<!-- ledger: kit -->
<!-- date: 2026-10-24 -->
<!-- last-change-note: quality round 10: commissioning-verify runs the Go auditor -->
log: 2026-10-24 · quality round 10 → `toolbelt/commissioning-verify.sh` is the thin wrapper over `go/cmd/commissioning-verify` (`go/internal/commissioningverify`; source lints through the invoking toolbelt, bog-audit in process; exit 0 clean / 1 any FAIL / 3 usage or env); the bog-audit exit-3 SKIP row no longer names python3, CHECK15/CHECK19 WARN rows reach the punch-list, every source lint fails closed on a bad exit and an unreadable values-owed table is exit 3 (bats CV-bog-skip, CV-bog-warn, CV-src-failclosed, CV-owed-unreadable). [ev: retro go-commissioning-verify]
