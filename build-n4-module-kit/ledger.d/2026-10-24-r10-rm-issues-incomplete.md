<!-- ledger: kit -->
<!-- date: 2026-10-24 -->
<!-- last-change-note: quality round 10: report-module shows a skipped floor check beside a FAIL -->
log: 2026-10-24 · quality round 10 → `report-module` reads a FAIL together with a skipped floor check (verify-module, schema-risk) as `ISSUES (+INCOMPLETE)` instead of plain ISSUES, exit unchanged (FAIL 1 over INCOMPLETE 4, env 3 over both; `--lint-only` still drops the floor), so clearing the FAILs never reveals the skipped floor late. `[ev: retro r10-client-gate-retro Δ4]`
