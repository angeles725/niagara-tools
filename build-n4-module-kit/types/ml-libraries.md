# ML Library Selection and Safety Architecture for N4 Modules

> **Scope: Java ML libraries on the N4 JVM path.** This document covers tabular
> learners (Weka, Smile, Tribuo, Java-ML), deep-learning / advanced inference
> (DL4J, ONNX Runtime Java, XGBoost4J), the train-offline/infer-ONNX pattern,
> the PROPOSE-VALIDATE safety architecture, and data/operating-mode discipline.
>
> For math and PID libraries (Apache Commons Math, EJML, BLoopPoint, state
> machines, engine-thread/BWorker boundary): see `types/math-control-libraries.md`.
> For JAR packaging mechanics (Shadow plugin, shading, classloader model, signing):
> see `types/third-party-libraries.md`.

---

## Placement rule — Supervisor vs. JACE-rt `[ev: corpus B1023, B26, B31]`

| Tier | JVM | ML inference allowed? | Training allowed? |
|------|-----|-----------------------|-------------------|
| **Supervisor** (Linux x64 / Win x64) | Full JDK 8 SE + native | **Yes** — pure-Java AND JNI-backed libraries | Never on Supervisor at runtime — always OFFLINE |
| **JACE-rt** (QNX7 ARMv7, Compact3 JDK 8) | Compact3; no native | Pure-Java only; ≤ linear/logistic regression; must run on a BWorker | Never |

**Three hard constraints** that determine JACE viability:

1. **No native JNI on QNX7/ARM:** a Maven artifact bundling Linux x64 `.so`, Windows
   `.dll`, or macOS `.dylib` has NO QNX7/ARM binary. Libraries with JNI (ND4J,
   DL4J, ONNX Runtime Java, XGBoost4J) throw `UnsatisfiedLinkError` at runtime on
   the JACE — this is a binary incompatibility, not a configuration issue.
   `[ev: corpus B26 §26.7, §26.4]`

2. **Engine-thread budget < 10 ms:** even a pure-Java ML inference that allocates
   large intermediate arrays amplifies GC pause risk (ParallelGC Full GC:
   500 ms–2 s on heap pressure). Any model heavier than a simple linear/logistic
   regression must run on a `BWorker`, not inline in `execute()`.
   `[ev: corpus B31 §31.1, §31.3.8]`

3. **Compact3 JDK 8 ceiling:** libraries that require Java 11+ (Tribuo) or that
   call `ClassLoader.defineClass` dynamically at runtime may fail on the JACE even
   though they compile and pass the Supervisor build. `[ev: corpus B1023 §ND4]`

**Training is always OFFLINE** — historical data is exported from Niagara histories
(oBIX or Haystack export) and a Python/Java data-science environment trains and
exports the model. Only the deployed artifact (ONNX file + inference JAR) ships in
the module. `[ev: corpus B67 §67.7.2]`

**The JACE gets only validated proposals:** the Supervisor (or a BWorker thread on
the Supervisor) runs inference, then the result passes through the PROPOSE-VALIDATE
gate before reaching any actuator or control core. `[ev: corpus B821, B733 §733.2]`

---

## Library verdict table `[ev: corpus B67, B26, B1023, B31, B617]`

