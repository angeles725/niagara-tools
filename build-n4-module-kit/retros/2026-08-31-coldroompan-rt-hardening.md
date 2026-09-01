# Retro — ColdRoomPan `-rt` hardening (2026-08-31) — PROPOSED kit deltas

Status: **propose-never-apply**. A human folds these into `types/logic.md` / `METHODOLOGY.md`.
Context: applied 5 audit findings to an already-built `-rt` control module; clean Java-8 + slotomatic + jar (bytecode 52, signed).

## What this build PROVED (seeds for `types/logic.md`)

1. **Adding slots to an already-generated rt component regenerates cleanly.** Hand-wrote the annotation + a matching AUTO region, then `slotomatic` re-derived the region and updated the type hash (`BColdRoom` → 1765798213, `BDefrostController` → 1331382228). Hand region only has to compile; slotomatic is authoritative. Confirms METHODOLOGY item "edit annotation + generated + imports, prove with slotomatic."

2. **Restart-persistent internal timer state = a persistent HIDDEN `BAbsTime` slot.** Pattern for any free-running interval that must survive a station restart:
   - slot `lastEventTime : BAbsTime`, `defaultValue = "BAbsTime.NULL"`, `flags = Flags.HIDDEN` (NOT transient).
   - on `atSteadyState`/re-arm: `elapsed = Clock.millis() - getLastEventTime().getMillis()`; schedule `max(interval - elapsed, 0)`; `isNull()` (never happened) → full interval.
   - record `setLastEventTime(Clock.time())` when the event fires.
   Without this, `Clock.schedule(this, getInterval(), ...)` on every `atSteadyState` resets the countdown → frequent restarts starve the interval. [verified API: `javax/baja/sys/Clock.java` millis()/time()/schedule; `BAbsTime.java` make(long)/getMillis()/isNull()/NULL]

3. **A plain non-BObject helper class compiles and bundles fine** (`CrLog.class` shipped in the jar). Use one for a shared per-module `java.util.logging.Logger` so `logError()` swallows-AND-logs on the engine thread instead of discarding the throwable. N4 idiom (corpus B20/B206): no `BLoggingService`; `Logger.getLogger("<module>")`.

4. **Safety fail-mode as a config slot, not a baked-in constant.** A sensor-fault control decision (hold last vs force-cool vs force-off) is a plant-safety posture the operator owns. Expose it as a `SUMMARY|OPERATOR` slot with the default = current behavior, so hardening never silently changes control on a live plant.

5. **`niagaraTest` does NOT run in WSL — make control logic a PURE class and test it standalone.** Verified: `niagaraTest` (runs `srcTest/`) needs the native `bin/test.exe` (Windows) + dev license; the plain `test` task runs `src/test` (empty). So a test under `srcTest` is dead weight in WSL. Fix: extract the decision into a plain class with ZERO Baja types (here `ColdRoomControl.decideCall`; the Baja `BComponent` keeps a thin adapter that reads slot value+status and delegates). Then compile+run with plain `javac` + the cached `junit-4.13.2.jar` (+ `hamcrest-core-1.3.jar`), no station:
   ```
   javac -source 8 -target 8 -cp <junit> Pure.java PureTest.java -d out
   java -cp out:<junit>:<hamcrest> org.junit.runner.JUnitCore <pkg>.PureTest
   ```
   This is the ONLY way to get real, runnable unit coverage of control/safety logic in WSL. METHODOLOGY already says "unit-test the pure-Java model" — this is the concrete how, plus WHY srcTest+niagaraTest is not it. Mirrors the DashboardDispatch (pure router) pattern.

## Type-guide gap this fills

`types/logic.md` TODO asked to "flesh out safety fail-modes from a real build." Seeds 2+4 above are that: the restart-anchor pattern and the fault-posture-as-slot rule. Fold into `types/logic.md` under a new "Safety fail-modes & timers" subsection.

## Build-target & deploy lessons (added 2026-09-01) — seeds for `build-verify.md`

Building the SAME module for a Honeywell Workbench (OptimizerSupervisor **N4.14.0.162**) after it was first built against **PowerB 4.15.3.28** surfaced three build-target rules the kit does not yet state:

6. **Build against the niagara_home of the TARGET version — the manifest stamps its `baja` dependency version.** Verified: built against PowerB 4.15 → `META-INF/module.xml` carries `<dependency name="baja" vendor="Tridium" vendorVersion="4.15"/>`; a **4.14 station/Workbench REJECTS that jar** (it needs baja ≥ 4.15). Rebuilt against Honeywell 4.14 → `vendorVersion="4.14"`, which loads in 4.14 (and forward in 4.15). RULE: pick `niagara_home` = the **lowest** target station version you must support. Check: `unzip -p <jar> META-INF/module.xml | grep baja`.

7. **The gradle-plugin version pinned in `settings.gradle.kts` MUST exist in `<niagara_home>/etc/m2`.** PowerB 4.15 ships `7.6.22`; Honeywell 4.14 ships only up to `7.6.17` (common to both: 7.6.1/7.6.3/7.6.5). A hardcoded `7.6.22` fails against 4.14 (plugin not found; Tridium plugins are not on the public portal). FIX — make it overridable: `val gradlePluginVersion = providers.gradleProperty("niagaraPluginVersion").getOrElse("7.6.17")`, then `-PniagaraPluginVersion=<v>` per install. `settingsPluginVersion` (7.6.3) is common to both.

8. **A running station LOCKS `<niagara_home>/modules/<module>.jar` — build against a READ-ONLY MIRROR, never stop a live supervisor.** With `station.exe`/`niagarad.exe` running, the niagara-module plugin's `clean`/`jar` fails deleting the deployed jar (`java.io.IOException: Unable to delete file …/modules/<module>.jar`). Do NOT stop the operator's live supervisor. Instead mirror the install — symlink every top-level dir + every module jar EXCEPT the one being built — into a dir with a **writable `modules/`**, and build with `-Pniagara_home=<mirror>`. The jar lands in `build/libs` (deliverable) + `<mirror>/modules` (throwaway); the real install is never written. This is the documented WSL `niagara_home`-mirror pattern made concrete + non-disruptive.

## Tools created

| Tool (current path) | CREATED — purpose | ORACLE | VERDICT |
|---|---|---|---|
| `mirror-niagara-home.sh` (`Cliente/Leon-Guanjuato/bitacora/`) | build a read-only mirror of a Niagara install with a writable `modules/`, so a module compiles against a specific/locked niagara_home without touching a running station (seed 8) | — | **promote** → `build-n4-module-kit/toolbelt/mirror-niagara-home.sh` + reference from `build-verify.md` |
