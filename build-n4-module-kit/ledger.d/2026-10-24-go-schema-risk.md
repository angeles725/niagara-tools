<!-- ledger: kit -->
<!-- date: 2026-10-24 -->
<!-- last-change-note: quality round 10: schema-risk runs the Go classifier -->
log: 2026-10-24 · quality round 10 → `toolbelt/schema-risk.sh` is the thin wrapper over `go/cmd/schema-risk` (`go/internal/schemarisk`, the B795 §795.4 table embedded; exit 0 SAFE / 1 LOSSY / 2 OUTAGE / 3 usage / 4 env); multi-row categories print in declaration order, an empty before snapshot adds every after slot, an unreadable `*.java` is env (4), an unparsed slot annotation is an UNKNOWN row and the type name comes from the `<type>` element (bats SR-multi, SR-empty-before, SR-unreadable-java, SR-unbalanced, SR-type-name). [ev: retro go-schema-risk]