| Library | Maven artifact | Java 8? | Pure Java? | Compact3-safe rt? | Supervisor-only? | License | N4 verdict |
|---------|----------------|---------|------------|-------------------|-----------------|---------|------------|
| **Weka 3.8.x** | `nz.ac.waikato.cms.weka:weka-stable:3.8.6` | ✓ | ✓ | Yes — core algos; shade to isolate (GUI packages are absent on Compact3; keep to core tabular path) | No (core; see license) | **GPL-3.0** ⚠ | Viable for internal / prototype use. **NOT safe for a distributed closed-source module** — GPL-3.0 contaminates the distribution. If Weka is used for training offline, it never ships in the module and GPL is confined to the dev environment. Bundle + shade under project package if used at runtime. |
| **Smile 2.6.x** | `com.github.haifengl:smile-core:2.6.0` | ✓ | ✓ (tabular algos; avoid `smile-nlp` which has native deps) | Yes — core tabular algos (Random Forest, Logistic, k-means, SVM, LDA) are pure Java | No (tabular core) | Apache 2.0 | **Preferred tabular ML library** — Apache 2.0, Java 8 compat, rich algorithm set. **Pin to 2.x — Smile 3.x+ requires Java 11+.** Bundle + shade. Use on BWorker for any model heavier than logistic regression. |
| **Tribuo (Oracle)** | `org.tribuo:tribuo-all:4.x` | **No — Java 11+** | ✓ | **No** | N/A | Apache 2.0 | **NOT viable** — requires Java 11+; both Supervisor and JACE are Java 8 only. `[ev: B1023 §ND4]` |
| **Java-ML** | Various (no Maven Central) | ✓ | ✓ | Technically yes | No | GPL-2.0 | **Avoid** — stale (last meaningful release ~2012), no Maven Central, GPL-2.0. Use Smile 2.x instead. |
| **Deeplearning4j (DL4J) 1.0-M2.1+** | `org.deeplearning4j:deeplearning4j-core:*` | ✓ (API) | **No — JNI (ND4J native backend)** | **No — native** | **Supervisor-only** | Apache 2.0 | JNI-backed via ND4J native platform artifact; `UnsatisfiedLinkError` on JACE QNX7/ARM `[ev: B26]`. Supervisor use only for training or inference. Recommended by B67 §67.7.2 for a `BNeuralNetInferenceBlock`-style family; or prefer ONNX Runtime Java (lighter, license-risk-free). |
| **ONNX Runtime Java** | `com.microsoft.onnxruntime:onnxruntime:1.x` | ✓ (API) | **No — JNI** | **No — native** | **Supervisor-only** | MIT | JNI-backed; `UnsatisfiedLinkError` on JACE QNX7/ARM `[ev: B26]`. **Preferred inference path for offline-trained Python models (sklearn/XGBoost/PyTorch → ONNX export → inference on Supervisor).** GPL training libraries (Weka/sklearn) never ship in the module — only the ONNX file and ONNX Runtime JAR. |
| **XGBoost4J** | `ml.dmlc:xgboost4j:1.x` | ✓ (API) | **No — JNI** | **No — native** | **Supervisor-only** | Apache 2.0 | JNI-backed; same QNX7/ARM incompatibility `[ev: B26]`. Train XGBoost offline in Python, export to ONNX, run inference via ONNX Runtime Java on Supervisor. |

---

## Train-offline / infer-ONNX pattern

The recommended pattern for any model beyond a simple linear regression on an N4 Supervisor module:

```
OFFLINE (Python / data science environment)
  1. Collect historical data from Niagara histories (oBIX or Haystack export)
  2. Separate by operating mode (off / startup / normal / defrost / fault) — see Data Discipline
  3. Use time-ordered train / test split (NEVER random split — avoids future leakage)
  4. Train: sklearn.ensemble.RandomForestRegressor / XGBRegressor / LogisticRegression / KMeans
  5. Export: skl2onnx / onnxmltools → model.onnx file

DEPLOY to Supervisor module
  6. Bundle model.onnx as a module resource (rc/ folder or embedded in the JAR)
  7. Load at component started():
         OrtEnvironment env = OrtEnvironment.getEnvironment();
         OrtSession session = env.createSession("model.onnx", ...)
  8. On each execute() cycle or BWorker tick:
         build float[] features → OnnxTensor → session.run() → float[] scores
  9. Feed score / setpoint to the PROPOSE-VALIDATE safety gate — NEVER directly to output
```

**Key advantage:** the training library (sklearn / XGBoost / Weka) never ships in
the deployed module. Only the ONNX Runtime JAR (~15 MB) and the model file ship.
GPL exposure from Weka or sklearn is confined to the development environment.

For a JACE-rt module that needs lightweight inference (e.g. anomaly score ≤ a
linear model), inference must be pure-Java (Smile 2.x linear/logistic) or a
hand-rolled lookup table — ONNX Runtime Java cannot run on the JACE.

---

## PROPOSE-VALIDATE safety architecture `[ev: corpus B821, B733 §733.2, B819]`

**Rule: the model PROPOSES; deterministic logic VALIDATES and LIMITS; the model
NEVER writes directly to a valve, VFD, compressor command, or staging slot.**

This is an extension of the kit's HOA / interlock / fallback doctrine
(`types/logic.md` §Safety fail-modes, §Staging & interlocks, §Zero-demand / idle state).

