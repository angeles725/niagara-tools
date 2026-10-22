<!-- review-status: folded -->
# 2026-10-22 · kit · r8-t9e-status-links

**Session**: niagara-research quality round 8, task T9e unit A (register WU-P0-2: C04, C05, C08), branch `docs/r8-t9e-status-links` from origin/main c9d1215
**Delta count**: 2

## What happened
Folding register items C04, C05 and C08 into `types/logic.md` showed two ways a kit card drifts from the source it
transcribes. The BStatus bit table had been copied from the right corpus block (B736) with every value from STALE on
shifted by one bit and ALARM missing, and nothing compared the table to the constants. The link section said "the
framework calls `doCheckLink` before any link is created … programmatically", a claim read from the method's role, not
from its callers: `linkTo`, `add(name, BLink)` and `makeLink` never reach `checkLink`. This unit corrects both cards and
adds `tests/kit-api-pins.bats` (KP1 pins the table to the constants, KP2 keeps the non-existent `BStatus.and()` out).

## Evidence
- Constants DISABLED 0x0001 … UNACKED_ALARM 0x0080 at `organized/docSource/docSource-doc/extracted/baja/javax/baja/status/BStatus.java:702-716`; kit table had STALE 0x08, OVERRIDDEN 0x10, NULL 0x20, no ALARM `[ev: corpus B1314 §1314.2]`
- `tests/kit-api-pins.bats` RED on c9d1215: KP1 not ok (doc STALE 0x08 vs source 0x10), KP2 not ok (`types/logic.md` `getPropagateFlags().and(`); GREEN 2/2 after the fold `[ev: tests/kit-api-pins.bats]`
- `checkLink` callers: only tools (BOX session handler, Workbench); `linkTo` adds TRANSIENT and activates without it (`BComponent.java:1776-1782`); `ComponentSlotMap.java:1684-1685` runs only a user-permission check `[ev: corpus B1309 §1309.2]` `[ev: corpus B1316 §1316.3]`

## Proposed kit deltas (propose-never-apply)
| Δ | Delta | Target file / § | Token |
|---|---|---|---|
| Δ1 | A table transcribed from source constants (bit masks, enum ordinals) carries a bats pin that compares it to those constants; a transcription error otherwise ships silently | `types/logic.md` § Slots — BStatus producers; `tests/kit-api-pins.bats` | `[ev: corpus B1314 §1314.2]` |
| Δ2 | A kit sentence "the framework calls X when Y" is checked against X's callers, not its javadoc role; `doCheckLink` is reached only through `checkLink`, which only tools call | `types/logic.md` § Link lifecycle — gating and reacting | `[ev: corpus B1316 §1316.3]` |

## Lessons
- A correct corpus block does not protect the kit card copied from it; pin the card, not the block.
- "Called by the framework" is a caller-trace claim; grep the callers before writing it.

---
**Status**: FOLDED (T9e unit A, same unit) — Δ1 `types/logic.md` § Slots — BStatus producers + `tests/kit-api-pins.bats` KP1/KP2, Δ2 `types/logic.md` § Link lifecycle — gating and reacting.
