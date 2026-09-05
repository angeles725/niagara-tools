# campaign6 — math-model specs MM2 + MM3 (implementation contracts)

**Author**: investigador1 (Opus 4.8). **Status**: SPEC (uncommitted, referenced by path — like MM1). Feeds QA (RED pre-stage) + the SDD tools-lane. I do NOT implement in niagara-tools; this is the contract the sdd-apply writer builds to and QA pins.
**Companion**: MM1 (gate coverage %) contract already delivered to QA (RED d7e52a8, 8 pins). MM2/MM3 follow the same discipline: pure function, isolated from I/O, deterministic vectors, mutations that MUST flip a pinned value, 0-denominator defined (never a false 100).

---

## MM2 — exposed-set coverage (palette / lexicon / type-registration)

**Why**: three documented silent-deploy footguns are the same set-difference — empty `module.palette` passes the whole gate yet nothing drags (retro module-palette · B5), a missing `module.lexicon` key renders raw camelCase (T8), a dangling `module-include.xml` `<type>` surfaces live as `Missing class` (B12). One metric over the exposed `@NiagaraType` set catches all three before deploy.

**Pure function** (isolated from file parsing):
```
set_coverage(declared: set[str], required: set[str]) -> (pct, missing: set[str], extra: set[str])
    present = required & declared
    pct     = round(100 * len(present) / len(required), 1)   if len(required) > 0
            = "N/A"                                            if len(required) == 0
    missing = required - declared        # the coverage gap (the footgun)
    extra   = declared - required        # dangling declarations (separate lint, NOT counted against pct)
```
- **`required`** = the authoritative exposed set (the `@NiagaraType`/slot set). **`declared`** = what the artifact lists (`<p t="mod:Type">` for palette, `key=` for lexicon, `<type>` for type-registration). Callers build the two sets; the math is one shared function across all three applications.
- **0-required → "N/A", NOT 100** (same anti-false-confidence rule as MM1): a module exposing zero types is a scaffold, not "fully covered" — reporting 100 would hide the empty-palette footgun this metric exists to catch.
- **`extra` is reported, not scored**: a palette entry for a non-exposed type is a dangling-declaration lint (its own signal), not a coverage miss — keep it out of `pct` so the two defects don't mask each other.

**Pin vectors** (QA):
| declared | required | pct | missing | extra |
|---|---|---|---|---|
| {A,B,C,D} | {A,B,C,D} | 100.0 | ∅ | ∅ |
| {A,B} | {A,B,C,D} | 50.0 | {C,D} | ∅ |
| {A,B,C,D,X} | {A,B,C,D} | 100.0 | ∅ | {X} |
| ∅ | {A} | 0.0 | {A} | ∅ |  ← empty palette, types exist = the B5 footgun as 0%
| ∅ | ∅ | "N/A" | ∅ | ∅ |  ← scaffold module, zero exposed types

**Mutations that MUST flip a pinned value:**
- denominator = `len(declared)` instead of `len(required)`: {A,B} vs {A,B,C,D} → 100.0 ≠ 50.0.
- `extra` folded into numerator: {A,B,C,D,X} vs {A,B,C,D} → 125.0 (impossible) ≠ 100.0.
- 0/0 → 100 instead of "N/A": flips the last row.
- `missing` computed as `declared - required`: swaps missing/extra on row 2/3.

---

## MM3 — schema-change survival risk classifier

**Why**: the saved-data survival matrix (retros slot-type-change · S1, corpus-index · S2, coldroompan-fan-mode · S3; consolidated at METHODOLOGY.md §"Schema / upgrade safety" lines 24-29 + block B754) is a decision table over a slot diff. A `.bog` binds to the class BY NAME, ungated — so a deterministic risk class computed from two module snapshots BEFORE deploy prevents the `ClassCastException` / "Cannot load station" boot-loop that took a live refrigeration rack down.

**Classification table** (verbatim from the matrix — do NOT paraphrase in impl):
```
SAFE   (boots, data kept)          : add_slot, reorder_slot, change_default, change_flags, change_facets, add_enum_tag
LOSSY  (boots, data dropped +warn) : remove_slot, rename_slot
OUTAGE (won't boot)                : retype_frozen_simple_slot, remove_enum_tag, rename_enum_tag
```

**Pure function**:
```
SEVERITY_ORDER = SAFE(0) < LOSSY(1) < OUTAGE(2)
schema_risk(changes: list[change_kind]) -> risk
    = max(CHANGE_SEVERITY[c] for c in changes)   if changes non-empty
    = SAFE                                         if changes empty   # no schema change = safe
    # an UNKNOWN change_kind must NOT map to SAFE — map to OUTAGE (fail-safe: escalate the unclassified)
```
- **Aggregation = MAX severity** (the worst change decides the deploy). A change set with one OUTAGE is an OUTAGE regardless of how many SAFE changes accompany it.
- **Empty → SAFE** (nothing changed). **Unknown kind → OUTAGE**, never SAFE — fail-safe posture consistent with the kit's "new safety slot defaults SAFE" doctrine (soft-start · L15): an unclassified schema change is treated as the worst case until proven otherwise.

**Pin vectors** (QA):
| changes | risk |
|---|---|
| [] | SAFE |
| [add_slot, reorder_slot] | SAFE |
| [change_flags, add_enum_tag] | SAFE |
| [add_slot, remove_slot] | LOSSY |
| [remove_slot, rename_slot] | LOSSY |
| [add_slot, remove_slot, retype_frozen_simple_slot] | OUTAGE |
| [remove_enum_tag] | OUTAGE |
| [some_unrecognized_change] | OUTAGE |  ← fail-safe on unknown

**Mutations that MUST flip a pinned value:**
- `remove_slot` misclassified SAFE: [add_slot, remove_slot] → SAFE ≠ LOSSY.
- aggregation MIN instead of MAX: [add_slot, remove_slot, retype_frozen_simple_slot] → SAFE ≠ OUTAGE.
- empty → OUTAGE (wrong default): [] → OUTAGE ≠ SAFE.
- unknown → SAFE (drops the fail-safe): [some_unrecognized_change] → SAFE ≠ OUTAGE.

---

## Integration notes for the SDD tools-lane

- MM2 lands as `toolbelt/slot-coverage.sh` (already in the lead's merge order); MM3 as a new `toolbelt/schema-risk.sh` (a pre-deploy guard, sits beside `verify-module.sh`). Both keep the pure math EXTRACTABLE (a `coverage()`/`risk()` function callable with plain args, not buried in a parse pipeline) so QA pins exact values — same rule that made MM1 biteable.
- The FILE-PARSING half (build the `declared`/`required` sets from `module.palette`/`module.lexicon`/`module-include.xml`; build the `changes` list from a two-snapshot slot diff) is separate and tested with fixtures; the SCORING half is the pure function pinned above.
- Prototype order stands: MM1 (staged) → MM2 → MM3. MM4 (assurance-stack %) / MM5 (risk-weighted checklist) specs follow on request.
