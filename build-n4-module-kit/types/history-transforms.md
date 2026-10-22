# History series transforms — transform graphs, nodes, aggregate functions

The `seriesTransform` module turns histories into derived series (roll-ups, sums, averages, scaled or
filtered series) through a **transform graph** of nodes that an integrator wires in Workbench. Before writing
custom Java to roll up energy or runtime, check whether a graph does it; write code only to add a missing
aggregate function or node. `[ev: retro t9c-reference-gaps Δ6]`

Derivations: corpus B1339 (the seriesTransform user guide and its query servlet), B45 (webChart and
seriesTransform). Official pages: devguide `seriesTransforms/` (`graphComponents`, `aggregateFunctions`,
`extendingGraphNode`, `nodeSchema`, `createTransformTable`, `creatingSeriesCursors`). There is no original
source in `docSource` for this module, so every `[CERT]` below is the 4.14.0.162 decompile:
`R=organized/seriesTransform/seriesTransform-rt` and `V=R/vineflower`, under niagara-research.

---

## 1 · Model
- **Module and dependency:** `seriesTransform` (`-rt`, `-ux`, `-wb`); the API is
  `javax.baja.seriestransform` (+ `.graph`), the built-in nodes and functions are `com.tridium.seriestransform.*`.
  [CERT] `R/extracted/META-INF/module.xml:2-91`. A module that extends it depends on `seriesTransform-rt` [INFER].
- **Graph:** `BTransformGraph extends BNodeContainer`; every node `extends BGraphNode` (a `BComponent`
  with `transformInputs`, `transformOutput`, `status`, `faultCause`) and is legal only under a
  `BNodeContainer`. [CERT] `V/javax/baja/seriestransform/graph/BGraphNode.java:75-81,313-315`
- **Built-in nodes** include source (`BHistorySourceNode`: `dataSource` ORD, `dateRange`, `srcSchema`,
  `quantInterval`), aggregate, rollup, scale, filter, cleanser, composite, timeshift. [CERT]
  `V/com/tridium/seriestransform/source/BHistorySourceNode.java:81-85`; `R/extracted/META-INF/module.xml:31-91`
- **Rollup:** `BRollupNode extends BAggregateNode` and adds `interval` (custom, fiveMin, fifteenMin,
  thirtyMin, hourly, daily, monthly, yearly); the aggregate node holds a `functionMap`. [CERT]
  `V/com/tridium/seriestransform/rollup/BRollupNode.java:30-31`; `.../rollup/BRollupIntervalEnum.java:12-31`
- **Schema:** `BSeriesSchema` (`javax.baja.seriestransform.graph`) lists the fields each record carries
  (`addField(name, BTypeSpec, BFacets)`, `getFieldNames`, `equivalent`); the key field is the one faceted
  `primaryKeyField=true`, else the timestamp. [CERT] `V/javax/baja/seriestransform/graph/BSeriesSchema.java:32-121`;
  [CERT-doc] `nh class BSeriesSchema`; `niagara-help/devguide-clean/seriesTransforms/nodeSchema.txt:8-24`
- **Output:** a node resolves to `BSeriesTransformTable` (a `BITable`) whose `SeriesCursor` yields records.
  [CERT] `V/javax/baja/seriestransform/BSeriesTransformTable.java:18`; `V/javax/baja/seriestransform/SeriesCursor.java:8`

## 2 · Running a graph
- **ORD:** `<graph ord>|transform:` — the `transform` scheme resolves through a local or Fox graph resolver;
  webChart builds exactly this ORD. [CERT] `R/extracted/META-INF/module.xml:30`;
  `V/com/tridium/seriestransform/servlets/SeriesTransformWebChartQueryServlet.java:149`
- **Java:** `BGraphNode.resolve(QueryParams, BOrd base, Context)` → `BSeriesTransformTable[]`. [CERT]
  `V/javax/baja/seriestransform/graph/BGraphNode.java:140-148`
- The browser query servlet path is still open (B1339-G1); charting output needs a timestamp and a
  non-String value and has no live mode `[ev: corpus B1339]`.

## 3 · Recipe — a custom aggregate function
Built-ins are exactly `FindAverage`, `FindMax`, `FindMin`, `SumSeries`, `PassThrough` — no count, first or
last, so "energy = last − first" needs a custom function. [CERT] `R/extracted/META-INF/module.xml:56-63`
1. Extend `javax.baja.seriestransform.graph.BTransformFunction` and implement its five abstract methods:
   `applyFunction(Map<String, List<? extends BComplex>> series, String[] srcProps, Property dst, BComplex result)`,
   `getFunctionName()`, `getFunctionDesc()`, `getReturnType()`, `getArgumentTypes()`. [CERT]
   `V/javax/baja/seriestransform/graph/BTransformFunction.java:31-54`; [CERT-doc]
   `niagara-help/devguide-clean/seriesTransforms/aggregateFunctions.txt:6-10`
2. Each source argument is `SeriesName.PropertyName`; use the helpers `getSeriesRecords`, `getSeriesName`,
   `getFieldName`, `getUnits`, `convertToUnits`, `setRecordValue`. [CERT] `BTransformFunction.java:56-140`
3. Declare it as an ordinary `@NiagaraType`: the Aggregate and Rollup nodes find every concrete subclass
   through the registry — no palette or agent entry. [CERT] `BTransformFunction.java:142-150`; [CERT-doc]
   `aggregateFunctions.txt:6`
4. A custom node instead extends `BGraphNode` and implements `doGetSchema()`, `doCheckSchema()` and
   `doResolve(...)`. [CERT-doc] `niagara-help/devguide-clean/seriesTransforms/extendingGraphNode.txt:13-50`

## 4 · Pitfalls
- **Keep `isCumulative()` true for roll-ups:** `BRollupNode` silently removes non-cumulative mappings
  (`PassThrough` is one). [CERT] `V/com/tridium/seriestransform/rollup/BRollupNode.java:84-95`
- **NaN and units:** `SumSeries` returns NaN when any operand is NaN or a unit conversion fails;
  `convertToUnits` throws on incompatible units. [CERT] `V/com/tridium/seriestransform/functions/BSumSeries.java:72-74,97-99`;
  `BTransformFunction.java:93-111`
- **Bucket alignment uses no explicit time zone:** `RollupCursor.makeBaselineTimestamp` builds buckets with
  `BAbsTime.make(y, m, d[, h, min])` [CERT] `V/com/tridium/seriestransform/rollup/RollupCursor.java:98-126`;
  that daily buckets follow the station's default zone, not the history's, is [INFER] — check it before
  reporting per-day energy across zones.
- **Schema drift:** when the input schema changes, `doCheckSchema` should throw `ConfigException` and
  fault the node. [CERT-doc] `extendingGraphNode.txt:50,74`
- Record status propagation through functions is not verified [INFER]; do not rely on it for alarms.
