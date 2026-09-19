<!-- review-status: folded -->
# 2026-09-18 · kit · ml-libraries-deltas

**Session**: corpus-mining for build-n4-module (operator: ML libraries for N4).
**Delta count**: 4

---

## What happened

Mined corpus blocks B67 (Analytics ML verdict + DL4J/ONNX recommendation), B1023 (JACE Compact3
ZRE distribution — `azul-zre-compact3-qnx7-arm`), B26 (QNX7 ARM native-lib constraint), B31
(JACE engine-thread/watchdog/GC), B617 (ModuleClassLoader topology; native JNI → Supervisor),
and B733 (kitControl modulating-control doctrine) to answer: which ML libraries are usable from
an N4 module, on what tier, and what safety architecture gates their output.

The prior math-control-libraries retro (2026-09-18-math-control-library-layer-deltas.md) covers
Apache Commons Math, EJML, BLoopPoint, and the engine-thread/BWorker boundary.  This retro covers
the ML-specific layer — tabular learners (Weka, Smile, Tribuo, Java-ML), deep-learning / advanced
(DL4J, ONNX Runtime Java, XGBoost4J), the train-offline/infer-ONNX pattern, the PROPOSE-VALIDATE
safety doctrine, and data/operating-mode discipline.  There is no overlap with that retro; the two
docs serve different authors (control-math vs. predictive-model authors).

**Key finding**: no existing kit file covers ML library selection or the safety architecture that
gates ML proposals on a BAS. The corpus (B67 §67.7.2) confirms that even Tridium's own Analytics
module ships zero ML — it recommends DL4J or ONNX Runtime Java as an add-on (P2 priority).  This
retro proposes the authoritative placement rule, library verdict table, safety architecture doc,
and cross-refs so the kit answers the ML question without duplicating the math-libraries doc.

---

## Evidence

### Placement rule — Supervisor vs. JACE rt

- **B1023 §ND4** — JACE ships `azul-zre-compact3-qnx7-arm` (Azul JDK 8 Compact3, QNX7 ARMv7).
  Compact3 excludes AWT/Swing, JDBC, CORBA; includes core `java.util`, `java.util.concurrent`,
  `java.math`.  Pure-Java libraries that stay within these packages are class-loadable on JACE.
  `[ev: corpus B1023]`
- **B26 §26.7 + §26.4** — JACE is QNX7 ARM EABI5.  A Maven artifact bundling native `.so` files
  (Linux x64, Windows DLL, macOS dylib) has NO QNX7/ARM binary.  Any library with JNI — ND4J,
  DL4J, ONNX Runtime Java, XGBoost4J — throws `UnsatisfiedLinkError` on the JACE at runtime.
  This is a binary incompatibility, not a configuration issue.  `[ev: corpus B26]`
- **B31 §31.1 + §31.3.8** — JACE engine thread must complete each `execute()` cycle in <10 ms
  (watchdog kills the component on overrun); ParallelGC (JDK 8 default) on the JACE causes
  stop-the-world Full GC pauses of 500 ms–2 s under heap pressure.  Even a pure-Java ML inference
  that allocates large intermediate arrays amplifies GC pause risk.  Any model heavier than a
  simple linear/logistic regression must be posted to a `BWorker` or run on the Supervisor.
  `[ev: corpus B31]`
- **B617 §617.5** — boot/NRE parent is tried FIRST in the module classloader delegation chain.
  If the NRE carries a conflicting class, the platform copy wins.  Shade (relocate) bundled
  libraries to a private package to avoid shadowing.  `[ev: corpus B617]`

### Analytics module verdict (B67)

- **B67 §67.7.2** — Analytics module ships ZERO ML: no imports of
  tensorflow/deeplearning4j/weka/smile/commons-math.  Tridium explicitly recommends adding
  DeepLearning4J 1.0-M2.1+ or ONNX Runtime Java as a P2 optional addition for a
  `BNeuralNetInferenceBlock`-style family.  The absence is a deliberate deferral, not a
  technical blocker — the module team assessed these libraries and found them viable (DL4J/ONNX)
  for Supervisor use.  `[ev: corpus B67 §67.7.2]`
