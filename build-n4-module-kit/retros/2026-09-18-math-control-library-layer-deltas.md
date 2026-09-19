<!-- review-status: folded -->
# 2026-09-18 · kit · math-control-library-layer-deltas

**Session**: corpus-mining for build-n4-module (operator: math/PID/control libraries for N4 rt modules — Java 8).
**Delta count**: 4

---

## What happened

Mined B733 (kitControl math blocks + BLoopPoint / modulating-output chain), B539 (BLoopPoint deep:
ramp anti-windup, disable actions, PI tuning), B730 (rt authoring idioms: BWorker off-thread), B737
(engine-thread + watchdog chain), B31 (JACE Compact3 ZRE profile, GC), B1023 (Compact3 JVM dist —
`azul-zre-compact3-qnx7-arm` on QNX7 JACE), and B67 (Analytics module: no Apache Commons Math used
but explicitly recommended for percentile/variance/skewness extension).

No prior kit file covers the third-party math/control library selection question — `types/logic.md`
documents BLoopPoint (LC2, B539) and the engine-thread scheduling constraints, but there is no
dedicated map answering "which external library can I add to an rt module, and with what N4-specific
caveats."  This retro opens that map.

The resource-threading retro (2026-09-18-resource-threading-consumption-deltas.md) covers the
engine-thread + BWorker recipe (Δ4/BWORKER-OFFTHREAD1) and scheduling constraints
(Δ1/ET-DIAG1).  Those are NOT re-proposed here — this retro cross-refs them.

The spa-library-integration retro (2026-09-18-spa-library-integration-deltas.md) covers ux/wb
frontend library packaging; that is a separate concern from rt Java libraries.

---

## Evidence

### Compact3 gate

- **B1023 §ND4** — JACE ships `azul-zre-compact3-qnx7-arm` (Azul JDK 8 Compact3 profile, QNX7
  ARMLE-V7).  Compact3 is the broadest of the three compact profiles (compact1 ⊂ compact2 ⊂
  compact3 ⊂ full SE 8).  It excludes: Java2D/AWT/Swing, CORBA/RMI, JDBC, JNDI, and some
  `javax.xml` subsets.  It INCLUDES: `java.lang`, `java.util`, `java.util.concurrent.*`,
  `java.math`, `java.io`, `java.net`, `java.security`, `javax.net`, `javax.crypto`, and
  `javax.script` (but no scripting engine ships with the compact JRE).  Pure Java libraries that
  do not touch the excluded packages run on JACE unmodified. [ev: corpus B1023]
- **B426** — `program` module compiler flags: `-profile compact3` when all deps are `rt`/`wb`;
  full SE only when any dep is `ux`+.  This confirms that the `rt` compile profile targets
  compact3 explicitly. [ev: corpus B426]
- **B31 §31.3.8** — GC-induced engine-thread latency: JACE QNX7 JVM runs ParallelGC (JDK 8
  default); Full GC pauses are stop-the-world and can last 500 ms–2 s on heap pressure.  Heavy
  math allocating large intermediate arrays amplifies this risk on rt. [ev: corpus B31]

### BLoopPoint — the N4-native PID

- **B539 §539.1–539.7** — BLoopPoint ships ramp anti-windup (rescales errorSum at slope
  transitions), bumpless-transfer pre-load on disable, four `disableAction` behaviors
  (`max/min/hold/zero`), `direct`/`reverse` action sign, `BLoopAlarmAlgorithm` for loop-deviation
  alarming, and `propagateFlags` masking.  The official tuning methodology (§539.7) recommends
  P or PI; PID is "seldom justified" on HVAC/refrigeration loops.
  `executeTime` is clamped to [100 ms, 60 s].  [ev: corpus B539]
- **B733 §733.2–733.3** — BLoopPoint fits refrigeration modulating control (EPR / VFD compressor
  speed / EEV); the math-block family (BQuadMath / BUniMath / BBiMath) handles arbitrary signal
  pre-processing ahead of the loop.  The block runs on the engine thread — only the setpoint
  computation and output clamping, not long optimization loops.  [ev: corpus B733]

### Off-engine rule for heavy math

- **B730 §730.8 + B737** — heavy or blocking work must run on a `BWorker` child (single named
  thread, framework-managed lifetime), not inline in `execute()`/`changed()`.  The engine thread
  must remain <10 ms per cycle.  A compute-heavy algorithm (optimization loop, ODE integration,
  matrix inversion) posted to BWorker returns its result via `post()/postAsync()` back to the
  engine-thread component.  ScheduledExecutorService / `Executors.*` / `new Thread(...)` are
  forbidden in BComponent subclasses (SecurityManager denies `modifyThread`).
  [ev: corpus B730 §730.8, B737, B31]

