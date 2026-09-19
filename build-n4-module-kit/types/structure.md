# Type: module structure — the layout, naming, and a conformance lint (`lint-structure.sh`)

The shape a Niagara N4 module must have on disk, and the L1–L11 checklist `lint-structure.sh` enforces. Grounded
in how Tridium/Honeywell lay out their own modules and an audit of our four. `[ev: corpus B817]`

## Layout — one `moduleName`, per-profile parts `[ev: corpus B817]`
- A module splits into per-`runtimeProfile` artifacts `<module>-{rt,ux,wb,se}`: `-rt` server runtime, `-ux` browser
  (BajaScript), `-wb` Workbench (Swing), `-se` server-only. The `-rt` part lists its siblings in `<moduleParts>`.
- `-doc` is a SEPARATE `runtimeProfile="doc"` module (with an empty `<types/>`, per B784 §784.1), NEVER a part of a code module.
- Author-side source tree: `<MOD>-rt.gradle.kts` + `module-include.xml` + `module.lexicon` + `module.palette` +
  `src/com/<vendor>/<MOD>/…` (+ `srcTest/` for tests, `rc/` for `-ux`). `[ev: corpus B817]`

## Naming — `javax.baja.*` = framework API, `com.<vendor>.*` = your code `[ev: corpus B817]`
- Tridium declares PUBLIC framework types under `javax.baja.<domain>` (e.g. `javax.baja.control.BControlPoint`) and
  keeps implementation under `com.tridium.<module>`. Honeywell/OEM modules use `com.<vendor>.<module>` throughout
  (ours: `com.angeles.<Module>`) — a leaf/OEM module NEVER declares `javax.baja.*` types.
- One public `@NiagaraType` per `.java` file; `BXxx` class naming. `[ev: corpus B817]`

## `module-include.xml` vs `META-INF/module.xml` `[ev: corpus B817]`
- You AUTHOR `module-include.xml` (the `<type>` list). The gradle plugin GENERATES `META-INF/module.xml` from it.
- A hand-authored `META-INF/module.xml` in source is WRONG (L6): it will be overwritten and drifts from the plugin
  output. Edit `module-include.xml`; never the generated manifest. `[ev: corpus B817]`

### Canonical per-module descriptor table `[ev: corpus B960 §960.4]`

Every module ships these five files (some may be empty stubs for minimal modules):

| File | Location | Role | Goes into |
|------|----------|------|-----------|
| `niagara-module.xml` | module root | module identity (`moduleName`, `preferredSymbol`, `runtimeProfiles`) | checked by build; not in a jar |
| `<part>/module-include.xml` | per runtime-profile part | production type registry (`<type name=… class=…>`) | the production **jar** |
| `<part>/moduleTest-include.xml` | per runtime-profile part | test type registry — test types ONLY | the **moduleTestJar** (never the shipping jar) |
| `<part>/module.palette` | per runtime-profile part | Workbench drag-drop palette (`bajaObjectGraph`) | the production jar |
| `<part>/module.lexicon` | per runtime-profile part | i18n string table (`key=value`) | the production jar |

**Critical separator:** `moduleTest-include.xml` is a SEPARATE file from `module-include.xml` — it is NOT
a section of `module-include.xml` and the two must never be merged.  A test type that leaks into
`module-include.xml` ships in the production jar, making test code available on production stations.
The Gradle plugin emits a distinct `moduleTestJar` output task that includes the test registry; the
regular `jar` task does not. `[ev: corpus B958 §958.4, B960 §960.4]`

## The good-module checklist — L1–L11 (`lint-structure.sh <module-root>`) `[ev: corpus B817]`
Row format: `FAIL|WARN  lint-structure  <path>  L<n>: <reason>`. Exit **0** clean · **1** any FAIL · **3** usage.
`.deploy-baseline/` subtrees are PRUNED (a deploy snapshot is not source). `[ev: corpus B817]`

| L | Checks | Verdict | Real shape that proved it |
|---|---|---|---|
| L1 | every `.java` package is `com.<vendor>.<Module>[.<sub>]`, consistent | FAIL | all four modules PASS |
| L2 | exactly one public `@NiagaraType` per `.java` file | FAIL | all four PASS |
| L3 | a pure-model pkg (`…/model/`) with a `srcTest` test per class AND zero `import javax.baja` in it | WARN | chihuahua `wb/model/` isolated ✓; ColdRoomPan/CompPan keep it flat (advisory) |
| L4 | `module.lexicon` present AND non-empty (≥1 `key=value`) when ≥1 type is declared | FAIL | chihuahua-rt + -ux lexicons EMPTY → **L4 FAIL** |
| L5 | `module.palette` present AND non-empty on a component module | FAIL | empty palette = the B788 footgun |
| L6 | source ships `module-include.xml`, NOT a hand-authored `META-INF/module.xml` | FAIL | plugin generates the manifest |
| L7 | every `<dependency>` `vendorVersion` is a 3-part floor (`4.14.0`), not 2-part or the 4-part self-stamp | FAIL | mutation `:baja:4.14` (2-part) → **L7 FAIL** |
| L8 | signed jar present (`NIAGARA4`/vendor `.SF`+`.RSA`) | FAIL | shared with `verify-module.sh` |
| L9 | no empty skeleton part — a declared `-wb`/`-ux` with 0 `.java` AND empty palette/lexicon | FAIL | DashboardPan-wb (0 source, empty palette) → **L9 FAIL** |
| L10 | no absolute HOST paths in a tracked `gradle.properties` (`C:\…` `niagara_home`/`user_home`/`nodeHome`) | FAIL | client `ColdRoomPan`/`CompPan` `gradle.properties` hardcode `C:\Honeywell\…` → **L10 FAIL** (`.deploy-baseline/` pruned) |
| L11 | a `srcTest` mixing pure-JUnit + Baja (`BTest`/`BTestNg`) declares BOTH `moduleTestImplementation(":test-wb")` AND junit declarations (or splits source sets) | FAIL | ColdRoomPan-rt/CompPan-rt declare only `:test-wb` → **L11 FAIL** |

