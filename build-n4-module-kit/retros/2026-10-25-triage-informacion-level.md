<!-- review-status: folded -->
<!-- module: kit -->
<!-- deltas: 0 -->
<!-- index-comment: truth fix: INFORMACIÓN is a triage-console level (truth-informacion-level) -->
# 2026-10-25 · kit · triage-informacion-level

**Session**: niagara-research quality round 10 (Ayudante3), branch `fix/triage-informacion-header` from origin/main
643ed84c
**Delta count**: 0 (implementation retro: closes the owed truth fix of retro go-triage-console)

## What happened
The bash `is_level` compared `substr(s, 1, 8)` with the 9-byte `"INFORMACI"`, which never matches, so an
`INFORMACIÓN` header (the Spanish java.util.logging name of INFO) never closed the open WARNING/SEVERE block and its
stack frames could become the previous row's own frame. The Go port kept that for byte identity. Now
`go/internal/triageconsole` `isLevel` matches `INFORMACIÓN` exactly, in UTF-8 and in Latin-1 (`Ó` = 0xD3), as a
declared truth difference; `INFORMACIONAL` and the unaccented `INFORMACION` (`edge-levels`) stay no level, so every
recorded golden is unchanged.

## Evidence
- JDK 8 `Level.getLocalizedName()` under `Locale("es")`: GRAVE, ADVERTENCIA, INFORMACIÓN, CONFIGURAR, DETALLADO,
  MUY DETALLADO, MÁS DETALLADO (2026-10-25, local openjdk 8 run)
- RED on 643ed84c: `truth-informacion-level` printed `@ Utf.frame(...)` and `@ Lat.frame(...)` for the blocks the
  INFORMACIÓN header closes; GREEN after, the other 76 goldens byte-identical `[ev: go/internal/triageconsole/golden_test.go TestGolden]`
- GRAVE and ADVERTENCIA are exact comparisons in the bash, so they never had the prefix-length bug.

## Follow-ups (not done here)
- CONFIGURAR, DETALLADO, MUY DETALLADO and MÁS DETALLADO (the other Spanish level names) are not levels in the bash
  or the port, so such a header continues the open block like INFORMACIÓN did; a two-word name needs a header
  parser that is not `fields12`. No console in the corpus is known to log at those levels.

## Lessons
- A prefix compare whose length argument differs from the literal's byte length never matches; with a multibyte
  literal, compare the whole word in each encoding instead.

---
**Status**: FOLDED 2026-10-25 (implementation complete; log in `ledger.d/2026-10-25-triage-informacion-level.md`).
