<!-- review-status: folded -->
<!-- module: kit -->
<!-- deltas: 0 -->
<!-- index-comment: truth fix: CONFIGURAR, DETALLADO, MUY DETALLADO, MÁS DETALLADO are triage-console levels (truth-es-levels) -->
# 2026-10-25 · kit · triage-es-levels

**Session**: niagara-research quality round 10 (Ayudante3), branch `fix/triage-es-levels` from origin/main b4a36fc5
**Delta count**: 0 (implementation retro: closes the follow-up of retro triage-informacion-level)

## What happened
The station console prints the JDK 8 `Level.getLocalizedLevelName()`, upper-cased in the bundle's locale, so under
`es` the remaining level names reach the console as `CONFIGURAR`, `DETALLADO`, `MUY DETALLADO` and `MÁS DETALLADO`.
Neither the bash nor the port knew them, so such a header continued the open WARNING/SEVERE block and its frames
could become the previous row's own frame. Now `go/internal/triageconsole` `isLevel` matches the four exactly (`MÁS`
in UTF-8 and in Latin-1, `Á` = 0xC1), and the new `levelField` reads `MUY DETALLADO` / `MÁS DETALLADO` as one token
across a single blank, where the old `fields12` split saw only `MUY`. `MUY`, `MÁS`, `DETALLADOS` and
`MUY DETALLADOS` stay no level; every recorded golden is unchanged.

## Evidence
- Console format: baja `com/tridium/logging/LogSettings.java:91-92` (vineflower) sets
  `java.util.logging.SimpleFormatter` with `%4$s [...][%3$s] %5$s%6$s%n`.
- `%4$s`: `SimpleFormatter.format` passes `record.getLevel().getLocalizedLevelName()` (station JRE 1.8.0_412
  rt.jar, vineflower 1.12.0 `SimpleFormatter.java:36`); `Level.computeLocalizedLevelName` returns
  `localizedName.toUpperCase(bundleLocale)` for the default bundle (`Level.java:73-74`).
- Bundle `sun/util/logging/resources/logging_es` (same rt.jar, javap): Configurar, Detallado, Muy Detallado,
  Más Detallado.
- RED on b4a36fc5: `truth-es-levels` printed `@ Cfg.frame`, `@ Det.frame`, `@ Muy.frame`, `@ Mas.frame` and
  `@ Lat.frame` for the five blocks those headers close; GREEN after, the other 77 goldens byte-identical
  `[ev: go/internal/triageconsole/golden_test.go TestGolden]`. A mutation that disables the two-word join turns
  `truth-es-levels` red again.

## Lessons
- A level name that contains a blank defeats a first-field header split; join the known multi-word names before
  testing the level, and only across the single blank the formatter prints.

---
**Status**: FOLDED 2026-10-25 (implementation complete; log in `ledger.d/2026-10-25-triage-es-levels.md`).
