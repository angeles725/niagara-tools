<!-- review-status: folded -->
# Campaign 8 — RT control doctrine fold (PR15)

**Module:** kit
**Date:** 2026-09-05
**Type:** PROMOTION — folds corpus B804 B805 B808 B822 B823 B825 B826 B828

## What happened

PR15 lands the RT-control doctrine from Campaign 8 probes into two kit type files.
`types/logic.md` gains a new §RT-control-logic section; `types/logic-authoring.md`
gains §"Logging a point to history" (B804) and §"Slot types for externally written
values" (B823/B822/B825/B826/B828); `types/dashboard.md` gains a one-line pointer to
the new logic-authoring section.

## Evidence — grep counts + anchors

**grep-before-fold** (`rg 'PID.*anti-windup|errorSum|BHistoryExt|BLatch.*edge' build-n4-module-kit/types/`):
- `types/logic.md`: 1 hit (existing LC2 BLoopPoint row, corpus B539 — different section, not B805 material)
- `types/logic-authoring.md`: 0 hits
- `types/dashboard.md`: 0 hits
- `grep -cE 'slot type.*external|externally written|wrapped.obj|silent.zero' types/logic-authoring.md`: 0 — NEW section

**Anchors with line numbers (post-commit):**
- `types/logic.md` §RT-control-logic: line 82 (inserted before §kitControl patterns)
  - §805.9 flowchart template `[ev: corpus B805]`
  - §805.10 PID anti-windup errorSum clamp `[ev: corpus B805]`
  - Deadband latch-on-cross `[ev: corpus B805]`
  - execute()/changed() split no-reentrancy `[ev: corpus B805]`
  - BLatch edge-D-latch-only `[ev: corpus B805]`
  - [CERT-negative] no ODE facility `[ev: corpus B805]`
  - Health/feedback surface R15.2 `[ev: corpus B808]`
  - Protection ownership `[ev: corpus B805]`
- `types/logic-authoring.md` §Logging a point to history: after §Minimal module (copy-start)
  - BHistoryExt as point ext `[ev: corpus B804]`
  - Interval vs COV `[ev: corpus B804]`
  - BHistoryConfig capacity + fullPolicy `[ev: corpus B804]`
  - One ext per slot `[ev: corpus B804]`
  - B804-G1 OPEN `[ev: corpus B804]`
- `types/logic-authoring.md` §Slot types for externally written values: after §Logging a point to history
  - Table: double/BRelTime/boolean/frozen-enum/action rows `[ev: corpus B823]` `[ev: corpus B822]` `[ev: corpus B828]`
  - Headline rule `[ev: corpus B823]` `[ev: retro obix-statusnumeric-wrapped-put]`
  - Preferred child-ORD path `[ev: corpus B826]` `[ev: corpus B825]`
  - Synchronous link propagation `[ev: corpus B823]` `[ev: corpus B825]`
  - Overlap caveat `[ev: corpus B816]`
- `types/dashboard.md` pointer line: line 33 (before §Reference exemplar)
  `[ev: corpus B823]` `[ev: retro obix-statusnumeric-wrapped-put]`

**Token remap note:** `[ev: retro obix-statusnumeric-wrapped-put]` is cited as the draft carries it.
No file named `*-obix-statusnumeric-wrapped-put.md` exists in `retros/` yet (it is a forward reference
to the live-probe retro from 2026-09-06; the doctrine draft records it at `retros/2026-09-06-…`).
`sweep-fold-audit.sh --strict` does NOT fail on this: the audit checks FOLDED INDEX rows for corpus
citations, not the reverse; no FOLDED row with that stem exists, so no audit WARN is triggered.
Corpus evidence is present for the same finding via `[ev: corpus B823]` on every affected line.

## Proposed kit deltas

| # | File | Delta |
|---|---|---|
| Δ1 | `types/logic.md` | New §RT-control-logic (9 bullet entries, B805+B808) |
| Δ2 | `types/logic-authoring.md` | New §Logging a point to history (5 bullets, B804) |
| Δ3 | `types/logic-authoring.md` | New §Slot types for externally written values (table + 4 prose blocks, B823/B822/B825/B826/B828/B816) |
| Δ4 | `types/dashboard.md` | 1 pointer line (B823) |

## Lessons (≤3)

1. BHistoryExt on a non-BControlPoint parent (B804-G1) has not been station-smoked; cite it as OPEN rather than asserting it works.
2. The "slot type is load-bearing" doctrine (B823) eliminates a class of silent-zero oBIX footguns that are otherwise invisible until a live setpoint write delivers 0.0 on HTTP 200.
3. Promotion requires both corpus `[ev:]` tokens AND a retro citation for each proved live finding; a forward retro token (`obix-statusnumeric-wrapped-put`) is acceptable when the corpus block provides standalone evidence.
