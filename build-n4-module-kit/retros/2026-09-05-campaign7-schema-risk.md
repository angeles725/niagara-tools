<!-- review-status: pending -->
# Retro: Campaign 7 PR5 — schema-risk.sh (B795/B799)

**Date**: 2026-09-05 | **PR**: feat/c7-schema-risk | **Issue**: #46

## What was built

`schema-risk.sh <before-dir> <after-dir>` — B795 §795.4 table-driven slot-change survival
classifier for Niagara N4 modules. Classifies each slot diff (add/remove/retype/reorder/rename/
unknown-kind) against the embedded CSV and returns the worst-cell verdict (OUTAGE/LOSSY/SAFE).
Exit codes 0/1/2 map to the verdict domain; 3=usage, 4=env sit above the verdict range so a bad
invocation never masquerades as OUTAGE in a deployment script (D4a deliberate deviation).

## TDD cycle

RED: cherry-picked `qa/c7-schema-risk` 6d27ff0 → 8 pins failing (script absent).  
GREEN: implemented schema-risk.sh (awk slot parser + diff engine + CSV lookup + worst-cell);  
added SR9 (unreadable module-include.xml → exit 4) + SR-CSV (heredoc byte-equal to oracle).  
All 10/10 green. Shellcheck 0 warnings.

## Named mutations (all reverted)

| Mutation | Affected test | Observation |
|----------|--------------|-------------|
| worst-cell → first-cell (`-gt` to `-lt`) | SR7 | mixed fixture reads SAFE instead of OUTAGE |
| kind_swap → unchanged (drop UNKNOWN fail-safe) | SR6 | unknown_kind falls through as no-change → SAFE |
| retyped → unchanged (retype detection off) | SR3 | retype_simple slot treated as unchanged → SAFE |
| is_rename=0 (rename detection off) | SR5 label pin | verdict stays LOSSY but no "rename" token |
| rename_slot,LOSSY → rename_slot,SAFE in CSV | SR-CSV | heredoc diverges from b795-795.4.csv oracle |

## Real-module smoke (read-only)

| Module | Commits compared | Verdict | Notes |
|--------|-----------------|---------|-------|
| CompPan-rt | HEAD~1 → HEAD (docs only) | SAFE | No slot changes in docs commit |
| CompPan-rt | HEAD~2 → HEAD~1 (HOA feature) | SAFE | HOA adds slots (add_slot = SAFE) |
| ColdRoomPan-rt | HEAD~1 → HEAD (docs only) | SAFE | No slot changes in docs commit |

## Diff algorithm limits (D4 L1-L3)

- **L1**: Multiple simultaneous renames (≥2 removed + ≥2 added) are not paired; they surface as
  separate remove_slot_unknown + add_slot rows. Verdict is unchanged (LOSSY-or-worse). Documented
  in script header.
- **L2**: remove_slot_complex (SAFE) and retype_complex (LOSSY) are unreachable by design. Every
  unresolved removal → remove_slot_unknown (LOSSY); every unresolved retype → retype_unknown (OUTAGE).
  Never downgrade on uncertainty (B795 §795.2).
- **L3**: Slot-kind swaps (property↔action at the same name) and unparseable annotations emit
  UNKNOWN → OUTAGE. No silent skips.

## Key lessons

1. **awk scalar-vs-array conflict**: In gawk, calling `length(arr)` on a variable that was never
   used as an array (no `arr[key] = val` assignments) treats it as a scalar. Then `for (key in arr)`
   fails with "attempt to use scalar as array". Fix: pre-declare all category arrays with `delete arr`
   in BEGIN, and track counts with explicit counters rather than `length()`.
2. **Verbatim CSV guard**: Embedding the CSV as a heredoc and diffing it against an oracle file
   (SR-CSV) catches accidental editing of the table. The heredoc terminator line `CSV` must be on its
   own line; awk extraction `/^CSV_TABLE=\$\(cat <</{f=1;next} f&&/^CSV$/{exit} f{print}` is reliable.
3. **Exit-code domain design**: Verdict exits (0/1/2) and usage/env exits (3/4) must be distinct
   ranges. If usage exit ≤ verdict max, a bad invocation masquerades as OUTAGE in deploy scripts
   checking `exit 2`. The deliberate deviation from the usual `2=usage/3=env` convention is worth
   documenting explicitly (D4a).
4. **Real-module verdicts**: Both CompPan-rt and ColdRoomPan-rt returned SAFE across the recent
   commits (docs-only and add-slot-only changes). The classifier correctly identifies add_slot as SAFE.

## Post-merge fix: parse_slots whitespace-around-= (2026-09-05)

**Defect**: `extract_attr` matched `name="x"` but not `name = "x"` (space around `=`).
Real modules (CompPan-rt, ColdRoomPan-rt, fixtures/MinimalPan) use the spaced form.
Result: `parse_slots` returned 0 slots on every real-module snapshot, producing a
false `verdict=SAFE` with no rows — the worst possible failure for a pre-deploy guard.

**Root cause**: The regex was written for the compact fixture form used in SR1-SR7
(`@NiagaraProperty(name="setpoint",type="double")`). Real Niagara sources generate
multi-line annotations with spaces (`name = "setpoint"`) per Java annotation style.

**Fix**: Regex `attr "[[:space:]]*=[[:space:]]*\"[^\"]*\""`, value extracted with
`index()` to find the first `"` inside the match span (no brittle offset math).

**Lesson**: Fixtures must use the shape real modules write. A parser test on a
synthetic compact form can be green while the tool is blind to production annotations.
Verify any new parser with at least one real-module smoke before merging.

**SR10**: Added real-shape fixture pair (MinimalPan production form, spaced multi-line
annotations). Named mutation: revert to unspaced regex → 0 rows → false SAFE → SR10 bites.
