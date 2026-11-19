<!-- ledger: kit -->
<!-- date: 2026-11-19 -->
<!-- last-change-note: docs/ is not an artifact; report FAIL restores the mirror jar -->
log: 2026-11-19 · build.sh and report-module share one artifact predicate (a docs/ folder is no artifact: no schema-risk ERROR or INCOMPLETE), and a report-module FAIL restores the pre-build mirror jar so a same-version fix is not drift. [ev: retro build-docs-not-artifact Δ1] [ev: retro build-docs-not-artifact Δ2]
