<!-- review-status: pending -->
# Retro: Campaign 7 PR3 — DashboardPan-ux as the write-surface reference exemplar (B796)

**Date**: 2026-09-05  
**Scope**: `build-n4-module-kit/types/dashboard.md`  
**PR branch**: `docs/c7-dashboard-b796`  
**SDD change**: `build-n4-module-campaign7`

## What was done

Folded B796 §796.4 into `types/dashboard.md` as a "Reference exemplar (our own module)" block under
`## ux — servlet + SPA`. The block adds:

- The DashboardPan-ux routing seam location (`DashboardDispatch.java:30`) and its 14 off-station `@Test`.
- The DUX2 anti-pattern instance (`DashboardReader.java:66` — `public final` but impure, 15+ Baja imports).
- The five DWS1 gates scored on real code (file:line for gates 1/2/3/5; gate 4 REQUIRED-but-absent → issue #49).
- The sentence that Tridium ships no vendor exemplar for the SPA/servlet split (B791 THIN verdict).

Every content line cites `[ev: corpus B796]`; B763 and B791 are cited where they are the authoritative source.

## Why

`types/dashboard.md` had the DWS1/DUX1/DUX2 RULES (from Campaign 6 PR4) but no concrete pointer
to WHERE those rules are implemented in our own code. B796 §796.4 captured exactly that evidence
with file:line. The kit teaches all five gates while being honest that the exemplar currently scores
4/5 (gate 4 tracked in issue #49).

## Lessons

1. **Grep-before-fold is a CD3 discipline, not a formality:** `DashboardPan-ux` and `B796` returned
   0 hits before the edit — confirming no double-fold risk. The existing DUX1/DUX2/DWS1 bullets use
   `DashboardDispatch` and `DashboardReader` by class name, not the `DashboardPan-ux` compound term,
   so the exemplar block is additive and distinct.

2. **A 4/5 exemplar is more honest than a 5/5 aspiration:** gate 4 is documented as
   REQUIRED-but-absent so the kit teaches the ceiling, not the floor. The open issue (#49) tracks
   the residue without blocking the exemplar from being useful today.

3. **The routing seam table (DWS1 five gates) is the right artifact shape:** a table with gate /
   status / file:line collapses a multi-paragraph B-block into a scannable reference, consistent
   with the cognitive-doc-design "recognition over recall" principle.
