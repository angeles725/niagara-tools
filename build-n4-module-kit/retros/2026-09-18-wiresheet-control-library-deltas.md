<!-- review-status: folded -->
# 2026-09-18 · kit · wiresheet-control-library-deltas

**Session**: corpus-mining gap-audit for build-n4-module — wire-sheet authoring, control-library/alarms/PID, state-machine, transport types.
**Delta count**: 1

## What happened

A gap audit checked four operator-nominated themes against the 2026-09-18 retro set (16 files) and
the kit's `types/logic.md` to determine which are genuine kit gaps versus already covered.

**Themes checked and their disposition:**

| Theme | Disposition | Reason |
|---|---|---|
| Control library alarms — `BAlarmSourceExt`, offnormal algorithms | **Already covered** | `types/logic.md §Protection anatomy` (lines ~200-228) documents Pattern A + Pattern B with Java snippets, cites B827; LC5 §LC5 covers `BAlarmSourceExt → BAlarmService` chain; the 2026-09-06 `campaign9-doctrine-fold` retro folded both patterns into the kit |
| PID / `kitControl.BLoopPoint` — B733 | **Already covered** | `types/logic.md §LC2` documents BLoopPoint: `loopAction`, P/I/D, `BLoopAlarmAlgorithm`, recommended tuning formula; cites corpus B539; the `BNumericWritable → AO proxy` chain is also covered |
| State machine — startup/shutdown/alarm/lockout/fault sequences | **Already covered** | `types/logic.md` §805.9 SEVEN-layer flowchart template explicitly names "STATES (the phase machine: idle / running / fault / …)"; `BLatch` limitation note + "protection latch = author-built field + faultReset OPERATOR action"; "implement as a pure-class finite-state machine"; latch/faultReset pattern at line ~25. No squirrel-state-machine library exists in N4 — the corpus-backed pattern is exactly what the kit documents |
| Transport types beyond `TcpComm` / envCtrlDriver | **No corpus-backed gap** | No dedicated corpus block on authoring new protocol transport types at a kit-actionable level; B896 (SMS/serial transport, alarm routing SPI) is module-mechanics context, not an authoring recipe; the DDF transaction-manager choice rule was already proposed in `2026-09-18-module-mechanics-deltas.md §Δ6` (B909) |
| Wire-sheet live-view recipe — facets + BStatus color on pin rows | **GENUINE GAP** | See below |

**Wire-sheet gap rationale:**

B747 §747.2 establishes a concrete three-part mechanism: `SlotBarGlyph` visibility is gated on
`Flags.isSummary()` (`SlotBarGlyph.java:56`, `Flags.java:12` SUMMARY=8); `PropertyBarGlyph.
updateValueString()` paints each visible pin row with the slot's live value **formatted by its facets**
(units, precision) AND **tints the row by its `BStatus`** (fault/down/stale/override colors via
`getShowStatusColors()`, `PropertyBarGlyph.java:35-46`). Refresh is push-on-change:
`handleComponentEvent → WsController.handleComponentEvent → glyph.changed(slot)`.

The kit already has three separate authoring rules:
- `TRANSIENT|SUMMARY|READONLY` for computed outputs (flag)
- `getSlotFacets` projection of a `facets` config slot → units/precision (recipe for authoring)
- `propagateFlags BStatus` pattern (recipe for controlling status propagation)

What the kit does NOT have is the **causal chain** that ties these three choices to the wire-sheet
rendering: "SUMMARY → visible pin row; facets on that slot → formatted value displayed in the pin
row; BStatus on that slot → pin row tinted by status color." A builder setting SUMMARY + facets +
BStatus for three separate reasons does not know that these three choices together make the
wire-sheet a live debugging surface. The 2026-09-18 `organization-tags-grouping-deltas` retro noted
"B747 SUMMARY = wire-sheet port (one-liner in flags bullet)" as folded — but that one-liner is
absent from `types/logic.md`; and neither it nor any other 2026-09-18 retro proposed the
facets+status-color rendering recipe as a delta.

## Evidence

- `[ev: corpus B747 §747.2]` — `PropertyBarGlyph.updateValueString()` paints live values with
  facets + tints by `BStatus`; `SlotBarGlyph.java:56` gates pin visibility on `Flags.isSummary()`;
  `Flags.java:12` SUMMARY=8; `PinSlotsCommand.java:38` "pin a slot" sets the summary flag;
  `WsController.java:544-574` push-on-change refresh.
- `[ev: corpus B735]` — flags/SUMMARY mechanics (cross-ref in B747).
- `[ev: kit types/logic.md line ~155]` — flags bullet lists SUMMARY/OPERATOR but does not explain
  wire-sheet pin visibility.
- `[ev: kit types/logic.md line ~157]` — `getSlotFacets` projection rule present, no wire-sheet
  rendering context.
- `[ev: kit types/logic.md line ~185]` — `propagateFlags BStatus` recipe present, no wire-sheet
  rendering context.
- `[ev: retro 2026-09-18-organization-tags-grouping-deltas §folded]` — B747 "one-liner in flags
  bullet" noted as folded but absent from kit; full facets+status-color recipe not proposed.

## Proposed kit deltas (propose-never-apply)

| Δ | Delta | Target file / § | Token |
|---|---|---|---|
| Δ1 | Add §"Wire-sheet live-view recipe" (or an inlined callout in `§Tridium rt idioms` flags bullet): **three authoring choices determine the wire-sheet rendering** — (1) `SUMMARY` flag on a slot → `SlotBarGlyph` shows the slot as a pin row (`Flags.isSummary()`, `SlotBarGlyph.java:56`, SUMMARY=8); (2) units + precision facets on that slot → `PropertyBarGlyph.updateValueString()` renders the live value formatted in the pin row; (3) `BStatus` on the slot's value → pin row is tinted by status color (fault=red, stale=yellow, override=magenta) via `getShowStatusColors()`. Refresh is push-on-change (no poll). Practical rule: curate SUMMARY on real I/O (not internals); add units/precision facets to every numeric slot that appears as a pin; set `BStatus` on every output — do this and the wire-sheet becomes a live, color-coded flow debugger for free. | `types/logic.md` — expand the flags bullet or add a `§Wire-sheet live view` callout under `§Tridium rt idioms` | `wiresheet-live-view-recipe` |

## Lessons

1. **Three-part synergy**: the wire-sheet live view is not one feature — it is the intersection of
   three independent authoring decisions (SUMMARY flag, facets, BStatus). The kit documents each
   separately but never ties them together into a single recipe. A builder who knows all three rules
   still needs the causal chain to understand the wire-sheet payoff.
2. **"Folded" ≠ applied**: the organization-tags retro marked B747 as "folded (one-liner)" but the
   one-liner is not in `types/logic.md`. Future retros should distinguish "folded → kit already has
   it" from "proposed → kit does not have it yet."
3. **Corpus-index P2 "no" is stale for B732/B733**: those blocks' content is now in the kit via
   `§Protection anatomy` (B827/B732) and `§LC2` (B539/B733). The corpus-index P2 "no" rows were
   set in the 2026-09-03 audit and not updated after the doctrine-fold retros applied the content.

---
**Status**: PENDING — INDEX row appended: `| retros/2026-09-18-wiresheet-control-library-deltas.md | kit | 2026-09-18 | pending | 1 |`
