<!-- review-status: folded -->
# 2026-09-18 · kit · sdk-examples-kit-deltas

**Session**: corpus-mining for build-n4-module (operator: generate kit deltas from niagara-research).
**Delta count**: 8

## What happened

Mined SDK example blocks B955–B961 for concrete, evidence-backed idioms the kit should adopt.
B961 §961.3 seeds the candidate list; B957–B960 supply the primary evidence. Duplicate scan
found NO prior folded retros covering the BTestNg recipe, Jasmine/Karma harness, doCheckLink/
INDIRECT-link pattern, or the moduleTest-include.xml descriptor gap. The `2026-09-16-n4-client-
build-config-standard.md` (pending) already proposes `nodeHome` for ux modules from the client
compare; Δ6 below sharpens that with independent B960 §960.3 evidence and a JRE-from-Niagara
addendum the pending retro does not cover. The `qa-stack-pure-tests` retro (folded) documents
that `niagaraTest` is not a WSL gate; it does NOT supply the authoring recipe — Δ1 and Δ2 are
therefore new.

## Evidence

- `BLinkCheckTestTest extends BTestNg`, TestNG `@Test(groups)` / `@BeforeClass` / `@AfterClass`;
  `handler = createTestStation(); handler.startStation(); handler.releaseStation()` — the station-
  side integration test harness, exact wiring our CompPan-rt/ColdRoomPan-rt were missing.
  `[ev: corpus B958 §958.3]`
- `moduleTest-include.xml` is a SEPARATE file from `module-include.xml` — it registers only the
  test type; the test type is NOT shipped in the production jar but in `moduleTestJar`.
  `moduleTestImplementation("Tridium:test-wb")` + runtime deps in the profile `.gradle.kts`.
  `[ev: corpus B958 §958.4]`
- B961 §961.3 explicitly proposes: "a module with production logic but no `moduleTest-include.xml`
  → WARN" as a `verify-module.sh` addition. `[ev: corpus B961 §961.3]`
- `srcTest/rc/spec/DemoSizeSpec.js` (Jasmine describe/it), `allSpecs.js` aggregator,
  `srcTest/rc/browserMain.js` (Karma entry; stubs the built bundle so unminified sources load),
  `srcTest/rc/stations/*/config.bog` (test station deployed by `grunt-niagara`); `package.json`
  devDeps include `grunt-niagara` + babel + `babel-plugin-istanbul`; `Gruntfile.js` wires
  `niagara.station{ stationName, sourceStationFolder }`. `[ev: corpus B957 §957.5]`
- `GruntBuildTask` in the ux `.gradle.kts`: `tasks("babel:dist","copy:dist","requirejs")` — the
  Gradle→Grunt bridge. `[ev: corpus B957 §957.6]`
- `BBajaScriptTypeExt` + `@AgentOn(types="…")` + `@NiagaraSingleton` + `BIOffline` + `JsInfo.make(jsOrd,
  JsBuild.TYPE)` — the Java→browser type binding; `BJsBuild extends BJsBuild` singleton pointing
  at `*.built.min.js` bundle ORD. Browser registration is implicit (driven by `@AgentOn`, never
  `baja.registerType()`). `[ev: corpus B957 §957.1–957.3]`
- `doCheckLink(BComponent source, Slot sourceSlot, Slot targetSlot, Context cx)` gates incoming
  links: `return LinkCheck.makeInvalid("reason")` rejects; `makeValid()` accepts. Called by the
  framework before any link is created. `[ev: corpus B958 §958.1]`
- `added(Property, Context)` / `removed(Property, BValue, Context)` react to link creation/
  removal. Reciprocal links should be **indirect** (`new BLink(sessionOrd, srcSlot, dstSlot,
  true)` — the `true` flag); rationale in the source comment: "we cannot guarantee the order in
  which the components will start when the station is restarted. Using an indirect link will ensure
  we always have an ORD to the source component." `[ev: corpus B958 §958.2]`
- `gradle.properties.EXAMPLE`: pins `niagara_home`, `niagara_user_home`, `nodeHome` (only for
  JS/ux modules), and `org.gradle.java.installations.paths` pointing at Niagara's OWN JRE
  folder — not a system JDK — with `org.gradle.jvm.toolchain.auto-detect=false /
  auto-download=false`. `[ev: corpus B960 §960.3]`
- Canonical per-module descriptor table: `niagara-module.xml` (module identity), per-part
  `module-include.xml` (production types), per-part `moduleTest-include.xml` (test types → goes
  to `moduleTestJar` ONLY), `module.palette`, `module.lexicon`. `[ev: corpus B960 §960.4]`

## Proposed kit deltas (propose-never-apply)