```
Sensor data (temp, pressure, amps, kW, occupancy, hour-of-day, day-of-week,
  compressor state, fan speed %, valve %, outdoor temp/humidity)
              ↓
  Feature extraction + quality gate
    • BStatus.isValid() on every input slot
    • Discard NaN / ±Infinity (Double.isFinite)
    • Operating-mode gate: skip if station is in DEFROST / FAULT / STARTUP
      (these modes were excluded from training data — model output is undefined)
              ↓
  ML Model inference (Supervisor: ONNX Runtime Java / Smile 2.x;
                       JACE-rt: Smile 2.x linear/logistic only)
              ↓
  PROPOSAL: float setpoint / stage count / anomaly score
              ↓
  ┌────────────────── VALIDATION GATE (deterministic) ──────────────────┐
  │  1. Range clamp: setpoint within [physicalMin, physicalMax]         │
  │  2. Rate-of-change limit: |Δsetpoint| ≤ maxStepPerCycle            │
  │  3. Anti-short-cycle guard: minOn / minOff timers apply             │
  │  4. HOA-OFF lockout: if HOA == OFF, proposal is REJECTED            │
  │  5. Interlock check: compressor cannot stage while in DEFROST       │
  │  6. Fallback: if model output is NaN or gate fails →                │
  │               use fixed default setpoint (SUMMARY|OPERATOR config)  │
  └─────────────────────────────────────────────────────────────────────┘
              ↓ VALIDATED setpoint / stage command
  Deterministic control core (kitControl.BLoopPoint or staging logic)
              ↓
  Actuator output (valve %, VFD speed %, compressor on/off)
```

**The model is an INPUT to the control system, not a replacement for it.**
Every protection layer that applies to a manually-set setpoint also applies to
an ML-proposed one. An ML proposal that bypasses HOA-OFF lockout, anti-short-cycle
timers, range clamps, or sensor-validity gates is a safety defect, not a feature.

Cross-ref: `types/logic.md` §Safety fail-modes (HOA/interlock doctrine),
§Staging & interlocks (anti-short-cycle, FIFO interlock queue),
§Zero-demand / idle state (demand gate). `[ev: corpus B821, B733, B819]`

---

## Data discipline

Correct data preparation prevents a model that validates well offline but causes
erratic behavior in production.

### Operating-mode separation (mandatory before training)

Separate raw history rows by operating mode BEFORE any ML pipeline. Train only on
NORMAL mode rows. Other modes produce non-stationary distributions that contaminate
the model — a model trained on DEFROST rows learns associations that only hold
during defrost and then applies them during normal cooling.

| Mode | Source slot / condition | Action |
|------|-------------------------|--------|
| OFF | station.running == false | Exclude from training |
| STARTUP | first N minutes after running → true | Exclude |
| NORMAL | all protections OK, steady-state | Train on this |
| DEFROST | inDefrost == true | Exclude |
| FAULT | any FAULT bit in BStatus | Exclude |
| INVALID | any NULL / STALE bit on key inputs | Exclude |

### Time-ordered train / test split (mandatory — NEVER random split)

Use a chronological split (e.g. first 80 % of time range for training, last 20 %
for testing). A random split leaks future data into the training set and inflates
test metrics that do not hold in production. This is the single most common
data-preparation mistake on time-series data.

### Feature checklist (HVAC / refrigeration)

outdoor dry-bulb temperature [°C], outdoor relative humidity [%], return-air
temperature, supply-air temperature, zone occupancy (0/1), hour-of-day (0–23),
day-of-week (0–6), compressor staging state (0/1/2/3), fan speed (%),
liquid-line valve position (%), site power draw (kW),
defrost-time-since-last (minutes, NORMAL mode only).

### Timestamp and scale discipline

- All rows must carry a station-local timestamp (`BAbsTime`) resolved from the N4
  history; do not use wall-clock capture time.
- Normalize or standardize features before SVM / logistic / neural network models
  (required); optional for tree-based models.
- Include `BStatus` bits as an additional boolean feature, or use them strictly as
  a filter gate (exclude invalid rows from training).

---

## Cross-refs

- **Math / PID / control library selection (Apache Commons Math, EJML, BLoopPoint,
  state machines, engine-thread / BWorker boundary):** `types/math-control-libraries.md`.
  That document covers the non-ML math layer; this document covers the ML / predictive
  model layer. The two serve different authors (control-math vs. predictive-model).
- **JAR packaging mechanics (Shadow plugin, shading, classloader model, signing,
  Compact3 SE-API gate):** `types/third-party-libraries.md`.
- **HOA / interlock / fallback doctrine (PROPOSE-VALIDATE targets):**
  `types/logic.md` §Safety fail-modes, §Staging & interlocks, §Zero-demand / idle state.
- **BWorker for heavy inference off the engine thread:**
  `types/math-control-libraries.md` §Engine-thread / BWorker rule and
  `types/logic.md` §Off-engine blocking I/O.
