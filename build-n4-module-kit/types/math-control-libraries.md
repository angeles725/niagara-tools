# Math / PID / Control Library Selection for N4 rt Modules

> **Scope: rt Java libraries only.** Browser-side JavaScript libraries (three.js,
> React, etc.) are covered in `types/dashboard.md`. Frontend ux/wb JavaScript
> packaging is covered in the spa-library-integration retro (separate concern).

---

## The Compact3 gate `[ev: corpus B1023, B426, B31]`

JACE ships `azul-zre-compact3-qnx7-arm` — Azul JDK 8 **Compact3** profile on
QNX7 ARMLE-V7. Compact3 is the broadest of the three compact profiles
(compact1 ⊂ compact2 ⊂ compact3 ⊂ full SE 8).

**Absent in Compact3** (relevant exclusions): Java2D / AWT / Swing, CORBA/RMI,
JDBC, JNDI, some `javax.xml` subsets.

**Present in Compact3** (key inclusions): `java.lang`, `java.util`,
`java.util.concurrent.*`, `java.math`, `java.io`, `java.net`, `java.security`,
`javax.net`, `javax.crypto`, `javax.script` (but no scripting engine ships in
the compact JRE — see Aviator/MVEL below).

**Rule:** a pure-Java library that touches none of the excluded packages runs on
JACE unmodified. "Pure Java" alone is not sufficient — a library that invokes
`ClassLoader.defineClass` dynamically (expression compilers, reflection-heavy
proxy generators) can fail at runtime on the JACE even though it compiles and
passes the Supervisor build.

**Existing gate:** `check_compact3_imports` in `toolbelt/verify-module.sh`
already enforces the SE-API constraint for rt/ux source (warns on `java.awt`,
`javax.swing`, `java.sql`, etc.). No additional script is required — the
existing check covers the static import boundary. Document the *runtime*
risks (dynamic bytecode, native JNI) separately here; they are not statically
detectable and must be identified per-library at design time.

> **Δ4 note — COMPACT3-GATE-SCRIPT1 downgraded:** the retro proposed a new
> `toolbelt/check-compact3.sh` stub. Because `check_compact3_imports` in
> `verify-module.sh` already enforces the static SE-API constraint, a standalone
> script would be redundant. The doc note above is the canonical pointer; no new
> script has been created.

**GC amplification on JACE:** ParallelGC (JDK 8 default) stop-the-world Full GC
pauses last 500 ms–2 s on heap pressure. Heavy math that allocates large
intermediate arrays amplifies this risk on rt — another reason to push
optimization loops and matrix inversions off the engine thread. `[ev: corpus B31 §31.3.8]`

---

## Default: kitControl.BLoopPoint `[ev: corpus B539, B733]`

**Use `kitControl.BLoopPoint` for all standard modulating loops.** It ships with
every N4 station (no Maven dependency, no JAR to bundle, no Compact3 gate
check).

Key capabilities already in BLoopPoint:

- Ramp anti-windup (rescales `errorSum` at slope transitions)
- Bumpless-transfer pre-load on disable / re-enable
- Four `disableAction` behaviors: `max / min / hold / zero`
- `direct` / `reverse` action sign
- `BLoopAlarmAlgorithm` for loop-deviation alarming
- `propagateFlags` masking
- `executeTime` clamped to [100 ms, 60 s]

Official tuning recommendation (§539.7): **P or PI**; PID is "seldom justified"
on HVAC/refrigeration loops.

**A third-party PID (MiniPID or custom) is the exception** — justified only when
BLoopPoint's fixed API is genuinely insufficient (e.g. serial commissioning
workflow, non-standard anti-windup shape). Add the dependency, bundle the JAR,
and pass the Compact3 gate check only when there is a concrete architectural
reason BLoopPoint cannot meet the requirement. `[ev: corpus B539 §539.1–539.7, B733 §733.2–733.3]`

---

## Engine-thread / BWorker rule for heavy math `[ev: corpus B730 §730.8, B737, B31]`

The engine thread must remain **< 10 ms per cycle**. Any computation that can
take longer than ~2 ms must be posted off the engine thread.

**What must go off-thread:** optimization loops (BOBYQA, CMA-ES, simplex),
ODE integration, matrix inversion, state-observer updates, expression
evaluation.

**Mechanism:** `BWorker` child component (single named thread, framework-managed
lifetime). Post work via the `BWorker.post(Runnable)` / `postAsync()` API;
deliver results back to the engine-thread component via a slot write or
`postAsync`.

**What is FORBIDDEN in BComponent subclasses:**
`ScheduledExecutorService`, `Executors.*`, `new Thread(...)`. The station
`SecurityManager` denies `modifyThread` to module code; any of these throws at
runtime and leaves the unit in a broken state. Use `Clock.schedule` /
`Clock.schedulePeriodically` for time-triggered callbacks and `BWorker` for
blocking or compute-heavy off-thread work.

> Full engine-thread and BWorker recipe: see `types/logic.md` §Off-engine
> blocking I/O (BWORKER-OFFTHREAD1) and §Engine-thread cost — HogsPage
> diagnostic. The threading discipline and the library verdict are inseparable:
> a safe library used on the engine thread still stalls it.

