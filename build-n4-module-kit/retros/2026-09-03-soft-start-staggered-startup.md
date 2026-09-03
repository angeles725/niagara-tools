# Retro — soft-start / staggered startup (ColdRoomPan + CompPan) · 2026-09-03

Second retro of the day (the first was `2026-09-03-coldroompan-dashboardpan-freeze-stat-leds.md`). Driven
by a LIVE FIELD INCIDENT: the operator loaded the module and "todo entró de golpe al arrancar, algo se
pudo haber quemado". Fixed across v2.0.1 → v2.0.3 (commits 5a593c5, ac5e58f, f738077). PROPOSED kit deltas
(propose-never-apply). Only genuinely NEW items vs the first retro.

## What this PROVED

1. **A NEW safety/startup behavior must default to the SAFE value, not to the historical-unsafe one —
   and that does NOT violate "never change control silently".** The kit's rt-hardening #4 ("default a
   hardening change to current behavior") led me to ship `powerOnDelay = 0 = disabled` first (v2.0.1). But
   "current behavior" WAS the incident (all evaporators energize in one cycle), so loading the fix with it
   OFF by default did not help — the operator asked, twice, "va a arrancar de golpe, sí o no?" and the
   honest answer was YES. The reconciliation (v2.0.2): make `0` mean **AUTO** (compute a safe default), and
   keep `> 0` as an explicit override. The "don't change control silently" rule protects an EXISTING
   operator setting; a brand-new slot has none to protect, so its default should be SAFE.
   → **PROPOSED METHODOLOGY delta (refine rt-hardening #4):** for a NEW *safety/startup* slot, default to
   the SAFE behavior (auto), not to the pre-fix behavior. Use a sentinel (`0 = AUTO`) + explicit override
   so operators can still tune, but "load it and walk away" is safe out of the box. Reserve "default to
   current behavior" for changes that alter an EXISTING, already-commissioned control path.

2. **A component can self-sequence across siblings by walking the station tree — no coordinator
   component needed.** `BEvaporatorUnit.computeAutoStartupMs()` derives a unit's plant-wide position =
   (units in the prior sibling `BColdRoom`s, slot order, via `getParent()`+`getChildren(BColdRoom.class)`)
   + (its index in its own room), times an internal step, all inside a `try/catch → return 0` so it never
   throws on the engine thread. That gave a whole-plant staggered start WITHOUT a new `@NiagaraType`
   coordinator + palette entry + station restructure (the heavier option the operator rejected).
   → **PROPOSED types/logic.md delta:** document "self-order by sibling walk" as the lightweight pattern
   for startup sequencing (or any position-derived behavior): walk `getParent()`/`getChildren(Type.class)`
   in slot order, guard with try/catch→safe-fallback, never a coordinator unless cross-module ordering or
   an explicit token is required.

3. **The `modules/<jar>` clean-lock is the NORMAL case during an iterate→deploy→test loop, not an edge
   case.** Hit `Unable to delete '<niagara_home>/modules/<mod>.jar'` at the `:clean` task THREE times this
   session (the operator had Workbench open loading the module each time); each time: close Workbench →
   rebuild passes. The failure surfaces only as a gradle stacktrace + exit 30.
   → **PROPOSED build.sh delta:** detect the `Unable to delete …/modules/<mod>.jar` clean failure and print
   ONE actionable line ("jar locked — close Workbench / stop the station, or build against a mirror
   (mirror-niagara-home.sh)") instead of only the raw stacktrace. Complements the first retro's BUILD-LOOP
   §0.b doc delta (two remedies) — this makes the tool say it at the moment it happens.

4. **The version-bump / release procedure belongs in the kit.** Shipped v2.0.0 → v2.0.3. Facts learned:
   the module version is `defaultModuleVersion("X.Y.Z")` in each module's `build.gradle.kts` `vendor {}`
   block → stamped as `vendorVersion` in `module.xml` at build time; a bump REQUIRES a rebuild to reach the
   jar; per-module version skew is fine (rebuild only the changed modules — DashboardPan stayed 2.0 while
   ColdRoomPan went 2.0.3); the repo tracks `build/libs/*.jar`, so a release commit includes the rebuilt
   jar; tag `vX.Y.Z`.
   → **PROPOSED build-verify.md (or BUILD-LOOP) delta:** a short "Versioning & release" section with the
   above — where the version lives, that a rebuild stamps it, per-module skew is OK, and the tag/commit
   convention.

5. **A "safe default" that hides a magic number the operator will want to tune should be a config slot,
   not a constant.** The cascade step was a constant `AUTO_STEP_MS`; the operator changed the value twice
   (8 s → 60 s), each a recompile + rebuild + redeploy cycle.
   → **PROPOSED delta (reinforce):** when an auto/default behavior embeds a number an operator is likely to
   tune in the field (a stagger step, a timeout), expose it as an OPERATOR config slot from the start, or
   at minimum document the override path prominently, to avoid a recompile per tweak.

## Cost / evidence
- **Delta 1** cost two operator round-trips ("sí o no?") + a re-release (v2.0.1 opt-in → v2.0.2 auto)
  before "load and walk away" was actually safe.
- **Delta 2** evidence: `computeAutoStartupMs()` in `BEvaporatorUnit`; ColdRoomPan-rt gate ALL PASS, the
  tree-walk compiled clean.
- **Delta 3** evidence: 3× `:ColdRoomPan-rt:clean FAILED … Unable to delete … modules/ColdRoomPan-rt.jar`,
  each cleared by closing Workbench.
- **Delta 5** evidence: `AUTO_STEP_MS` edited 8000→60000 as a whole v2.0.3 cycle for a one-number change.

All are cheap doc/tooling refinements; none change the gate contract.