- **B67 §67.7.2** — also lists `commons-math3` as the recommended stats library for percentile/
  variance/skewness extensions (confirmed viable; see math-control-libraries retro for that
  verdict). `[ev: corpus B67]`

### kitControl safety doctrine cross-reference

- **B733 §733.2–733.3** — kitControl modulating-control blocks (`BLoopPoint`, `BQuadMath`, etc.)
  run on the engine thread.  They are the validated output stage; an ML-proposed setpoint feeds
  INTO `BLoopPoint` as a `setpoint` slot value, never directly to a valve or VFD command.
  `[ev: corpus B733]`
- **B821** — protection anatomy of our RT modules: HOA-OFF lockout > demand gate > staging logic.
  An ML-derived stage request is a DEMAND signal, subject to: HOA-OFF lockout, anti-short-cycle
  timers, range clamps, and sensor-validity gates — exactly as a manual setpoint would be.
  Any ML proposal that bypasses these layers is a safety defect, not a feature.
  `[ev: corpus B821]`

---

## Library verdict table

| Library | Maven artifact | Java 8? | Pure Java? | Compact3-safe rt? | Supervisor-only? | License | N4 verdict |
|---------|---------------|---------|------------|-------------------|-----------------|---------|------------|
| **Weka 3.8.x** | `nz.ac.waikato.cms.weka:weka-stable:3.8.6` | ✓ | ✓ | Yes (core; shade if AWT code paths touch GUI classes — GUI packages are not on Compact3) | No (core algos Compact3-safe; but see license) | **GPL-3.0** ⚠ | Viable for internal / prototype use. **NOT safe for a distributed closed-source module** — GPL contaminates the distribution. Call it out explicitly. Bundle + shade under project package to isolate. |
| **Smile 2.6.x** | `com.github.haifengl:smile-core:2.6.0` | ✓ | ✓ (tabular algos; NLP sub-modules use native) | Yes — core tabular algos (RF, logistic, k-means, SVM, LDA) are pure Java; avoid `smile-nlp` (native) | No (tabular core) | Apache 2.0 | Preferred tabular ML library — Apache 2.0 license, Java 8 compat, rich algorithm set. Note: **Smile 3.x+ requires Java 11+** — pin to 2.x on Java 8 targets. |
| **Tribuo (Oracle)** | `org.tribuo:tribuo-all:4.x` | **No — Java 11+** | ✓ | **No** | N/A | Apache 2.0 | **NOT viable** — requires Java 11+; JACE is Java 8 only. |
| **Java-ML** | Various (no Maven Central release) | ✓ | ✓ | Technically yes | No | GPL-2.0 | **Avoid** — stale (last meaningful release ~2012), no Maven Central, GPL, no maintained documentation. Use Smile 2.x instead. |
| **Deeplearning4j (DL4J) 1.0-M2.1+** | `org.deeplearning4j:deeplearning4j-core:*` | ✓ (API) | **No — JNI (ND4J native backend)** | **No — native** | **Supervisor-only** | Apache 2.0 | JNI-backed via ND4J native platform artifact; `UnsatisfiedLinkError` on JACE QNX7/ARM `[ev: B26]`. Use only on Supervisor for training or inference; or skip entirely in favor of ONNX Runtime Java. Recommended by B67 §67.7.2 for a `BNeuralNetInferenceBlock` on Supervisor. |
| **ONNX Runtime Java** | `com.microsoft.onnxruntime:onnxruntime:1.x` | ✓ (API) | **No — JNI** | **No — native** | **Supervisor-only** | MIT | JNI-backed; `UnsatisfiedLinkError` on JACE QNX7/ARM `[ev: B26]`. The preferred pattern for deploying an offline-trained Python model (sklearn/XGBoost/PyTorch → ONNX export → inference on Supervisor). **License risk-free in the deployed module** because the heavy training lib (Weka/sklearn) never ships — only the ONNX file and the ONNX Runtime inference JAR. |
| **XGBoost4J** | `ml.dmlc:xgboost4j:1.x` | ✓ (API) | **No — JNI** | **No — native** | **Supervisor-only** | Apache 2.0 | JNI-backed; same QNX7/ARM incompatibility `[ev: B26]`. Train XGBoost offline in Python, export to ONNX, run inference via ONNX Runtime Java on Supervisor. |

