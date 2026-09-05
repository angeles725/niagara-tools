<!-- review-status: folded -->
# Retro: Campaign 7 PR6 — verify-module.sh --plano (B797 aspect-ratio agreement check)

**Date**: 2026-09-05 | **Module**: kit | **PR**: feat/c7-plano | **Change**: build-n4-module-campaign7

## Summary

Delivered `verify-module.sh --plano <index.html|jar>`: a standalone mode that verifies
`Rc == Rv == Ri` and every numeric `aspect-ratio` equals `Rc` via integer cross-multiplication
(auto exempt). PL1-PL6 green; mutation (full count-only) causes PL2+PL4 to lose their FAIL.
Real DashboardPan-ux smoke: FAIL naming `1247/771 != Rc(1248/891)` — confirms issue #49.

## Proposed deltas

1. **`id="zonas"` targeting for Rv parse** — The design said "first viewBox; ≥2 distinct → FAIL".
   The real DashboardPan-ux has three SVG elements with different viewBoxes (zonas 1248×891,
   fotoCsvg 100×100, chart 900×340). Naive `grep viewBox` → "ambiguous" instead of the expected
   1247/771 FAIL. Fix: grep for `id="zonas"` first, fall through to any-viewBox only if absent.
   Lesson: real-SPA multi-SVG files require targeted element selection, not file-wide grep.

2. **Strict TDD confirmed the B797 formula early** — Cherry-picking the QA RED before touching
   the script forced a full formula understanding before a single line was written. The mutation
   (all three _ceq checks removed) cleanly falsified both PL2 and PL4, proving cross-source
   equality is load-bearing.

3. **`od -An -tu1 -N24` + awk multi-line collect** — Reading PNG IHDR bytes without Python
   requires awk to collect the numbers across od's line-wrapped output. The pattern
   `{for(i=1;i<=NF;i++)a[++n]=$i}` accumulates all 24 bytes correctly regardless of line width.

4. **No $HOME, no version control, VCS-free** — All three constraints satisfied; the `mktemp`
   temp-dir is the only external state (cleaned up via `trap _plano_cleanup EXIT`).

5. **First 64 b64 chars sufficient for IHDR** — 64 base64 chars → 48 decoded bytes; IHDR ends
   at byte 33. `head -c 64` on the inline data URI gives a multiple-of-4 slice that `base64 -d`
   decodes cleanly even for tiny (44-char) images like the 2×3 fixture.