`[ev: corpus B817]` for every row. Deeper cites B817 carries: lexicon/palette STANDARD ← B780/B759 (§817.4), the
empty-lexicon/empty-palette AUDIT that fires L4/L5 ← B788; L6 ← B790 §14; L7 ← B784; L8 ← B807; L10/L11 ← B815 §815.12.

## L10 doctrine — gradle.properties template + git-ignored local override `[ev: retro gradle-properties-consistency Δ3]`

L10 fires when `gradle.properties` contains absolute host paths (`C:\...`, `niagara_home=`, `user_home=`, `nodeHome=`). The correct pattern is:

1. **Commit a template** `gradle.properties` with paths commented out (or with a safe placeholder), so a fresh checkout builds without host-path edits.
2. **Git-ignore a local override** `gradle.properties.local` (or `local.properties`) where each developer stores their actual `niagara_home`, `org.gradle.java.installations.paths`, etc. The build script reads the local override if present; Gradle's `gradle.properties` in `$GRADLE_USER_HOME` also works.
3. **`build.sh` overrides `niagara_home` via arg 3** — so a WSL build always gets the correct path from the operator regardless of what `gradle.properties` says. But a Windows `gradlew` build reads `gradle.properties` directly and will fail if it still points at a Windows path that differs on the build machine.
4. **`org.gradle.java.installations.paths` / `auto-detect`:** commit `auto-detect=false` + a commented-out `paths=` line in the template. A commented-out or absent block means auto-detect, which is machine-dependent and can pick the wrong JDK major (the "gradle :jar with the default JDK is NOT a build" hazard). The local override sets the real path.

**Why this matters:** three modules in the same group deploying to ONE station had three different `niagara_home` versions in committed `gradle.properties`. A WSL `build.sh` build (which overrides niagara_home) masked the divergence — all jars were major-52. A Windows `gradlew` build would have picked up the divergent paths and built against different API versions. [ev: retro gradle-properties-consistency Δ3]

5. **`nodeHome` is a REQUIRED `gradle.properties` key for any ux/JS module** (alongside `niagara_home` and
   `org.gradle.java.installations.paths`).  Without it the Grunt/babel build task cannot locate Node.  Add
   it to the `.EXAMPLE` template (commented-out or with a placeholder) and to the local override.
   Corroborated by the SDK `gradle.properties.EXAMPLE` (`nodeHome=` at line 8–9). `[ev: corpus B960 §960.3]`

6. **Point `org.gradle.java.installations.paths` at Niagara's own bundled JRE, not a system JDK:**
   use `<niagara_home>/jre` (the JRE folder shipped inside the Niagara installation).  Set
   `org.gradle.jvm.toolchain.auto-detect=false` and `org.gradle.jvm.toolchain.auto-download=false`
   alongside it.  This guarantees the Gradle compiler matches the runtime bytecode level exactly —
   using a system JDK risks a different major version and produces a jar the NRE rejects.
   `[ev: corpus B960 §960.3]`

## PASS state + scaffold `[ev: corpus B817]`
`scaffold-module.sh <MOD>` output passes L1–L11 at exit 0 — the skeleton is the GREEN fixture. A mutation that
empties the palette (L5/L9), a lexicon (L4), drops a 3-part floor (L7), hardcodes a `C:\` path (L10), or mixes
tests without both deps (L11) flips the corresponding row to FAIL. `[ev: corpus B817]`

**Scaffold should emit an empty `moduleTest-include.xml` stub** alongside `module-include.xml`:

```xml
<!-- <part>/moduleTest-include.xml — generated by scaffold-module.sh -->
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE moduleInclude PUBLIC
  'moduleInclude.dtd'
  'http://niagara.tridium.com/xml/dtd/moduleInclude/moduleInclude.dtd'>
<moduleInclude>
  <!-- TODO: register BTestNg test types here when you add station-side integration tests.
       See types/moduleTest.md for the full recipe.
       This file feeds moduleTestJar ONLY — never add test types to module-include.xml. -->
  <types/>
</moduleInclude>
```

**Why emit it at scaffold time:** if a developer adds a `BTestNg` test class later and forgets this
file, the build silently omits the test from `moduleTestJar` and `niagaraTest` / Workbench can't find
it.  An empty stub costs nothing and prevents the "my test vanished" diagnosis.
`[ev: corpus B961 §961.3, B958 §958.4]`

## Recommendations for our modules (impact ÷ cost) `[ev: corpus B817]`
R1 chihuahua — populate rt+ux `module.lexicon` (10 types unlocalized; cheap, operator-visible). R2 DashboardPan-wb
— delete the empty skeleton OR fill it + add its JUnit dep. R3 DashboardPan — test `DashboardReader` (14-baja, the
sole data engine, untested) via a `BTestNg` station test (B815 recipe). R4 ColdRoomPan/CompPan — isolate the pure
model in a `model/` sub-package + add its junit moduleTest dep (L11). R5 ColdRoomPan — add the missing
`BFanMode`/`fanRunMode` lexicon keys. `[ev: corpus B817]`