### Analytics third-party map (B67)

- **B67 §67.1 + §67.7** — the Analytics module ships zero Apache Commons Math imports; all stats
  implemented custom in `combine/`.  B67 explicitly recommends adding `commons-math3 3.6.1`
  for percentile/variance/skewness/kurtosis coverage.  This is the only first-hand corpus
  evidence that an N4 module team assessed Apache Commons Math for rt use and found it viable
  (pure Java, no excluded APIs).  [ev: corpus B67]

---

## Library verdict table

| Library | Maven artifact | N4 verdict | Notes |
|---------|---------------|-----------|-------|
| **Apache Commons Math 3.6.1** | `org.apache.commons:commons-math3:3.6.1` | **Compact3-safe rt ✓** | Pure Java; no native; no excluded APIs; stats, linear algebra, Brent/BOBYQA/CMA-ES/simplex/LSQ optimizers, Kalman (linear), ODE integrators, interpolation. Heavy calls (optimizer loops, ODE integration) → BWorker. |
| **EJML (Efficient Java Matrix Library) 0.43** | `org.ejml:ejml-simple:0.43` | **Compact3-safe rt ✓** | Pure Java; compact matrices; recommended for state observers and own-Kalman implementations. Matrix inversions → BWorker. |
| **ND4J (N-Dimensional Arrays for Java)** | `org.nd4j:nd4j-native-platform:*` | **NOT viable on JACE — Supervisor/offline only** | JNI-backed native binaries (OS/arch-specific `.so`/`.dll`); incompatible with Compact3 ZRE on QNX7. Use on Supervisor or Windows offline analysis only. |
| **MiniPID** | GitHub only (no Maven Central release) | **Skip — prefer kitControl.BLoopPoint** | Simple pure-Java PID, but BLoopPoint already ships ramp anti-windup + bumpless transfer + loop alarms + priority-array output; add MiniPID only when BLoopPoint's fixed API is genuinely insufficient (e.g. serial commissioning, non-standard anti-windup shape). |
| **kitControl.BLoopPoint** | N4 framework (no Maven dep) | **Default choice for PID — always available** | Ramp anti-windup, bumpless transfer, four disable actions, direct/reverse, loop-deviation alarm. Prefer for all standard modulating loops. Third-party PID is the exception. |
| **Squirrel Foundation** | `org.squirrelframework:squirrel-foundation:0.3.10` | **Compact3: unverified — prefer plain BEnum pattern** | Reflection-heavy annotation processor; dynamic proxy generation may fail on Compact3 ZRE (no `java.beans`, limited `java.lang.reflect`). The proven N4 pattern for AHU/defrost/lead-lag state machines is a plain `BEnum` slot + engine-cycle if/else transitions (BColdRoom, BCompressorControl exemplars); adopt Squirrel only after a Compact3 smoke-test on the JACE target. |
| **Aviator** | `com.googlecode.aviator:aviator:5.x` | **Compact3: NOT proven — risk of `ClassLoader.defineClass`** | Expression compilers typically invoke `ClassLoader.defineClass` to emit bytecode at runtime; SecurityManager on Compact3 ZRE may deny this. Not proven safe on JACE. Supervisor use only until verified. |
| **MVEL2** | `org.mvel:mvel2:2.4.x` | **Compact3: NOT proven — same risk as Aviator** | Dynamic bytecode generation; same JACE caveats as Aviator. |
| **ScheduledExecutorService / Executors.*** | JDK (`java.util.concurrent`) | **NOT viable in BComponent** | SecurityManager denies `modifyThread`; throws at runtime. Use `Clock.schedule` / `Clock.schedulePeriodically` for time-triggered callbacks and `BWorker` for blocking off-thread work. |

---

## Proposed kit deltas (propose-never-apply)

