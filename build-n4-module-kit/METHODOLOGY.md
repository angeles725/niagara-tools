# Common checklist — every N4 module (the verify gate)

Applies to all module types. Each item is proven from real builds (DashboardPan/ColdRoomPan/chihuahua, 2026-08). Run this against the built module before "done".

## rt (components)
- [ ] **Slot flags** on every `@NiagaraProperty`: `Flags.SUMMARY` for visible; `SUMMARY|OPERATOR` for operator-writable config; `TRANSIENT|SUMMARY|READONLY` for computed outputs; `HIDDEN` for internal. (control-rt `BNumericWritable` is the exemplar.)
- [ ] **Facets / units.** Temperatures carry a Celsius facet: `BFacets.make(BFacets.UNITS, BUnit.getUnit("celsius"))`. A MIN/MAX facet wraps its number in `BDouble.make(...)` — `BFacets.make(BFacets.MIN, 0d)` does NOT compile (no `make(String,double)` overload).
  - **delta vs absolute:** a *difference* (e.g. a hysteresis `differential`, a deadband) is in *degrees*, not absolute Celsius (offset 273.15). Show it as a band, label it as such.
- [ ] **@NiagaraProperty edits in 3 places:** the annotation `@Facet(...)`, the generated `newProperty(...)` region, AND the imports. Prove with `slotomatic` (it regenerates from the annotation — a fix only in the generated region reverts on the next regen). `import javax.baja.sys.BDouble` if the annotation uses `BDouble.make`.
- [ ] **Status flags.** Read/propagate `BStatus` (fault/down/stale/disabled/null/overridden/alarm) — do not collapse everything to null. Guard control decisions on `getStatus().isValid()`.
- [ ] **module.lexicon** populated (type + user-facing slot display names); format `key=value` (see control-rt.lexicon).
- [ ] **Icon:** a 16×16 PNG in `src/rc/` + `getIcon()` returning `BIcon.make(BOrd.make("module://<mod>/rc/icon16.png"))`.
- [ ] **Permissions:** delete the empty `type="all"` wizard scaffold in module-permissions.xml → `<permissions/>` (a component/dashboard module needs the base grant). [module-anatomy B636 deviation #1]

## Domain correctness
- [ ] Compare a value only against a limit that APPLIES to it: setpoint/deviation/alarm belong to zone sensors, NOT to evaporator/resistance temps (those alarm against their own high/low limits). Wrong comparisons = false alarms.
- [ ] Times shown legibly (hours/minutes), not raw seconds/ms.

## Build (see build-verify.md)
- [ ] Built with **Java 8** + **clean + slotomatic + jar**. Bytecode major version **52**.
- [ ] Both/all profile jars **signed** (`META-INF/NIAGARA4.SF`).
- [ ] Tests: unit-test the pure-Java model (e.g. a pure router) with JUnit — `niagaraTest` does not run in WSL.

## Tradeoffs to state, not hide
- Adding alarm sources / control points to a "pure display" facade makes it an alarm SOURCE — a real change of role. Flag it.
