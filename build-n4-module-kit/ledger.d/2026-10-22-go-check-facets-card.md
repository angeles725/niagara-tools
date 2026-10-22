<!-- ledger: kit -->
<!-- date: 2026-10-22 -->
<!-- last-change-note: quality round 8: check-facets-card ported to Go -->
log: 2026-10-22 · quality round 8 → `toolbelt/check-facets-card.sh` ported to `go/cmd/check-facets-card` (`go/internal/facetscard`) and reduced to a thin `go/go-tool.sh` `go_tool_bin` wrapper (exit 3 when it cannot be built); stdout/stderr/exit identical to the bash version over 34 argument shapes except the unterminated last `.java` key line, which bash `while read` dropped (silent pass, now MISSING/ORDER); `tests/check-facets-card.bats` unchanged 6/6, mutations FC-order/FC-empty observed. Retro retros/2026-10-22-go-check-facets-card.md (1Δ, pending). [ev: retro go-check-facets-card Δ1]