| Δ | Delta | Target file / § | Token |
|---|-------|-----------------|-------|
| Δ1 | Create `types/math-control-libraries.md` — the library selection map for N4 rt module authors: (a) Compact3 gate explanation + how to verify (compile with `-profile compact3`, run on JACE; pure-Java/no-native = safe); (b) the verdict table above; (c) "prefer kitControl.BLoopPoint" doctrine — it ships with every N4 station, has ramp anti-windup + bumpless transfer + loop-deviation alarm; third-party PID only when genuinely needed; (d) engine-thread / BWorker rule: any optimization loop, matrix inversion, ODE step, or expression eval that takes >2 ms must be posted to a `BWorker` child — never inline in `execute()` or `changed()`; (e) plain BEnum state-machine pattern preferred over Squirrel Foundation for AHU/defrost/lead-lag; (f) Aviator/MVEL not proven on JACE; (g) ND4J Supervisor-only note. Cross-refs: resource-threading retro (BWORKER-OFFTHREAD1), spa-library-integration retro (frontend libs are separate). | NEW `types/math-control-libraries.md` | MATH-LIB-MAP1 |
| Δ2 | Add a one-line cross-ref in `types/logic.md` after the LC2 / BLoopPoint bullet pointing control-block authors to `types/math-control-libraries.md` for third-party library selection: "→ For third-party math/control library selection (Apache Commons Math, EJML, ND4J, MiniPID, state-machine libs) and the engine-thread/BWorker rule for heavy math: see `types/math-control-libraries.md`." | `types/logic.md` — after LC2 bullet (end of §kitControl patterns) | MATH-LOGIC-XREF1 |
| Δ3 | Cross-ref the resource-threading retro token BWORKER-OFFTHREAD1 from `math-control-libraries.md` §BWorker rule section and the token POLL-LIMITS1 for polling context; cross-ref the spa-library-integration retro explicitly noting it covers ux/wb frontend libs, not rt Java libs — so authors reading one retro find the other. (No new file — these cross-refs belong inside the new doc from Δ1.) | `types/math-control-libraries.md` §Cross-refs (within Δ1 doc) | MATH-XREF-THREADING1 |
| Δ4 | Add a `toolbelt/check-compact3.sh` sketch: `unzip -l <path-to-rt.jar> | grep ".class"` piped to a class-list check against the JDK compact3 manifest; or simpler: `javac -profile compact3 -cp ... *.java` compile gate that errors on excluded APIs. Document the recipe in `math-control-libraries.md` under the Compact3 gate section. (Concrete script authoring deferred to apply phase; this delta proposes the recipe and the pointer.) | `types/math-control-libraries.md` §Compact3 gate + `toolbelt/check-compact3.sh` (stub) | COMPACT3-GATE-SCRIPT1 |

---

## Lessons

1. The single most load-bearing rule is the **engine-thread / BWorker boundary**: Apache Commons Math
   and EJML are safe on JACE, but an optimization loop or ODE integration step running inline in
   `execute()` will stall the engine (B31 §31.1) and trigger the watchdog or GC amplification.  The
   library verdict and the threading discipline are inseparable — a doc that gives one without the
   other creates false confidence.

2. **BLoopPoint is the right default for PID**.  It ships with every N4 station, already has the
   correct anti-windup, bumpless transfer, ramp limiter, and loop-alarm integration.  The corpus
   (B539, B733) fully documents its internals.  A third-party PID adds a dependency, a JAR to
   bundle, and a Compact3 gate check — justified only for custom behavior BLoopPoint cannot provide.

3. **ND4J is a hard no on JACE**: its native-platform artifact pulls OS-specific JNI shared libs
   that are incompatible with Compact3 QNX7 ARM.  This is not a configuration issue; it is a
   binary incompatibility.  Any corpus block proposing ND4J for rt use should be read as
   Supervisor/offline only.

4. **The plain BEnum state machine is the proven N4 pattern** (BColdRoom, BCompressorControl
   already use it).  Squirrel Foundation is pure Java but its reflection + proxy generation is an
   unverified Compact3 risk; the existing exemplars demonstrate that engine-cycle if/else over a
   BEnum covers the AHU/defrost/lead-lag state machines encountered in refrigeration modules.

5. **Aviator/MVEL runtime expressions carry a Compact3 risk** that is not immediately obvious: the
   failure mode (SecurityManager denying `ClassLoader.defineClass`) happens at runtime on the JACE,
   not at compile time on the Supervisor build host.  The risk must be called out explicitly rather
   than letting authors assume "pure Java = safe".

6. B67's recommendation to add `commons-math3` to an analytics module is the strongest corpus
   evidence that Apache Commons Math 3.6.1 is viable for N4 rt use, even though the Analytics
   module itself did not ship it.  This makes it a RECOMMENDED library, not just a possible one.

---

**Status**: PENDING — INDEX row appended: `| 2026-09-18-math-control-library-layer-deltas.md | kit | 2026-09-18 | pending | 4 |`
