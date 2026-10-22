<!-- review-status: pending -->
<!-- module: kit -->
<!-- deltas: 6 -->
# 2026-10-16 · kit · n4-course-handson-slotomatic-authoring

**Session**: hands-on Certification Exam P1 (`BLimiter`) and P2 (`BHistoryExportService`), Bonus Lab Part 2 (agent point manager), and the course palettes, built with slot-o-matic and run live on 4.15.6.20.
**Delta count**: 6

## What happened
Writing new classes by hand for the exam exposed four slot-authoring behaviours the kit states wrongly or not at
all: pre-written empty generated-region markers make slot-o-matic skip the file while the build blames a missing
slot; a `BDouble`-typed property yields primitive accessors; the Workbench shows slots in annotation declaration
order, which led the operator to type the bounds into the wrong fields and pin the limiter output; and the
`module-include.xml` entry of a type does not pick up an `@AgentOn` added later. Running the kit gate on the
course jars also showed the palette check miscounting palettes saved by Workbench.

## Evidence
- [CERT-live] Exam P1 build, `cmd.exe /c "gradlew.bat :Cristian_Angeles_Cert-rt:slotomatic :Cristian_Angeles_Cert-rt:jar"`:
  the only errors shown are `error: Slot with name max not found on class com.certexam.Cristian_Angeles_Cert.BLimiter; have you run slot-o-matic?`
  (same for `min`), `Execution failed for task ':Cristian_Angeles_Cert-rt:compileJava'`, `2 actionable tasks: 2 executed`
  (slot-o-matic counted as executed, not failed). A `--info` run of `:slotomatic` alone shows the cause:
  `Could not process …\BLimiter.java: Found multiple metadata blocks` (transcript 2026-10-16T18:27).
  The class had been written with the empty `BEGIN/END BAJA AUTO GENERATED` region markers; written without them,
  slot-o-matic inserted the block and the build passed `[ev: engram obs 11478]`.