---

## Train-offline / infer-ONNX pattern

This is the recommended pattern for deploying any model beyond a simple linear regression
on an N4 Supervisor module:

```
OFFLINE (Python / data science environment)
  1. Collect historical data from Niagara histories (oBIX or Haystack export)
  2. Separate by operating mode (off / startup / normal / defrost / fault) — see Data Discipline
  3. Use time-ordered train / test split (NEVER random split — avoids future leakage)
  4. Train: sklearn.ensemble.RandomForestRegressor / XGBRegressor / LogisticRegression / KMeans
  5. Export: skl2onnx / onnxmltools → model.onnx file

DEPLOY to Supervisor module
  6. Bundle model.onnx as a module resource (rc/ folder or embedded in the JAR)
  7. Load at component started(): OrtEnvironment env = OrtEnvironment.getEnvironment();
                                  OrtSession session = env.createSession("model.onnx", ...)
  8. On each execute() cycle: build float[] features → OnnxTensor → session.run() → float[] scores
  9. Feed score/setpoint to the PROPOSE-VALIDATE safety gate (see below) — NEVER directly to output
```

The key advantage: the **training library (sklearn/XGBoost)** never ships in the module.
Only the ONNX Runtime JAR (~15 MB) and the model file ship.  GPL exposure from Weka/sklearn
is confined to the development environment, not the deployed artifact.

For a JACE-rt module that needs lightweight inference (e.g. anomaly score ≤ linear model),
the same pattern applies except inference must be pure-Java (Smile 2.x linear/logistic) or a
hand-rolled lookup table — ONNX Runtime Java cannot run on the JACE.

---

## PROPOSE-VALIDATE safety architecture

**Rule**: the model PROPOSES; deterministic logic VALIDATES and LIMITS the proposal; the model
NEVER writes directly to a valve, VFD, compressor command, or staging slot.

This is a direct extension of the kit's HOA/interlock/fallback doctrine (B821, types/logic.md).

```
Sensor data ──────────────────────────────────────────────────────────┐
  (temp, pressure, amps, kW, occupancy, hour-of-day, day-of-week,     │
   compressor state, fan speed %, valve %, outdoor temp/humidity)      │
                ↓                                                       │
  Feature extraction + quality gate                                    │
    • BStatus.isValid() on every input slot                            │
    • Discard NaN / ±Infinity (Double.isFinite)                        │
    • Operating-mode gate: skip if station is in DEFROST / FAULT /     │
      STARTUP (see Data Discipline — these modes were excluded         │
      from training data, so model output is undefined)                │
                ↓                                                       │
  ML Model inference (Supervisor: ONNX Runtime / Smile 2.x;           │
                       JACE-rt: Smile 2.x linear/logistic only)        │
                ↓                                                       │
  PROPOSAL: float setpoint / stage count / anomaly score               │
                ↓                                                       │
  ┌─────────────────── VALIDATION GATE (deterministic) ─────────────┐ │
  │  1. Range clamp: setpoint within [physicalMin, physicalMax]      │ │
  │  2. Rate-of-change limit: |Δsetpoint| ≤ maxStepPerCycle         │ │
  │  3. Anti-short-cycle guard: minOn / minOff timers apply          │ │
  │  4. HOA-OFF lockout: if HOA == OFF, proposal is REJECTED         │ │
  │  5. Interlock check: compressor cannot stage while in DEFROST    │ │
  │  6. Fallback: if model output is NaN or gate fails → use fixed   │ │
  │     default setpoint (operator-configurable SUMMARY|OPERATOR)   │ │
  └──────────────────────────────────────────────────────────────────┘ │
                ↓ VALIDATED setpoint / stage command                    │
  Deterministic control core (kitControl.BLoopPoint or staging logic)  │
                ↓                                                        │
  Actuator output (valve %, VFD speed %, compressor on/off)            │
```

