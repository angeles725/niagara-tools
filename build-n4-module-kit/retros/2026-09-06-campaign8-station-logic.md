<!-- review-status: pending -->
# Campaign 8 PR20 — station-logic wiring checks retro

Session: 2026-09-06, branch feat/c8-station-logic, kit PR20.

---

## D1 — CHECK16 both directions implemented; clean fixture updated

The spec (wave3.md §20) says "hasDefrost<->DefrostController sibling" — both
directions are now enforced.  Initial retro stated "only forward" — corrected
in the second session after coordinator flagged the missing reverse.

Reverse direction: a DefrostController whose parent has no unit with
`hasDefrost=true` → FAIL.  The BA5 clean fixture was updated to include a
`<p n='Evap1' h='e1' t='CRP:EvaporatorUnit'><p n="hasDefrost" t="b:boolean"
v="true"/></p>` sibling alongside the DefrostController, making it a legal
station (both present → no FAIL).

---

## D2 — Self-closing components not registered in handle_map (SL13 fixture bug)

Initial SL13 fixture had `SrcA h='s1'` and `SrcB h='s2'` as self-closing XML
elements (`/>`).  The bog-audit.sh parser registers a component in handle_map
only when it encounters a non-self-closing `<p h='...'>` opening tag.
Self-closing tags produce no handle_map entry → module_counts stays empty →
CHECK1 exits 3 (no own-module components found), not 1.

Fix: made SrcA/SrcB non-self-closing by adding a dummy `<p n="evapOut" .../>` child.
This is also the correct shape for a real Niagara component (slots are children).

**Lesson**: any fixture whose CHECK under test is not CHECK1 must have at least
one non-self-closing own-module component so the module-count guard is satisfied.

---

## D3 — Clean fixture exits with CHECK11/CHECK14 due to relay design

The first iteration of station-logic-CHECK13-clean.bog had a relay target that
lacked a fallback value — CHECK11 (proxy-link-safety) fired.  Also, the
`src_slot` in the link did not match the slot name in the Comp object, so
CHECK14 (own-output-unlinked) fired.

Fix: redesigned the clean fixture to include a fallback on the relay target, and
used consistent `src_slot='evapOut'` matching the slot name in handle_map.

---

## D4 — Python f-string !r conversion with conditional not valid

First attempt used `f'... under {_ppath!r or "(root)"}'`.  The `!r` conversion
specifier cannot be combined with an `or` expression inside an f-string
conversion field.  Python raises SyntaxError.

Fix: extracted the conditional into `_plabel = repr(_ppath) if _ppath else '(root)'`
and referenced `{_plabel}` in the f-string.

---

## Named mutations

| ID | Mutation | Check | Expected | Verified |
|----|----------|-------|----------|---------|
| M1 | Add second source to same relay slot (BooleanWritable only) | CHECK13 | FAIL | yes (SL13) |
| M2 | Remove outgoing relay link from type-inferred *Out output | CHECK14 | WARN | yes (SL14) |
| M3 | Set hasDefrost=true with no DefrostController sibling | CHECK16 | FAIL | yes (SL16) |
| M3r | DefrostController without hasDefrost=true unit sibling | CHECK16 | FAIL | yes (SL16r) |
| M4 | ColdRoom_1 links that address evap3* slots | CHECK17 | FAIL | yes (SL17) |
| M5 | Evap unit whose HOA tile (evap3) != freeze tile (evap1) | CHECK18 | FAIL | yes (SL18) |
| M6 | C-room-labeled slot sourced from wrong unit index | CHECK15 | WARN | yes (SL15) |
| M7 | Panel writes state slot into control container | CHECK19 | WARN | yes (SL19) |

---

## D5 — CHECK14 misses TRANSIENT output slots (bog-only approach)

The original CHECK14 only checked bog-stored slots (slots with the OPERATOR `o`
flag).  EvaporatorUnit's `evapOut` has `Flags.TRANSIENT | SUMMARY | READONLY`
— it is never written to disk.  So the bog has no record of `evapOut` on
EvaporatorUnit2, and 47 false WARNs on CONFIG inputs (valveMode, fanMode, etc.)
were replaced by 0 WARNs instead of the expected 1.

**Fix**: type-inferred detection.  After building `_used_outputs` from link_list,
a second pass builds `_type_out_slots[type_]` — the set of `*Out`/`condenser[N]`
slot names that any own-module component of type `type_` exports as a link source
within the station.  This lets the check detect that EvaporatorUnit2 (same type as
EU1 which does link `evapOut`) is missing its link.  The defrost suppression was
also extended from `r'defrost'` to `r'(defrost|resistance)'` to prevent false
WARNs for `resistanceOut` on units without `hasDefrost=true`.

---

## D6 — CHECK19 false positive on terminateOnResistanceTemp

