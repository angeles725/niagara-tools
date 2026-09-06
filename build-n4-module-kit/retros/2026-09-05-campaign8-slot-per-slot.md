<!-- review-status: folded -->
# Retro: Campaign 8 — slot-coverage.sh per-slot mode + empty-lexicon FAIL escalation (PR5)

**Date:** 2026-09-05
**Change:** build-n4-module-campaign8 PR5 — `feat/c8-slot-per-slot`
**Scope:** `build-n4-module-kit/toolbelt/slot-coverage.sh` — per-slot subcommand + D6a empty-lexicon escalation
**Lead:** Campaign 8 PR5 executor

[ev: corpus B788] [ev: corpus B759] [ev: corpus B780]

---

## What happened

Two additive changes to `slot-coverage.sh`:

1. **`per-slot` subcommand (D6):** `slot-coverage.sh per-slot <module-include.xml> <module.lexicon> <src-dir>` compares `OPERATOR`-flagged `@NiagaraProperty` slots against lexicon keys. Uses a paren-balance multi-line awk state machine (same technique as `lint-delays.sh` Pass 1) to handle multi-line annotations. Dot-dirs pruned via `-type d -name '.*' -prune` (D9b). Emits `pct=<n.n> (per-slot)`, `MISSING <slot>` (no lexicon key), `STALE <key>` (key matches no known annotation, type name, or enum tag — truly dead). Exits 1 on any MISSING, 0 when clean.

2. **Empty-lexicon FAIL escalation (D6a):** empty `module.lexicon` with `>=1` declared type now exits 1 (FAIL) unconditionally instead of exit 0/WARN-only (or exit 1 only with `--strict`). The T8 footgun (`chihuahua`) passed the aggregate before this change despite every slot rendering raw camelCase.

**RED-vs-design deviation (recorded per K13):** D6 text says "required = every @NiagaraProperty". The RED (SP1/SP3 pins at commit ab194a5) gates on `Flags.OPERATOR` slots only — a non-OPERATOR slot is NOT flagged as MISSING. Implementation follows the RED; D6 text is secondary to the authoritative test pins.

---

## Evidence

### TDD cycle

| Phase | Evidence |
|-------|----------|
| RED   | Commit `ab194a5`: SC6-parse, SP1, SP2, SP3, SP4 fail; SC1–SC6/dup-keys green |
| GREEN | `slot-coverage.sh` extended; 214/214 bats green (SC6-parse, SP1–SP4 all pass) |
| REFACTOR | shellcheck exit 0; SC2016 disabled inline for awk single-quoted program |

### Named mutations (each on `mktemp` copy, flip observed)

| Mutation | Change | Observed flip |
|----------|--------|---------------|
| (a) drop required slot key | Added `setpoint=Consigna` to fixture lexicon (tmp copy) | `pct=100.0 (per-slot)` · no MISSING row · exit 0; SP1 would FAIL (no "MISSING"+"setpoint") |
| (b) revert empty-lexicon to exit 0 | Changed `EMPTY_LEX_FAIL=1` → `EMPTY_LEX_FAIL=0` in tmp copy | chihuahua exits 0 instead of 1; SC6-parse `[ "$status" -eq 1 ]` FAILS |
| (c) remove stale detection | Commented out `printf 'STALE %s\n'` in tmp copy | No STALE row; SP2 `[[ "$output" == *"STALE"* ]]` FAILS |
| (d) remove dot-dir prune | Dropped `-type d -name '.*' -prune -o` from find | `staleKnob` from `.deploy-baseline/` appears as MISSING; SP4 `[[ "$output" != *"staleKnob"* ]]` FAILS |

### Real smokes (verbatim outputs)

#### Smoke 1: ColdRoomPan-rt per-slot

```
$ slot-coverage.sh per-slot ColdRoomPan-rt/module-include.xml ColdRoomPan-rt/module.lexicon ColdRoomPan-rt/src
pct=10.0 (per-slot)
MISSING fanMode
MISSING fanRunMode
MISSING freezeDiffRestart
MISSING freezeDiffStop
MISSING freezeProtect
MISSING freezeSetpoint
MISSING powerOnDelay
MISSING resistanceMode
MISSING valveMode
Exit: 1
```

**Real count: 9 MISSING / 0 STALE** (design estimated 19 — see deviation below). Zero STALE because the known-keys set (fix 2: all slots ∪ type names ∪ @Range tags) now covers all legitimate lexicon entries.

#### Smoke 2: chihuahua parse mode (empty lexicon — SC6-parse shape)

```
$ slot-coverage.sh chihuahua-rt/module-include.xml chihuahua-rt/module.lexicon
slot-coverage: FAIL empty lexicon with 8 declared type(s)
pct=0.0 (type-set)
missing=ChiCarcamo,ChiCarcamoMonitor,ChiDashboardService,ChiDatalogger,ChiDataloggerMonitor,ChiUp,ChiUpMonitor,Planta
extra=
Exit: 1
```

Exits 1 with FAIL row (was exit 0/WARN before D6a).

#### Smoke 3: CompPan-rt per-slot

```
$ slot-coverage.sh per-slot CompPan-rt/module-include.xml CompPan-rt/module.lexicon CompPan-rt/src
pct=100.0 (per-slot)
Exit: 0
```

CompPan-rt has all OPERATOR slots covered (pct=100.0), exits 0. Zero STALE: every lexicon key matches a known slot, type name, or enum tag.

---

## Proposed kit deltas

| Delta | File | Description |
|-------|------|-------------|
| D6a-behaviour-doc | `types/logic-authoring.md` | Document the STALE known-keys set (slots ∪ type names ∪ @Range tags) and recommend using MISSING list for module fix tracking |
| Per-slot workflow | `BUILD-LOOP.md` §5 | Already updated: per-slot mode listed beside parse mode; `[ev: retro campaign8-slot-per-slot]` |

---

## Lessons

1. The paren-balance awk technique from `lint-delays.sh` is reusable for any multi-line Java annotation: accumulate into a buffer, count `(` and `)`, emit when `depth <= 0`.
2. STALE must compare lexicon keys against ALL `@NiagaraProperty` annotations (any flags), not just OPERATOR ones. A lexicon key that translates a READONLY or SUMMARY slot is live context in the operator view, not a dead translation. Using OPERATOR-only as the stale reference inflated counts from legitimate non-OPERATOR keys (ColdRoomPan: 28→6; CompPan: 35→1 after fix). SP5 pin proves the correct behavior; the defect was caught in lead review and fixed before merge.
3. Design estimates for missing slot counts ("~19") were off (real: 9) because only `BEvaporatorUnit` has OPERATOR slots; the design likely counted all `@NiagaraProperty` slots without gating on OPERATOR.
4. D6a: gating empty-lexicon FAIL on `--strict` only defeats the purpose when the motivating module (`chihuahua`) is always run without `--strict` in the normal workflow.
5. The STALE known-keys set must include type display-name keys (`module-include.xml <type name="X">`) and frozen-enum tag keys (`@Range("tag")` annotations) — both are live translations by Niagara convention, not dead. Omitting them inflated STALE with legitimate entries (ColdRoomPan: 6 fake-STALE cleared; CompPan: 1 cleared). SP6 pin proves the correct behavior; the defect was caught in lead review and fixed before merge.