**The model is an INPUT to the control system, not a replacement for it.**
Every protection layer that applies to a manually-set setpoint also applies to an ML-proposed one.

Cross-ref: `types/logic.md` §Safety fail-modes (HOA/interlock doctrine), §Staging & interlocks
(anti-short-cycle, FIFO interlock queue), §Zero-demand / idle state (demand gate).

---

## Data discipline

Correct data preparation prevents a model that looks good in offline validation but causes
erratic behavior in production.

**Operating-mode separation (mandatory before training)**:
Separate raw history rows by operating mode BEFORE any ML pipeline.  Train only on NORMAL mode
rows.  The other modes produce non-stationary distributions that contaminate the model:

| Mode | Source slot/condition | Action |
|------|-----------------------|--------|
| OFF | station.running == false | Exclude from training |
| STARTUP | first N minutes after running → true | Exclude |
| NORMAL | all protections OK, steady-state | Train on this |
| DEFROST | inDefrost == true | Exclude |
| FAULT | any FAULT bit in BStatus | Exclude |
| INVALID | any NULL/STALE bit on key inputs | Exclude (null/stale sensors) |

**Time-ordered train / test split (mandatory — NEVER random split)**:
Use a chronological split (e.g. first 80% of time range for training, last 20% for testing).
A random split leaks future data into the training set and inflates test metrics.

**Feature checklist (HVAC / refrigeration)**:
outdoor dry-bulb temperature [°C], outdoor relative humidity [%], return-air temperature,
supply-air temperature, zone occupancy (0/1), hour-of-day (0–23), day-of-week (0–6),
compressor staging state (0/1/2/3), fan speed (%), liquid-line valve position (%),
site power draw (kW), defrost-time-since-last (minutes, NORMAL mode only).

**Timestamp and scale discipline**:
- All rows must carry a station-local timestamp (BAbsTime) resolved from the N4 history;
  do not use wall-clock capture time — history rows carry the sample time.
- Scale consistently: normalize or standardize before tree-based models are optional; required
  for SVM / logistic / neural networks.
- Point-quality column: include BStatus bits as an additional boolean feature or use them
  strictly as a filter gate (exclude invalid rows from training).

---

## Proposed kit deltas (propose-never-apply)

| Δ | Delta | Target file / § | Token |
|---|-------|-----------------|-------|
| Δ1 | Create `types/ml-libraries.md` — the authoritative ML layer guide for N4 module authors: (a) PLACEMENT RULE: ML model inference belongs on the Supervisor (full JVM, native-capable, memory); the JACE-rt gets only validated setpoint/score proposals; training is always OFFLINE; (b) the library verdict table from this retro (Weka/Smile/Tribuo/Java-ML/DL4J/ONNX-Runtime-Java/XGBoost4J with Java8/native/Compact3/license columns); (c) the train-offline/infer-ONNX pattern as a step-by-step recipe; (d) the PROPOSE-VALIDATE safety architecture ASCII diagram + the rule "model is an input, never a bypass"; (e) data discipline: operating-mode separation table, time-ordered split, feature checklist, timestamp/scale/quality rules; (f) license note: Weka GPL contaminates a distributed module — use Smile 2.x or the ONNX pattern; (g) cross-refs: math-control-libraries.md (Apache Commons Math/EJML/BLoopPoint for non-ML math), resource-threading-consumption-deltas (BWorker for heavy inference), third-party-libraries.md (packaging/classloader/signing mechanics), types/logic.md §Safety (HOA/interlock/fallback doctrine). | NEW `types/ml-libraries.md` | ML-LIB-MAP1 |
| Δ2 | Add one cross-ref line in `types/logic.md` §Safety fail-modes, after the "Alarm-limit slots… alarms NOTIFY, they never STOP control" bullet: "→ When an ML model proposes a setpoint or stage command, the same doctrine applies: wrap the proposal in the PROPOSE-VALIDATE gate (range clamp, rate limit, anti-short-cycle, HOA-OFF lockout, fallback-on-NaN) before it reaches the control core. See `types/ml-libraries.md` for the full safety architecture and the library selection map." | `types/logic.md` — §Safety fail-modes, after the alarms-NOTIFY bullet | ML-LOGIC-SAFETY-XREF1 |
| Δ3 | Add ML library rows to the viability table in `types/third-party-libraries.md` with a pointer to `types/ml-libraries.md` for the full placement rule and data/safety doctrine. Rows to add: Weka 3.8.x (GPL, Compact3-safe core, license-blocked for distributed), Smile 2.6.x (Apache 2.0, Java 8 ✓, tabular ML, Supervisor+rt), DL4J (JNI, Supervisor-only, Apache 2.0), ONNX Runtime Java (JNI, Supervisor-only, MIT), XGBoost4J (JNI, Supervisor-only, Apache 2.0). Append note under table: "ML libraries: see `types/ml-libraries.md` for placement rule (Supervisor vs. JACE-rt), safety architecture, and data discipline." | `types/third-party-libraries.md` — Library viability table + footer note | ML-TPL-TABLE-XREF1 |
| Δ4 | Add a bidirectional cross-ref between `types/ml-libraries.md` (Δ1) and `types/math-control-libraries.md` (prior retro MATH-LIB-MAP1): in the ML doc, note that Apache Commons Math / EJML / BLoopPoint are the math and PID layer (see math-control-libraries.md); in the math doc, note that tabular ML classifiers/regressors and ONNX inference are a distinct concern covered in ml-libraries.md. This prevents authors from reading one and missing the other. (No new file — two one-line cross-ref inserts.) | `types/ml-libraries.md` §Cross-refs (within Δ1); `types/math-control-libraries.md` §Cross-refs | ML-MATH-XREF-BIDIR1 |