---

## State machines: plain BEnum pattern `[ev: corpus B819, B730, exemplars BColdRoom / BCompressorControl]`

**Prefer a plain `BEnum` slot + engine-cycle if/else transitions** for
AHU/defrost/lead-lag state machines. This is the proven N4 pattern:
`BColdRoom` and `BCompressorControl` both use it; the existing exemplars
demonstrate that it covers the refrigeration state machines encountered in
production.

**Squirrel Foundation** (`squirrel-foundation:0.3.10`) is pure Java but its
reflection-heavy annotation processor and dynamic proxy generation are
**unverified on Compact3 ZRE** — `java.beans` is absent and
`java.lang.reflect` is constrained on Compact3. Adopt only after a Compact3
smoke-test on the target JACE.

---

## Library verdict table `[ev: corpus B67, B1023, B426, B31, B539, B733, B800, B806]`

| Library | Maven artifact | N4 verdict | Notes |
|---------|---------------|-----------|-------|
| **Apache Commons Math 3.6.1** | `org.apache.commons:commons-math3:3.6.1` | **Compact3-safe rt ✓** | Pure Java; no native; no excluded APIs. Covers stats, linear algebra, Brent/BOBYQA/CMA-ES/simplex/LSQ optimizers, Kalman (linear), ODE integrators, interpolation. Heavy calls (optimizer loops, ODE integration) → BWorker. B67 explicitly recommends it for percentile/variance/skewness extension. |
| **EJML 0.43** | `org.ejml:ejml-simple:0.43` | **Compact3-safe rt ✓** | Pure Java; compact matrices; recommended for state observers and own-Kalman implementations. Matrix inversions → BWorker. |
| **ND4J** | `org.nd4j:nd4j-native-platform:*` | **NOT viable on JACE — Supervisor/offline only** | JNI-backed native binaries (OS/arch-specific `.so`/`.dll`); incompatible with Compact3 ZRE on QNX7 ARM. Hard binary incompatibility, not a configuration issue. Use on Supervisor or Windows offline analysis only. |
| **MiniPID** | GitHub only (no Maven Central release) | **Skip — prefer kitControl.BLoopPoint** | Simple pure-Java PID. BLoopPoint already ships ramp anti-windup + bumpless transfer + loop alarms + priority-array output. Add MiniPID only when BLoopPoint's fixed API is genuinely insufficient. |
| **kitControl.BLoopPoint** | N4 framework (no Maven dep) | **Default choice for PID — always available** | See §Default: kitControl.BLoopPoint above. |
| **Squirrel Foundation** | `org.squirrelframework:squirrel-foundation:0.3.10` | **Compact3: unverified — prefer plain BEnum pattern** | Reflection-heavy annotation processor; dynamic proxy generation may fail on Compact3 ZRE. Verified only on Supervisor. Smoke-test on JACE before any rt use. |
| **Aviator** | `com.googlecode.aviator:aviator:5.x` | **Compact3: NOT proven — risk of `ClassLoader.defineClass`** | Expression compilers invoke `ClassLoader.defineClass` to emit bytecode at runtime; SecurityManager on Compact3 ZRE may deny this. The failure occurs at runtime on the JACE, not at compile time on the Supervisor build host. Supervisor use only until verified. |
| **MVEL2** | `org.mvel:mvel2:2.4.x` | **Compact3: NOT proven — same risk as Aviator** | Dynamic bytecode generation; same JACE runtime risk as Aviator. Supervisor use only until verified. |
| **ScheduledExecutorService / Executors.\*** | JDK (`java.util.concurrent`) | **NOT viable in BComponent** | SecurityManager denies `modifyThread`; throws at runtime. Use `Clock.schedule` / `Clock.schedulePeriodically` for time-triggered callbacks; `BWorker` for blocking off-thread work. |

---

## Cross-refs

- **Engine-thread / BWorker recipe (BWORKER-OFFTHREAD1):** `types/logic.md`
  §Off-engine blocking I/O and §Engine-thread cost — HogsPage diagnostic. These
  are the canonical threading doctrines; this document does not duplicate them.
- **Compact3 / packaging mechanics (JAR embedding, shading, SE-API gate):**
  `types/third-party-libraries.md`. This document covers library *viability*
  (which library, what caveats); that document covers *how to bundle* a
  third-party JAR into an N4 module.
- **Frontend / ux JavaScript libraries:** `types/dashboard.md` (JS libs loaded
  from `rc/` resource folders are a separate mechanism entirely).
- **ML / predictive-model libraries (tabular classifiers/regressors, ONNX
  inference, DL4J, Weka, Smile, XGBoost4J, PROPOSE-VALIDATE safety architecture,
  train-offline/infer-ONNX pattern, operating-mode data discipline):**
  `types/ml-libraries.md`. That document covers the ML layer; this document covers
  the non-ML control-math layer (PID, stats, linear algebra, state machines). The
  two serve different authors and are intentionally split.
