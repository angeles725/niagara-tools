# Station diagnostics — engine stats, log levels, console messages

The triage read of a running station. How a module logs, exposes spy pages and surfaces faults is in
`types/observability.md`; console triage is `toolbelt/triage-console.sh`. `[ev: retro n4-course-parity-observability-horizons Δ4]`

Derivations: course deck/lab screenshots in corpus B1255 (§1255.7), B1256 (§1256.3), B1257 (§1257.10), B1236
(§1236.2). Code: `V=organized/baja/baja/vineflower/com/tridium` (4.14.0.162 decompile, under niagara-research).

## 1 · Engine-thread stall: Resource Manager statistics

The engine's resource report (Workbench Resource Manager view) holds `engine.scan.peak`, `engine.scan.peakInterscan`
(the longest gap between two engine scans after steady state, with `engine.scan.timeOfPeakInterscan`) and
`engine.queue.actions` (actions waiting for the engine thread, shown as `<now> (Peak <n>)`), next to the short,
medium and long timer queues. [CERT] `V/sys/engine/EngineManager.java:169,414-431`. A long `peakInterscan` and a
high action-queue peak date a callback or action that held the engine thread; correlate the time with the console.

## 2 · Log levels `[ev: retro n4-course-lexicon-logging-testing Δ5]`

- **Default INFO.** Every logger starts at INFO (course Lab 7 step 10; corpus B1236 §1236.2), so FINE lines are
  invisible until someone raises the category.
- **Raise one category, do not log louder.** Add the logger name (the module name, `types/observability.md` § 1) at
  FINE in Services › DebugService (type `baja:LoggingService`, `baja.lexicon:311`) or through `spy:/logSetup`
  (`V/sys/Nre.java:730`). FINE lines then reach the console but not LogHistory (`types/observability.md` § 1,
  `minimumSeverity`).
- **Two spy pages for "is it loaded, is it logging":** Spy › Log Setup lists every logger the station defined,
  while the DebugService lists only configured categories; Spy › sysManagers › registryManager lists the installed
  modules and each module's types (the place to confirm a type registered after a deploy).
  `[ev: corpus B1418 §1418.2 f]` `[ev: corpus B1417 §1417.3 f]` `[ev: retro course-coverage Δ13]`
- **Per-frame and per-poll traces go at FINE** behind `isLoggable(Level.FINE)`; INFO in a poll or receive body floods
  the console and LogHistory of every site.

## 3 · Console messages and what they mean

| Console text | Meaning | Action |
|---|---|---|
| `ActionInvokeException` → `ServerException` → `AccessControlException` | a module permission is missing; read the innermost exception (B1257 §1257.10 rule 5) | raise logger `security.niagaraPolicy` (`V/security/UIPermissionGroup.java:40`) to FINE: it prints the permission and the ProtectionDomain that failed (B1255 §1255.4). If that domain is a framework jar (`fox-rt.jar` in the course), the check reached a caller lower on the stack: wrap your call in `AccessController.doPrivileged` |
| SEVERE `Cannot stop component: <name>` | something threw out of `stopped()`/`descendantsStopped()` (`V/sys/engine/EngineUtil.java:105,123`) | in the course it was `shutdownNow()` denied without `MANAGE_EXECUTION`; the threads leak (B1255 §1255.7) |
| WARNING `Could not validate certificate path for entry … in module …` | the jar's signer is not trusted; logged only while verification is not required (`V/sys/module/ModuleClassLoader.java:77,497-503`) | it loads at `low` and fails at `medium`/`high` (B1257 §1257.10 rule 3); sign per `build-verify.md` § Signing per deploy target |
| a UI label equal to its lexicon key, or untranslated text | see `types/issues-and-gotchas.md` K1 (diagnostic lexicon) | — |

**Spanish-locale consoles print localized level names.** OpenJDK 8 `sun.util.logging.resources.logging_es`:
SEVERE `Grave`, WARNING `Advertencia`, INFO `Información`, CONFIG `Configurar`, FINE `Detallado`, FINER
`Muy Detallado`, FINEST `Más Detallado` (read from `jre/lib/rt.jar` of OpenJDK 1.8.0, 2026-10-23). A live course
station printed them upper-case (`INFORMACIÓN`, `GRAVE`, `DETALLADO`). `triage-console.sh` recognises only `GRAVE`,
`ADVERTENCIA` and `INFORMACI*`, and FINER/FINEST are two words. `[ev: retro n4-course-lexicon-logging-testing Δ4]`
`[deferred: retro n4-course-lexicon-logging-testing Δ4]` (script fix owed as G73).