---

## Lessons

1. **The placement rule is the most important thing to get right.** ML inference that can run
   on the Supervisor (ONNX, DL4J) must NOT be assumed viable on the JACE-rt — the QNX7/ARM JNI
   incompatibility is a hard binary fault, not a configuration gap `[ev: B26]`.  An author who
   bundles ONNX Runtime Java into a `-rt` module will not see the failure until it loads on a
   live JACE.

2. **GPL from Weka is a real license risk for a distributed closed module.**  Pure-Java + Java 8
   compatible does not mean safe to ship.  The retro calls this out explicitly because it is
   easy to prototype with Weka and then accidentally ship a GPL-contaminated distribution.
   The Smile 2.x + Apache 2.0 path or the ONNX-inference pattern are the correct alternatives.

3. **The PROPOSE-VALIDATE gate is non-negotiable safety doctrine.**  An ML-derived setpoint is
   still a setpoint — it is subject to every HOA/interlock/anti-short-cycle rule in the kit.
   The model never replaces deterministic protection; it is a smarter source of the setpoint
   input.  This aligns with the existing HOA/fallback doctrine in `types/logic.md` and B821.

4. **Operating-mode separation before training is the single biggest data-quality mistake.**
   A model trained on DEFROST and FAULT rows alongside NORMAL rows learns associations that
   do not hold in normal operation — the model will propose setpoints that mimic fault-mode
   behavior.  The mode-separation table is a mandatory preprocessing step, not an optimization.

5. **Time-ordered train/test split is mandatory.**  A random split on time-series data leaks
   future readings into the training set and produces overly optimistic validation metrics that
   do not hold in production.  This mistake is common enough that it belongs in the kit doc, not
   just in a data-science tutorial.

6. **Tribuo requires Java 11+** — this is not obvious from its marketing.  The JACE is Java 8
   only.  Any author reading Oracle's Tribuo docs without checking the Java version requirement
   will waste time on a dependency that throws `UnsupportedClassVersionError` on deploy.

7. **B67 §67.7.2 is the best corpus evidence for the ML library recommendations** — it shows
   that Tridium's own Analytics team assessed these libraries and chose DL4J/ONNX as the P2
   add-on path.  The absence of ML in Analytics is a deliberate priority decision, not a
   technical impossibility.  This corpus reference is the authority behind the DL4J/ONNX
   recommendations in this retro.

---

**Status**: PENDING — INDEX row appended: `| 2026-09-18-ml-libraries-deltas.md | kit | 2026-09-18 | pending | 4 |`