| Δ | Delta | Target file / § | Token |
|---|---|---|---|
| Δ1 | New `types/moduleTest.md`: complete BTestNg authoring recipe — `srcTest/` class extending `BTestNg`, `@NiagaraType` + TestNG `@Test(groups)`, `@BeforeClass createTestStation()/startStation()` + `@AfterClass releaseStation()`, `moduleTest-include.xml` (SEPARATE, NOT in production jar), `moduleTestImplementation("Tridium:test-wb")` + needed runtime deps in `.gradle.kts`; note this is the station-side complement to the pure-JUnit WSL tier and requires a running NRE (run from Workbench → Tools → Run Tests until niagaraTest task bug resolved) | new `types/moduleTest.md` | `[ev: corpus B958 §958.3–958.4]` |
| Δ2 | `toolbelt/verify-module.sh`: add new `check_moduletest_present` WARN check (default, with `--src`): when the -rt profile dir has `src/` Java sources but no `moduleTest-include.xml`, emit `WARN moduletest-present <jar> "production logic in src/ but no moduleTest-include.xml — add BTestNg integration tests"` (WARN not FAIL; never changes exit code per K20); add to the check function list and update the usage header | `toolbelt/verify-module.sh` | `[ev: corpus B961 §961.3]` |
| Δ3 | `types/dashboard.md §ux`: add new subsection "Browser test harness (DUX-TEST1)" — Jasmine `describe/it` specs in `srcTest/rc/spec/`, `allSpecs.js` AMD aggregator, `browserMain.js` Karma entry with built-bundle STUB trick (`define('…built.min', {})` so Karma loads unminified sources), minimal test station bog in `srcTest/rc/stations/`, `grunt-niagara` devDep + `niagara.station{ stationName, sourceStationFolder }` in Gruntfile; complements `dashboard-preview.py` (layout preview, no assertions); this is currently entirely absent from the kit | `types/dashboard.md §ux` (new subsection after DJS1) | `[ev: corpus B957 §957.5]` |
| Δ4 | `types/dashboard.md §Web-tier exemplars (DUX-WEB1)`: add B957 to the exemplars table — `BBajaScriptTypeExt` + `@AgentOn` + `JsInfo` + `BJsBuild` singleton = the ux type-extension recipe for custom `BSimple` values; `BIOffline` for no-live-station servability; JS side: `define(['baja!'], function(baja){ class MyType extends baja.Simple { … } })` (implicit registration via `@AgentOn`, never `baja.registerType()`); `bajaScript-ux` + `js-ux` + `web-rt` runtime deps; distinct from servlet-SPA — a dashboard does NOT need this unless it has custom typed values | `types/dashboard.md §Web-tier exemplars (DUX-WEB1)` | `[ev: corpus B957 §957.1–957.4]` |
| Δ5 | `types/logic.md §Linking across custom modules`: add new subsection "Link lifecycle — gating and reacting" — `doCheckLink(source, sourceSlot, targetSlot, cx)` to validate incoming links before creation (`LinkCheck.makeInvalid("reason")` / `.makeValid()`); `added(Property, Context)` / `removed(Property, BValue, Context)` to react to link creation/removal; **INDIRECT-link rule**: a reciprocal or cross-component link added in `added()` must use `new BLink(sessionOrd, srcSlot, dstSlot, true)` (the `true` flag = indirect, ORD-resolved) to survive restart ordering — a direct link breaks if the target starts before the source | `types/logic.md §Linking across custom modules` | `[ev: corpus B958 §958.1–958.2]` |
| Δ6 | `types/structure.md §L10 doctrine`: add two bullets — (a) for a ux/JS module `nodeHome` is a REQUIRED `gradle.properties` key (alongside `org.gradle.java.installations.paths`); (b) point `org.gradle.java.installations.paths` at Niagara's own bundled JRE folder (`<niagara_home>/jre`) rather than a system JDK to guarantee the compiler matches the runtime (corroborated independently by B960 `gradle.properties.EXAMPLE`); these complement the pending `2026-09-16-n4-client-build-config-standard` Δ1 which proposes the same `nodeHome` key from the client compare source | `types/structure.md §L10 doctrine` | `[ev: corpus B960 §960.3]` |
| Δ7 | `types/structure.md §module-include.xml vs META-INF/module.xml`: extend to a canonical descriptor table — niagara-module.xml (module identity + runtimeProfiles), per-part `module-include.xml` (production types → jar), per-part `moduleTest-include.xml` (test types → moduleTestJar ONLY; NOT in the shipping jar), `module.palette`, `module.lexicon`; make explicit that `moduleTest-include.xml` is a SEPARATE file, not a section of `module-include.xml`, and that the test type must NOT appear in the production jar | `types/structure.md §module-include.xml vs META-INF/module.xml` | `[ev: corpus B960 §960.4]` |
| Δ8 | `types/structure.md §PASS state + scaffold`: note that `scaffold-module.sh` output should include an empty `moduleTest-include.xml` stub (with a `<types/>` element and a TODO comment) alongside `module-include.xml`; without it, the L11 check fires if a developer adds a `BTestNg` test without the XML — giving an empty stub at scaffold time is cheaper than diagnosing the lint later | `types/structure.md §PASS state + scaffold` | `[ev: corpus B961 §961.3, B958 §958.4]` |

## Lessons

- The `qa-stack-pure-tests` retro (folded) correctly said `niagaraTest` is not a WSL gate; it never gave the authoring recipe. The recipe (Δ1) and the verify check (Δ2) are the natural next step — a gap between "documented unavailable" and "here is how to write one when you do have a station."
- The ux test harness (Δ3) is entirely absent from the kit; `dashboard-preview.py` runs a layout preview but asserts nothing. The Jasmine/Karma pattern from B957 is the only first-party template for executable browser-side assertions.
- The `doCheckLink` / INDIRECT-link pattern (Δ5) appears nowhere in the kit — `types/logic.md §Linking across custom modules` covers the cross-module double vs enum rule but says nothing about gating incoming links or restart-order safety for programmatically added links.
- B960 §960.4 provides the canonical five-file descriptor table. Making `moduleTest-include.xml` explicit in `types/structure.md` (Δ7) removes a common confusion: developers adding a test file to the wrong XML (polluting the production jar with test types).

---
**Status**: PENDING — INDEX row appended: `| 2026-09-18-sdk-examples-kit-deltas.md | kit | 2026-09-18 | pending | 8 |`
