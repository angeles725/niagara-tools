<!-- review-status: folded -->
# 2026-09-18 · kit · spa-library-integration-deltas

**Session**: corpus-mining for build-n4-module (operator: generate all possible kit deltas — SPA/library integration).
**Delta count**: 6

## What happened

Mined B752 (three serving recipes, rc/ ORD scheme), B957 (typeExtensionDemo-ux: grunt/requirejs/BJsBuild for
a bajaux AMD module), and B1015 (UmbrellaDashboard: packaging a 2.73MB three.js SPA into `src/rc/`, removing
grunt from the ux `.gradle.kts`, injecting a live-wiring block, `sourceMode` demo/live toggle). Cross-checked
every finding against `types/dashboard.md`, the retros index, and today's other pending retros.

**Prior coverage confirmed (skipped as duplicate):**
- Three serving recipes decision (servlet / bajaux / PX) → `dashboard.md §ux` ✓
- Absolute `/<prefix>/...` URL rule for XHR guard → `dashboard.md §ux` ✓
- `static assets in src/rc/` classloader serving → `dashboard.md §ux` ✓
- base64-heavy SPA editing (Python str.replace + node --check) → `METHODOLOGY.md §Editing technique` ✓ (folded 2026-09-04)
- dashboard-preview.py `--mock` for iterate-before-build → `dashboard.md §HMI kiosk` ✓ (folded 2026-09-04)
- GruntBuildTask(babel/copy/requirejs) + grunt-niagara in Jasmine/Karma harness → sdk-examples-kit-deltas Δ3 PENDING
- BJsBuild + BBajaScriptTypeExt type-extension recipe → sdk-examples-kit-deltas Δ4 PENDING
- "self-contained 3D SPA → /api/equipment live-wiring" (override reading()/currentState(), ordinal map, demo fallback)
  → umbrelladashboard-module-creation Δ3 PENDING (Δ3 and Δ4 below sharpen it)

Six net-new deltas remain, spanning the grunt-removal decision, the embedding recipe, the injection technique,
the commissioning-safety framing, the build strategy decision table, and the rc-scan WARN expectation.

## Evidence

- `UmbrellaDashboard-ux.gradle.kts`: `com.tridium.niagara-grunt` plugin+task REMOVED — "we serve static
  `rc/`, no babel/requirejs". Only `web-rt` dep + `project(":UmbrellaDashboard-rt")` + `compileOnly servlet-api`
  remain. `[ev: corpus B1015 §1015.2]`
- `src/rc/index.html` = the 2.73MB three.js "Productos de Agua" SPA placed as-is in `src/rc/`; gradle copies
  `src/rc → rc/` at build time; servlet serves via `getClassLoader().getResourceAsStream("rc/"+path)`. ux jar
  weighs 1.63MB (JAR compression ratio ≈ 0.60 on the single-file SPA). `[ev: corpus B1015 §1015.2, §1015.4]`
- rc-scan check on the built jar: `clean but 1 cosmetic WARN` — the large `index.html` triggers the size warning;
  `verify-module.sh` still reports ALL PASS (17 passed, 0 failed). The WARN is informational. `[ev: corpus B1015 §1015.4]`
- Live-wiring injection block appended at the end of the main `<script>` in `index.html`: gates all overrides
  on `sourceMode==='live'`; overrides `reading(u)` and `currentState(u)` to return `__LIVE[unitKey]` records;
  `__pollLive()` fetches absolute `/umbrelladashboard/api/equipment` with `X-Requested-With: XMLHttpRequest`,
  maps `UP-0N → Unit${N}` ordinal, sets `lastUpdate=Date.now()`, calls `refreshData()`. Verified with
  `node --check` on the extracted script block. `[ev: corpus B1015 §1015.3]`
- `sourceMode` selector: "Estación (en vivo)" option added; demo stays as DEFAULT until live is explicitly
  selected; poll interval 5s (live only). Live mode is opt-in, not default. `[ev: corpus B1015 §1015.3]`
- typeExtensionDemo-ux (B957) shows grunt IS needed when authoring AMD modules for a bajaux `@AgentOn` view:
  `GruntBuildTask { tasks("babel:dist","copy:dist","requirejs") }` + `BJsBuild` singleton pointing at
  `*.built.min.js`; `package.json` devDeps: `grunt` + `grunt-niagara` + babel. `[ev: corpus B957 §957.3, §957.6]`
- B752 §752.1 classifies the three serving recipes; B957 shows the AMD/type-extension path; B1015 shows the
  pre-built-bundle path. Together they form the complete grunt vs static rc/ decision space. `[ev: corpus B752 §752.1, B957 §957.6, B1015 §1015.2]`

## Proposed kit deltas (propose-never-apply)

