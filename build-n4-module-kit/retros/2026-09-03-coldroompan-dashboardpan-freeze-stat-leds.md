<!-- review-status: folded -->
# Retro — freeze-stat + interlock (ColdRoomPan rt) & LEDs/Cuarto-5 (DashboardPan) · 2026-09-03

Shipping two changes across ColdRoomPan-rt and DashboardPan (rt+ux): a resistance→valve interlock
and a per-evaporator freeze-stat in the rt; output-state LEDs, per-evaporator freeze config, and a
compressor-only Cuarto 5 in the dashboard. Both modules built Java 8 + slotomatic, verify gate ALL PASS.
PROPOSED kit deltas (propose-never-apply). Commits: `e3a9c5f` (coldroompan), `0d8358b` (dashboardpan).

## What this PROVED

1. **The `clean`-lock failure has a SECOND remedy the kit doesn't state: free the Windows lock, don't
   only mirror.** `build.sh` failed at `:ColdRoomPan-rt:clean` with
   `java.io.IOException: Unable to delete file '<niagara_home>/modules/ColdRoomPan-rt.jar'` — the
   deployed jar was locked by Workbench/station on the Windows side. The kit currently frames the MIRROR
   (`mirror-niagara-home.sh`) as THE remedy (BUILD-LOOP §0.b + the script's WHY). This session proved the
   simpler remedy when you own the Windows session: **close Workbench (release the lock), then rebuild
   against the live install** — the retry passed, ALL PASS. The operator explicitly rejected the mirror
   ("no hagas mirror, hazlo al station").
   → **PROPOSED BUILD-LOOP §0.b delta:** present BOTH remedies for a locked `modules/<mod>.jar`:
   (a) free the lock — close Workbench / stop the station you own — then build against the install; or
   (b) mirror (`mirror-niagara-home.sh`) when you must NOT touch the Windows side (a live supervisor,
   a station you don't own, a shared box). The mirror is the safe default for a running production
   supervisor; freeing the lock is fine and faster when it's just your own Workbench holding the jar.

2. **A PURE facade takes annotations-only — no AUTO stubs needed — because `build.sh` runs slotomatic
   BEFORE javac.** Added 26 `@NiagaraProperty` to `BRoomPanel` (C5 HOA modes, BStatusBoolean state
   LEDs, per-evaporator freeze config) with ZERO hand-written stub Property/getter/setter, and the build
   passed: `clean → slotomatic → compileJava` regenerates the whole AUTO region from the annotations
   first, and a facade has no hand code that references the generated getters. Contrast: `BEvaporatorUnit`
   (rt) whose hand code DOES call `getFreezeProtect()` — I added stubs there, but it would ALSO compile
   annotations-only for the same reason (slotomatic precedes javac).
   → **PROPOSED `types/logic.md` "Regenerating slots" delta:** clarify that the AUTO stub is only needed
   to compile WITHOUT slotomatic (IDE, partial build); in the `build.sh` flow annotations-only is
   sufficient and is far less error-prone when adding many slots at once. Keep the "annotation + stub"
   recipe as the belt-and-suspenders option, not the requirement.

3. **Adding a numeric MIN/MAX facet to a module whose facets were all `BRelTime` requires importing
   `BDouble`.** `BEvaporatorUnit`'s existing facets used `BRelTime.makeSeconds(0)`; the freeze diffs
   needed `BFacets.make(BFacets.MIN, BDouble.make(0d))` → compile failed until
   `import javax.baja.sys.BDouble;` was added. The verify-gate "facets" check (no raw-number MIN/MAX)
   passes either way, so this only bites at compile.
   → **PROPOSED METHODOLOGY (slot rules) delta:** one line — "a MIN/MAX facet needs `BDouble.make(...)`;
   confirm `import javax.baja.sys.BDouble` is present (a module that only used `BRelTime` facets won't
   have it)."

4. **Pure-JUnit in WSL: the junit + hamcrest jars live in the gradle cache — name the path.** Ran the
   pure `ColdRoomControl` tests standalone (JDK 8 + JUnitCore) from
   `~/.gradle/caches/modules-2/files-2.1/junit/junit/4.13.2/*/junit-4.13.2.jar` +
   `.../org.hamcrest/hamcrest-core/1.3/*/hamcrest-core-1.3.jar`. A broad `find /` timed out before I
   scoped the search to `~/.gradle`.
   → **PROPOSED delta (toolbelt note / build-verify.md):** the test-file note already says "javac +
   junit-4.13.2.jar (see the kit retro)"; add the exact gradle-cache glob so the next person doesn't
   hunt, and warn against `find /` (scope to `~/.gradle ~/.m2`).

## Cost / evidence

- **Delta 1** cost one failed build (`build.sh` exit 30) + a round-trip with the operator before the
  lock was freed and the retry passed. Evidence: build task output `:ColdRoomPan-rt:clean FAILED …
  Unable to delete … modules/ColdRoomPan-rt.jar`, then a clean rebuild ALL PASS after the lock cleared.
- **Delta 2** evidence: `BRoomPanel` +26 annotation-only slots → `DashboardPan-rt` gate ALL PASS
  (2 types resolve, typecount OK); slotomatic regenerated the AUTO region + type hash unattended.
- **Delta 3** evidence: first `BEvaporatorUnit` compile referenced `BDouble.make` without the import.
- **Delta 4** evidence: `find /` timed out (120s); scoped `find ~/.gradle` returned both jars instantly;
  `JUnitCore` → `OK (21 tests)`.

All four are cheap doc/preflight refinements; none change the build or the gate contract.