`_STATE_SLOT_RE` originally included `Temp` as a suffix to catch state
temperature readbacks.  `terminateOnResistanceTemp` (a defrost-termination
setpoint threshold flowing from panel→control) ends in `Temp` and triggered
a false WARN.  Panel→control config setpoints are the EXPECTED direction (R20.8).

**Fix**: removed `Temp` from `_STATE_SLOT_RE`.  The rule now uses
`r'(State|Out|status)$'`.  Actual temperature state readbacks (`coilTemp`,
`evapTemp`) flow sensor→control or control→panel, never panel→control.

---

## Real smoke results

Bogs available at:
- `PANCCADIA`: `/mnt/c/Users/equipo/Niagara4.14/OptimizerSupervisor/stations/PANCCADIA/config.bog`
- `MX60`: `/mnt/c/Users/equipo/Niagara4.14/OptimizerSupervisor/stations/HoneywellMX605132026/config.bog`

### PANCCADIA (--module ColdRoomPan CompPan DashboardPan, parse 0.216s)

```
CHECK11  FAIL  Drivers/NrioNetwork/io34_5_2/points/ro2   own-module h:44b7d.evapOut -> writable with no explicit fallback
CHECK11  FAIL  Drivers/NrioNetwork/io34_5_2/points/ro3   own-module h:44b7e.evapOut -> writable ...
CHECK11  FAIL  Drivers/NrioNetwork/io34_5_2/points/ro4   own-module h:44b7c.valveOut -> writable ...
CHECK11  FAIL  Drivers/NrioNetwork/io34_5_2/points/ro5   own-module h:44b7d.valveOut -> writable ...
CHECK11  FAIL  Drivers/NrioNetwork/io34_5_2/points/ro6   own-module h:44b7e.valveOut -> writable ...
CHECK11  FAIL  Drivers/NrioNetwork/io34_5_2/points/ro7   own-module h:44b80.evapOut -> writable ...
CHECK11  FAIL  Drivers/NrioNetwork/io34_5_2/points/ro8   own-module h:44b80.valveOut -> writable ...
CHECK11  FAIL  Drivers/NrioNetwork/io34_5_2/points/ro9   own-module h:44b86.evapOut -> writable ...
CHECK11  FAIL  Drivers/NrioNetwork/io34_5_2/points/ro10  own-module h:44b86.valveOut -> writable ...
CHECK11  FAIL  Drivers/NrioNetwork/io34_1_6/points/ro2   own-module h:44b83.evapOut -> writable ...
CHECK11  FAIL  Drivers/NrioNetwork/io34_1_6/points/ro3   own-module h:44b82.valveOut -> writable ...
CHECK11  FAIL  Drivers/NrioNetwork/io34_1_6/points/ro4   own-module h:44b83.valveOut -> writable ...
CHECK11  FAIL  Drivers/NrioNetwork/io34_1_6/points/ro5   own-module h:44b82.resistanceOut -> writable ...
CHECK11  FAIL  Drivers/NrioNetwork/io34_1_6/points/ro6   own-module h:44b83.resistanceOut -> writable ...
CHECK11  FAIL  Drivers/NrioNetwork/io34_4_3/points/ro4   own-module h:45366.evapOut -> writable ...
CHECK11  FAIL  Drivers/NrioNetwork/io34_4_3/points/ro5   own-module h:45366.valveOut -> writable ...
CHECK11  FAIL  Drivers/NrioNetwork/io34_4_3/points/ro6   own-module h:45367.valveOut -> writable ...
CHECK14  WARN  Programacion/ColdRoom_5/EvaporatorUnit2   output slot 'evapOut' has no outgoing relay link (own-output-unlinked)
CHECK18  FAIL  Programacion/ColdRoom_1/EvaporatorUnit_1  link tile numbers 1, 3 disagree across HOA/state/freeze links (tile-number mismatch)
CHECK18  FAIL  Programacion/ColdRoom_1/EvaporatorUnit_3  link tile numbers 1, 3 disagree across HOA/state/freeze links (tile-number mismatch)
```

Per-check summary: CHECK11=17 FAIL, CHECK13=0, CHECK14=1 WARN, CHECK15=0,
CHECK16=0, CHECK17=0, CHECK18=2 FAIL, CHECK19=0.

### MX60 chihuahua (--module chihuahua, parse 1.824s)

No CHECK13-CHECK19 rows.  All new checks: 0.

---

## Verification

- 33 bats tests passing (0 failures), including SL13-SL19, SL-smoke-panccadia (exact counts), SL-smoke-mx60
- SL-smoke-panccadia: 9/9 assertions pass — CHECK11=17, CHECK13=0, CHECK14=1 (EvaporatorUnit2/evapOut), CHECK15=0, CHECK16=0, CHECK17=0, CHECK18=2 (EU_1 and EU_3), CHECK19=0
- SL-smoke-mx60: CHECK13-19 all 0, parse ~1.8s

---

retro: retros/2026-09-06-campaign8-station-logic.md (6 deltas, review-status: pending)
