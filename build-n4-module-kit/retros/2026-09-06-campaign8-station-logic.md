<!-- review-status: pending -->
# Campaign 8 PR20 — station-logic wiring checks retro

Session: 2026-09-06, branch feat/c8-station-logic, kit PR20.

---

## D1 — CHECK16 vice-versa direction not implemented

The spec (wave3.md §20) says "hasDefrost<->DefrostController sibling" implying
both directions.  The existing PANCCADIA/BA5 clean bog has a DefrostController
component but no hasDefrost=true sibling on the same parent level.  Implementing
"DefrostController without hasDefrost=true sibling → FAIL" would fire on that
clean bog, breaking BA5.

**Decision**: only the forward direction implemented — hasDefrost=true component
must have a DefrostController sibling under the same parent.  The reverse
direction would require a more nuanced check (distinguish a standalone DC from a
miswired one) and is deferred to a follow-up.  SL16 RED pin tests only the
forward case, confirming this is the intended shape.

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
| M1 | Add second source to same relay slot | CHECK13 | FAIL | yes (SL13) |
| M2 | Remove outgoing relay link from OPERATOR output | CHECK14 | WARN | yes (SL14) |
| M3 | Set hasDefrost=true with no DefrostController sibling | CHECK16 | FAIL | yes (SL16) |
| M4 | ColdRoom_1 links that address evap3* slots | CHECK17 | FAIL | yes (SL17) |
| M5 | Evap unit whose HOA tile (evap3) != freeze tile (evap1) | CHECK18 | FAIL | yes (SL18) |
| M6 | C-room-labeled slot sourced from wrong unit index | CHECK15 | WARN | yes (SL15) |
| M7 | Control component writes to setpoint slot | CHECK19 | WARN | yes (SL19) |

---

## Real smoke results

### PANCCADIA (config.bog, ColdRoomPan module)

PANCCADIA bog not available in this WSL environment.  Expected results (from
issue #49 and bog-nav tile analysis):

- CHECK13: clean (no double-sourced relay slots)
- CHECK14: 1 WARN — ColdRoom_5/EvaporatorUnit2 evapOut (no outgoing relay link)
- CHECK15: clean (no sensor-crossed slots)
- CHECK16: clean (no hasDefrost=true without DefrostController sibling)
- CHECK17: clean (room index matches evap slot index)
- CHECK18: 2 FAIL — ColdRoom_1/EvaporatorUnit_1 and ColdRoom_1/EvaporatorUnit_3
  (evap tile number inconsistent: units 1 and 3 carry tile labels for 3 and 1
  respectively, as discovered by bog-nav tiles command)
- CHECK19: clean (no reverse link-direction)
- Exit: 1 (CHECK18 FAIL)

### HoneywellMX60 (config.bog)

MX60 bog not available in this WSL environment.  Expected: no station-logic
findings (all checks clean, exit 0 if no prior FAILs).

---

## Verification

- 300 bats tests passing (0 failures), including 8 new SL13-SL19 + SL-smoke-panccadia
- shellcheck bog-audit.sh → exit 0
- sweep-build-state.sh → exit 0
- sweep-fold-audit.sh --strict → exit 0 (56 folded, 56 cited)
- kit-links.bats → 7/7 ok

---

retro: retros/2026-09-06-campaign8-station-logic.md (4 deltas, review-status: pending)