- [CERT] Kit K15 (`METHODOLOGY.md:122`) says "an empty skeleton … has no generated region; building it triggers
  'Found multiple metadata blocks'" — the opposite of the mechanism recorded in its own source,
  `retros/2026-09-05-campaign7-scaffold.md:26-31` ("An empty `//region … //endregion` skeleton causes … the `/*+`
  on both region and endregion lines looks like two distinct blocks"). Live evidence matches the retro, not the K15 line.
- [CERT-live] `@NiagaraProperty(name="out", type="BDouble", defaultValue="BDouble.DEFAULT", …)` generated
  `newProperty(Flags.SUMMARY | Flags.READONLY, BDouble.DEFAULT.as(BDouble.class).getDouble(), null)`,
  `public double getOut() { return getDouble(out); }`, `public void setOut(double v) { setDouble(out, v, null); }`
  (`CertificationExam/…/BLimiter.java:53-65`).
- [CERT-live] `BLimiter` declares `out`, `in`, `max`, `min` in that order (`BLimiter.java:17-38`); slot-o-matic
  emits them in the same order (:46-136) and the Wiresheet showed Max above Min; the operator entered the values
  by position and swapped them once `[ev: engram obs 11478]`. With Max=25 / Min=75 the clamp
  (`BLimiter.java:165-176`, min test first) returns 75 for every input below 75 (live 2026-10-16).
- [CERT] `toolbelt/schema-risk.sh:212` already classifies a same-name-set declaration reorder as a schema change,
  so slot order has to be right at first authoring.
- [CERT-live] `BLimiter.changed()` opens with `if (!isRunning()) return;` and dispatches on
  `in.equals(property) || min.equals(property) || max.equals(property)` (`BLimiter.java:154-159`); the writes to
  `out` re-enter `changed()` and fall through. Verified live in the exam station `[ev: engram obs 11009]`.
- [CERT-live] Bonus Lab Part 2: adding `@NiagaraType(agent = @AgentOn(types = {"trafficLightDriver:TrafficLightDriverPointDeviceExt", "trafficLightDriver:TrafficLightDriverPointFolder"}))`
  to an existing `BTrafficLightDriverPointManager` did not change its existing `module-include.xml` line; the file
  was deleted so the build regenerated it `[ev: engram obs 11479]`. The regenerated
  `trafficLightDriver-wb/module-include.xml` carries `<agent><on type="…PointDeviceExt"/><on type="…PointFolder"/></agent>`
  and the manager appeared as the view on both types (live).
- [CERT] `toolbelt/lint-agent-on-shape.sh` checks only that each `<on type>` is `module:Type`-shaped; nothing
  compares `module-include.xml` agents with the source `@AgentOn`.
- [CERT-live] `verify-module.sh` on `training-rt.jar` and `Cristian_Angeles_Cert-rt.jar` → `WARN palette … empty palette but module declares 15 type(s)` / `2 type(s)`.
  The palettes hold 10 and 2 component entries written as `<p m="t=training" t="t:FirstComponent" n="FirstComponent" />`
  (Workbench-saved form, `bajaObjectGraph version="4.0"`, `n=` last); `check_palette` counts only `<p n=`
  (verify-module.sh:420). Both palettes were used live to drop components. The same jars also WARN `palette-root`
  for `t="b:Folder"`; the wizard's driver palette (`devkit-wb.jar!gradle/ndriver/module.palette.vm`) emits
  `t="b:UnrestrictedFolder"` and both wizard-generated driver palettes PASS.

## Proposed kit deltas (propose-never-apply)
| Δ | Delta | Target file / § | Token |
|---|---|---|---|
| Δ1 | REWORD K15 to the observed mechanism: "write a new class with its annotations and NO `BEGIN/END BAJA AUTO GENERATED` markers; pre-written empty markers make slot-o-matic log `Could not process <file>: Found multiple metadata blocks` WITHOUT failing its task, and the build then fails later in `compileJava` with `Slot with name <x> not found … have you run slot-o-matic?`". Add a mechanical pre-build check to `build.sh` (before gradle): FAIL naming the file when a `.java` under `src/` has a `BEGIN BAJA AUTO GENERATED` marker with no `/*@ $` stamp line before its `END` marker; and a post-failure hint that maps "have you run slot-o-matic?" to "re-run `:slotomatic --info` and look for `Could not process`". Confirms `n4-course-as-exemplar-source` Δ4 (region-less sources need slot-o-matic before `jar`). Lint: YES. Acceptance: a fixture with empty markers → build.sh exits non-zero before gradle with the file named; the same file without markers → no row. | `METHODOLOGY.md` K15 (:122) + `toolbelt/build.sh` (pre-gradle scan + failure hints) | `[ev: transcript 2026-10-16 exam P1 slotomatic --info]` |
| Δ2 | Add to the property card: `type = "BDouble"` with `defaultValue = "BDouble.DEFAULT"` generates PRIMITIVE accessors (`double getX()`, `setX(double)`, `getDouble`/`setDouble`), so logic reads and writes `double`, never `BDouble`; state which `B*` simple types get primitive accessors (observed: `BDouble`; `int`/`String` properties in the same exam produced `getInt`/`getString` accessors) and which return the object. Lint: no. Acceptance: the card shows the generated lines from `BLimiter.java:53-65` verbatim. | `types/logic-authoring.md` § New-type authoring checklists | `[ev: code CertificationExam/…/BLimiter.java:53-65]` |
| Δ3 | Add the slot-order rule: Workbench property sheet and Wiresheet show slots in annotation declaration order, so declare paired bounds low-to-high (`min` before `max`, `low` before `high`, `lo` before `hi`) and inputs before outputs the operator edits; decide it at first authoring, because a later reorder is a schema change (`schema-risk.sh:212`). Lint: YES — WARN in `verify-module.sh --src` (or `lint-structure.sh`) when one class declares a `max*`/`high*`/`hi*` property before its `min*`/`low*`/`lo*` partner with the same suffix. Acceptance: `BLimiter.java` WARNs (`max` at :29 before `min` at :34); a min-first fixture is clean. | `types/logic-authoring.md` § Programmatic slot-name rules + `toolbelt/verify-module.sh --src` | `[ev: engram obs 11478]` |
| Δ4 | Add a bounds-inversion rule to the safety fail-modes: a component with operator-entered `min`/`max` must handle `min > max` explicitly (fault the output or refuse the write) instead of silently pinning to one bound — the exam limiter held `out` at 75 for all inputs below 75 after one swapped entry. `[INFER]` for the remedy (the exam accepted the unguarded code). Also confirms `n4-course-slots-components-contracts` Δ3 live (`isRunning()` guard first, dispatch on input slots only). Lint: no. Acceptance: the rule text names the observed symptom and links Δ3. | `types/logic.md` § Safety fail-modes & timers | `[ev: live 2026-10-16 BLimiter Max=25 Min=75]` |
| Δ5 | Refine `n4-course-authoring-files-and-libraries` Δ5: the annotation processor never UPDATES an existing `module-include.xml` `<type>` line either — adding `@AgentOn` to a type that already has a line leaves it without `<agent>`; the fix is to delete the stale line (or the file) and rebuild. Add a check: for every source class with `@AgentOn(types = {…})`, its `<type class=…>` in `module-include.xml` must list an `<on type=…>` for each declared type; FAIL otherwise (extends `lint-agent-on-shape.sh`, which checks shape only). Lint: YES. Acceptance: the Bonus Lab `-wb` tree passes; the same tree with the `<agent>` block removed from `module-include.xml` FAILs naming `BTrafficLightDriverPointManager`. | `toolbelt/lint-agent-on-shape.sh` (or `verify-module.sh --src`) + `types/module-wiring.md` § 2 · Own-type registration | `[ev: engram obs 11479]` |
| Δ6 | FIX the palette entry count: count every `<p` element that carries an `n="…"` attribute in ANY position and a non-folder `t="x:Type"`, not only `<p n=`; keep the folder exclusion. Confirms `n4-course-authoring-files-and-libraries` Δ3 on the root: the wizard emits `b:UnrestrictedFolder` (its driver palettes PASS `palette-root`) while palettes saved from Workbench keep `b:Folder`. Lint: YES. Acceptance: `training-rt.jar` → `PASS palette … 10 component entries`, `Cristian_Angeles_Cert-rt.jar` → 2; an empty-palette fixture still WARNs (FAIL under `--strict`). | `toolbelt/verify-module.sh` `check_palette` (:420) | `[ev: live 2026-10-16 verify-module.sh training-rt.jar]` |

## Lessons
- Slot-o-matic reports its own parse failure as a log line and lets `compileJava` fail later; read the
  slot-o-matic output before chasing a "slot not found" error.
- Declaration order is a UI contract: the operator reads the property sheet top to bottom and types by position.
- Generated registration files are append-only in practice: a change to a type's annotations needs the stale
  line removed, and only a source-vs-include check catches it.
- Palettes saved by Workbench reorder attributes; text checks on XML must not assume attribute order.

---
**Status**: PENDING — INDEX row appended: `| 2026-10-16-n4-course-handson-slotomatic-authoring.md | kit | 2026-10-16 | pending | 6 |`
