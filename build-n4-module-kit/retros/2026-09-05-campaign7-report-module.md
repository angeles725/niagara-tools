<!-- review-status: pending -->
# Retro: campaign7-report-module — report-module.sh aggregated conformance report

**Session**: Campaign 7 PR8 (stretch) · 2026-09-05  
**Change**: toolbelt/report-module.sh, BUILD-LOOP.md §5, skill/SKILL.md v0.7  
**Delta count**: 3 proposed deltas  

## Context

PR8 closes Campaign 7 by shipping `report-module.sh`, an aggregated read-only conformance
report that composes the campaign-6 toolbelt (verify-module --src, slot-coverage parse + dup-keys,
lint-timers, --plano) over every profile artifact under a module root. Grounded in the B798
baseline run on ColdRoomPan-rt (9 PASS · 1 FAIL · 1 WARN · 1 SKIP, exit 1).

## Proven Lessons

### Delta 1 — aggregate-first composition: one command to see all conformance FAILs

**Observation**: before report-module.sh, a developer had to run 4+ tools separately to know if
a module was ready for hand-off; a missed lint-timers run meant a timer leak shipped silently.

**Proposed addition to METHODOLOGY.md or BUILD-LOOP.md §5:**
> After all individual pre-gate checks, run `toolbelt/report-module.sh <module-root>` as the final
> gate step. Exit 0 = CLEAN (safe to hand off); exit 1 = ISSUES (at least one FAIL to resolve
> before hand-off). This prevents partial-gate hand-offs where a developer ran verify-module but
> skipped lint-timers or slot-coverage.

### Delta 2 — row-level aggregation is required when member exits ≠ member WARNs

**Observation**: slot-coverage exits 0 even when coverage < 100% (WARN, not FAIL). An exit-code-only
aggregate would mis-classify WARN-only runs as CLEAN. Row-level parsing is the primary signal; member
exit codes are a secondary, ORed guard for crashed tools.

**Proposed note in build-verify.md or METHODOLOGY.md:**
> When composing toolbelt scripts, prefer row-level output parsing over exit-code-only aggregation.
> A tool that exits 0 may still print WARN rows that the operator must see. Only row parsing produces
> the correct PASS/FAIL/WARN/SKIP split for aggregated reports.

### Delta 3 — dup-keys is a count, not a relay; 0 = PASS, N > 0 = FAIL

**Observation**: slot-coverage emits per-key WARN lines; the aggregated report collapses them into
a single `dup-keys  N` row (PASS when N=0, FAIL when N>0). This is cleaner than relaying N WARN
lines per duplicate key in a multi-artifact report.

**Proposed convention in toolbelt authoring guide:**
> When relaying dup-count checks from a composed tool, emit a single count row (PASS N=0 / FAIL N>0)
> rather than relaying each WARN line. Keeps the aggregated report readable for multi-artifact modules.

## Not Proposed

- Per-key dup-key FAIL rows in the aggregated report: the count row is sufficient and cleaner.
- Changing slot-coverage's own WARN lines: those are correct for its own use; only the aggregate
  collapses them.