| Δ | Delta | Target file / § | Token |
|---|---|---|---|
| Δ1 | When the ux module serves a pre-built third-party JS SPA from `src/rc/`, remove `com.tridium.niagara-grunt` plugin and `gruntBuild` task from `<mod>-ux.gradle.kts` entirely — no babel/requirejs step needed; minimal deps are `web-rt` + `project(":…-rt")` + `compileOnly servlet-api`. Grunt is only needed when the module authors its own AMD/requirejs JS (bajaux type-extensions, B957). A leftover `niagara-grunt` dep on a static-serving module creates a phantom Grunt dependency, may require `nodeHome`, and runs a no-op build task on every compile. | `types/dashboard.md §ux — servlet + SPA` (new bullet after "gradle: …") | `[ev: corpus B1015 §1015.2]` |
| Δ2 | Add the third-party JS library embedding recipe: (1) place the pre-built single-file bundle (`index.html` or `library.min.js`) under `src/rc/` as-is — no compile step; (2) gradle's default `src/rc → rc/` copy brings it into the jar; (3) the servlet serves it via `getClassLoader().getResourceAsStream("rc/"+relativePath)` with a traversal guard. Large libraries compress well in JARs (2.73MB three.js SPA → 1.63MB ux jar, ratio ≈ 0.60). Reference any asset from the SPA using its `/<prefix>/` path prefix at runtime. | `types/dashboard.md §ux — servlet + SPA` (after existing "Static assets in src/rc/" sentence) | `[ev: corpus B1015 §1015.2, §1015.4]` |
| Δ3 | Add the live-wiring INJECTION technique for a pre-built SPA: inject a thin block at the end of the SPA's main `<script>` rather than rebuilding it. Pattern: (a) gate every override on `sourceMode==='live'`; (b) shadow the SPA's own data-provider functions (`reading(u)`, `currentState(u)`) to return records from a `__LIVE` map when in live mode; (c) `__pollLive()` fetches the absolute servlet URL with `X-Requested-With: XMLHttpRequest`, maps the SPA's entity keys to the facade's slot-path keys (e.g. `UP-0N → "Unit${N}/slotName"`), updates `__LIVE`, then calls the SPA's own refresh. Always verify the injected block with `node --check` on the extracted script. This sharpens the pending 2026-09-16 Δ3 (umbrella) with the injection framing and the `node --check` step. | `types/dashboard.md §Extending an existing dashboard` (new bullet "Wiring a pre-built third-party SPA", after "The module is the SKELETON…") | `[ev: corpus B1015 §1015.3]` |
| Δ4 | Name the `sourceMode` demo/live toggle as a commissioning-safety pattern: the live-wiring block adds "Estación (en vivo)" to the SPA's source selector; demo/sample mode STAYS as the default. Live is opt-in until the station is wired and confirmed. This prevents a blank or broken HMI during the commissioning window when point links are not yet established. Never make live the default until `verify-module.sh` AND a live smoke on the real station pass. Extends/sharpens pending 2026-09-16 Δ3 "demo fallback" with an explicit commissioning-safety framing. | `types/dashboard.md §Extending an existing dashboard` (as a named rule, after Δ3's bullet) | `[ev: corpus B1015 §1015.3]` |
| Δ5 | Add a JS build strategy decision table to the ux type guide: (a) **bajaux `@AgentOn` view with custom AMD modules** → keep `com.tridium.niagara-grunt` + babel + requirejs + `BJsBuild` singleton (B957 recipe; the built bundle ships in `rc/*.built.min.js`); (b) **servlet-SPA with a pre-built single-file bundle** (three.js, Chart.js, etc.) → remove grunt, serve static `rc/`; no `BJsBuild` needed; (c) **module with BOTH a bajaux type-extension and a servlet-SPA** → only the bajaux profile needs grunt; the servlet-SPA profile stays grunt-free. The decision pivots on whether the module AUTHORS its own AMD modules or simply embeds a pre-built bundle. | `types/dashboard.md §ux — servlet + SPA` (new subsection "JS build strategy") | `[ev: corpus B957 §957.3, §957.6, B1015 §1015.2]` |
| Δ6 | Document that `verify-module.sh rc-scan` emits a cosmetic WARN for a large pre-built library file in `rc/` (triggered on `index.html` embedding a 2.73MB SPA). This WARN is informational and does not affect the ALL PASS gate. Distinguish it from a real FAIL (which affects exit code / gate). Note it as expected under the `## Benign WARNs` pattern already documented for `phantom-dep` and `ord-literal`. | `types/dashboard.md §ux — servlet + SPA` (after Δ2's embedding recipe) or as an annotation in `toolbelt/verify-module.sh` usage header near the rc-scan section | `[ev: corpus B1015 §1015.4]` |

## Lessons

- The kit already documents "static assets in src/rc/" for serving the SPA — but says nothing about REMOVING
  grunt from the ux build. The removal is load-bearing: leaving grunt on a static-serving module adds a
  phantom node dependency and a no-op build step. Δ1 + Δ5 are the corrective.
- B957 and B1015 are complementary data points: B957 shows grunt IS needed for AMD/type-extension ux; B1015
  shows grunt must be REMOVED for a pre-built-bundle ux. Neither block alone gives the full decision rule.
  Together they define the two branches of Δ5.
- The live-wiring injection technique (Δ3) is the key insight for library integration: you don't rebuild the
  SPA, you append a thin integration block that shadows the SPA's data providers. This keeps the library
  maintainable (the original SPA can still be upgraded as-is) and the integration layer small.
- The `sourceMode` opt-in framing (Δ4) is a commissioning safety measure, not a feature: it ensures the SPA
  shows coherent demo data even when the station point links are not yet established. Making live the default
  risks a blank/broken HMI during on-site commissioning.
- Δ3 and Δ4 sharpen the pending 2026-09-16 Δ3 (umbrella retro) — that delta names the elements but does not
  give the injection framing or the commissioning-safety rationale. Both should be folded together.

---
**Status**: PENDING — INDEX row appended: `| 2026-09-18-spa-library-integration-deltas.md | kit | 2026-09-18 | pending | 6 |`
