# Module wiring — Gradle deps, type registration, agent-on, cross-module references

How a module declares dependencies on other modules, registers its own types, and
calls into types across module boundaries. `[ev: code nmodsreflow gradle.kts; corpus B802; real module.xml]`

---

## 1 · Gradle dependency configurations → `module.xml` impact

| Gradle declaration | `module.xml` effect | Notes |
|---|---|---|
| `nre(":nre")` | Sets `nre="true"` attribute on `<module>` — NO `<dependency>` entry | Marks the module as NRE-level; every production module should carry this |
| `nre("group:artifact")` (no version) | Nothing in `module.xml` | Compile against a library the NRE ships in `!bin/ext` (devguide 4.15.6.20 `build.txt:186`); never bundle it — `types/third-party-libraries.md` § Before bundling `[ev: retro devguide415-r2-shipped-libraries Δ1]` |
| `api(":baja")` / `api(":web-rt")` | Emits `<dependency name="baja" vendor="Tridium" vendorVersion="X.Y.Z"/>` | Transitive: a module that `api()`s yours also sees baja; 3-part floor required (L7) |
| `api(project(":mymod-rt"))` | Emits `<dependency name="mymod-rt" vendor="<OEM>" vendorVersion="<major.minor>"/>` | Sibling part dep; vendor and version stamped from `defaultModuleVersion` in the GROUP `build.gradle.kts`, NOT from the part's own `.kts` `[ev: nmodsreflow-ux module.xml; client-module-version-key memory]` |
| `compileOnly("javax.servlet:...")` | **Nothing** in `module.xml` | Servlet API is provided by the container at runtime; compile-time only |
| `compileOnly(files("$niagara_home/bin/ext/jetty-all-compact3-*.jar"))` | **Nothing** in `module.xml` | Niagara-private provided jar; no Maven coordinate → path-pinned; absent on JACE (Compact3 profile) |
| `uberjar("group:artifact:version")` | **Nothing** in `module.xml` | Fat-jar embed built into the `com.tridium.niagara-module` plugin; simpler than Shadow for single-module bundling (no relocation); `third-party-libraries.md` teaches Shadow for shade/relocate scenarios `[ev: code nmodsreflow gradle.kts; WIR-G1]` |
| `moduleTestImplementation(":test-wb")` | **Nothing** in `module.xml` (test scope only) | Feeds `moduleTestJar`; never appears in the shipping jar `[ev: corpus B958]` |

**Version floor for Tridium deps:** Tridium modules are resolved from `niagara_home/!modules/`
at build time. The floor is the 3-part string you declare (e.g., `4.14.0`); the runtime
installs any version ≥ floor. A 2-part floor (e.g., `4.14`) fails lint L7. `[ev: corpus B784; lint-structure L7]`

**OEM sibling version:** set `defaultModuleVersion("X.Y.Z")` in the **GROUP**
`build.gradle.kts`, not in any individual part `.kts`. The plugin stamps every sibling
`<dependency>` with `major.minor` (2-part) from this value. `[ev: client-module-version-key memory]`

### 1.1 · Supertype modules and the part build script

**Depend on the module of every supertype, not only the type's own module.** javac needs each superclass and
implemented interface on the compile classpath. Course Lab 10 needed three for two types: `api(":history-rt")`
for `BHistoryToCsv`, `api(":file-rt")` because it extends `com.tridium.file.exporters.BITableToCsv`, and
`api(":bql-rt")` because `BHistoryDatabase` implements `javax.baja.bql.BIRelational` (javap on 4.15.6.20 jars).
Symptoms: javac `class file for <X> not found`; IntelliJ shows no class between the type and `Object`, then
"BValue cannot be cast". Find the chain with `javap -cp <niagara_home>/modules/<mod>.jar <class>` and the
owning jar with `unzip -l`. The `build.sh` hint that maps the missing class to its `api(":<mod>")` line is owed
(G60). `[ev: course Labs/Lab Instructions.pdf p81-86]` `[ev: retro n4-course-build-deps-permissions-signing Δ1]`
`[deferred: retro n4-course-build-deps-permissions-signing Δ1]`

**Part build script (`<part>.gradle.kts`)** — the course `slides-rt.gradle.kts` is the reference shape
(`Slide Code/slides/slides-rt/slides-rt.gradle.kts:8-76`; tasks and dependency forms: `Slides/6 Gradle and
Permissions.pdf` p6-18):
- `plugins { id("com.tridium.niagara-module"); id("com.tridium.niagara-signing"); id("com.tridium.bajadoc");
  id("com.tridium.niagara-jacoco"); id("com.tridium.niagara-annotation-processors");
  id("com.tridium.convention.niagara-home-repositories") }` — signing also applies at the root.
- `moduleManifest { moduleName.set("<mod>"); runtimeProfile.set(rt) }`.
- Dependency forms by release: 4.12 and older `compile "Tridium:alarm-rt:4.12"`; 4.13 `api("Tridium:alarm-rt")`;
  4.14 and newer `api(":alarm-rt")`, `nre(":nre")`, `api(project(":<sibling>-rt"))`.
- Non-source files in the jar: `tasks.named<Jar>("jar") { from("src") { include("<pkg path>/**/*.png") } }`.
- `tasks.named<Bajadoc>("bajadoc") { options.encoding = "utf-8"; includePackage("<pkg>") }` — every package must
  be listed by hand and the list drifts: the course's own file names packages that do not exist under its
  `src/` and misses ones that do. A lint comparing the two sets is owed (G62).
- Tasks: `slotomatic` (generate from annotations), `jar` (compile, assemble, copy into `<niagara_home>/modules`),
  `clean` (also deletes the jar from `modules`), `moduleTestJar`; qualify them (`:<part>:jar`).
`[ev: retro n4-course-authoring-files-and-libraries Δ4]`

---

## 2 · Own-type registration (`module-include.xml` → `module.xml`)

`module-include.xml` lives in source (javac writes its lines, below; you prune them); the gradle plugin
generates `META-INF/module.xml` from it. **Never hand-author `META-INF/module.xml`** — it will be overwritten (lint L6). `[ev: corpus B817]`

- **`module-include.xml` holds only `<types>`.** Defs, lexicons, installation and finer-grained dependencies
  go in the part's `moduleManifest { … }` block, and the parts come from the root `niagara-module.xml`
  (devguide `upgradingBuild.txt:95,107`); a `<defs>`, `<lexicons>` or `<moduleParts>` element in it is
  Gradle-6-era residue. `[ev: corpus B1313 §1313.3 C2]` `[ev: retro devguide415-horizons-authoring-security Δ6]`
- **javac's annotation processor writes it, append-only.** For each `@NiagaraType` whose `<type>` line is missing
  or differs, `NiagaraTypeAnnotationProcessor` adds one and saves the file; it never removes or renames one. After
  renaming or deleting a type, delete its stale line (or the file, to regenerate it), or registry build logs
  SEVERE `Missing class for "<type>"` at station start (`Builder.java:766`; a listed class without `Sys.loadType()`
  logs `Missing Sys.loadType() <type>`, `:776`). Test types
  (`test.com.<vendor>…B*Test`) belong only in `moduleTest-include.xml`: the processor picks that file for classes
  under a configured test root. `[ev: corpus B1233 §1233.4]` `[ev: corpus B1128]`
  `[ev: retro n4-course-authoring-files-and-libraries Δ5]`

```xml
<!-- module-include.xml (authored) -->
<moduleInclude>
  <types>
    <!-- plain type -->
    <type class="com.acme.mymod.BMyService"   name="MyService"/>
    <!-- type that also registers a BOrdScheme handler for the "myord:" prefix -->
    <type class="com.acme.mymod.BMyScheme"    name="MyScheme"    ordScheme="myord"/>
  </types>
</moduleInclude>
```

Generated shape in `META-INF/module.xml`:

```xml
<types>
  <type class="com.acme.mymod.BMyService" name="MyService"/>
  <type class="com.acme.mymod.BMyScheme"  name="MyScheme" ordScheme="myord"/>
</types>
```

**TYPE anchor in Java** (slot-o-matic generates this):

```java
public class BMyService extends BComponent {
    public static final Type TYPE = Sys.loadType(BMyService.class); // bootstrap
    @Override public Type getType() { return TYPE; }
    ...
}
```

`Sys.loadType(BX.class)` is called once at class-init. The resulting `Type` object is the
canonical identity token used everywhere — type comparisons, agent lookup, and ORD resolution
all go through `BX.TYPE` [INFER]. `[ev: organized/docSource/docSource-doc/extracted/baja/javax/baja/sys/Sys.java:443-446]`

---

## 3 · Agent-on registration — two equivalent paths

An *agent* type is one that attaches itself to a target type in the Workbench sidebar.

### Path A — XML-first in `module-include.xml` (preferred for command/handler types)

```xml
<type class="com.acme.mymod.BMyCommands" name="MyCommands">
  <agent requiredPermissions="r">
    <on type="mymod:MyService"/>
  </agent>
</type>
```

`requiredPermissions` is the permission the user needs on the target to get the agent: `"r"` = operator read,
`"rw"` = operator read + write; with none declared a view requires admin write (devguide `security/security.txt:191`,
`slot-o-matic.txt:221`). `[ev: code nmodsreflow-rt module.xml; nmodsreflow-ux module.xml]`

A ux view that opens on `ReflowService` uses the same pattern:

```xml
<type class="com.niagaramods.nmodsreflow.ux.BReflow" name="Reflow">
  <agent requiredPermissions="r">
    <on type="nmodsreflow:ReflowService"/>
  </agent>
</type>
```

### Path B — `@AgentOn` annotation (the annotation processor writes the XML)

```java
@NiagaraType(agent = @AgentOn(types = "mymod:MyService", requiredPermissions = "r"))
public class BMyCommands extends BComponent { ... }
```

javac's annotation processor (not slot-o-matic) writes the identical `<agent><on type>` XML above
(devguide `slot-o-matic.txt:193`).
Both paths produce the same runtime behaviour — choose whichever is cleaner for
the type. XML-first is typical for command types that carry no view class; `@AgentOn`
fits plain Java-first authoring. `[ev: nmodsreflow module.xml]`

---

## 4 · Cross-module type references at runtime

### 4.1 ORD form — `moduleName:TypeName`

```java
// Resolve the type object by its module-qualified name
Type t = Sys.getType("nmodsreflow:ReflowService");
```

The 2-part colon form is the storable, version-stable ORD for types.
The `moduleName` comes from `niagara-module.xml`, not the jar name. `[ev: devguide build.txt:455-458,508; organized/docSource/docSource-doc/extracted/baja/javax/baja/sys/Sys.java:421]`

### 4.2 Service lookup — same call, cross-module transparent

```java
// Works whether BMyService lives in this module or any api()-declared module
BMyService svc = (BMyService) Sys.getService(BMyService.TYPE);
```

The classloader finds `BMyService.TYPE` through the `<dependency>` chain:
consumer declares `api(project(":mymod-rt"))` → module.xml gets
`<dependency name="mymod-rt" .../>` → Niagara's `ModuleClassLoader` makes the
type visible. No extra plumbing needed. `[ev: corpus B617 §617.3; third-party-libraries.md]`

### 4.3 Storable ORD forms

| ORD scheme | Example | When to use |
|---|---|---|
| `service:mod:Type` | `service:mymod:MyService` | Resolve a service from a persisted ORD (e.g., a link target) |
| `slot:/path` | `slot:/Services/MyService` | Navigate to a concrete slot path in the station tree |
| `BOrd.make(...)` | `BOrd.make("myord:/...")` | Dispatch through a registered `ordScheme` handler |

`BOrd.make("myord:/...")` routes to the `BMyScheme` registered under `ordScheme="myord"` in
`module.xml`: `BOrdScheme.lookup(id)` asks the registry for the scheme type and instantiates it
(docSource `baja/javax/baja/naming/BOrdScheme.java:55-57`), so resolving the ORD needs no `<dependency>` on the
scheme's module; only code that names the handler class does. `[ev: corpus B35 §35.7; B617 §617.3]`
Writing the handler (own `OrdQuery`, the two exceptions, `canRead()` on every returned hop): `types/security.md`
§ 4.4. `[ev: retro devguide415-nav-subscriptions-tags-ords Δ8]`

### 4.4 Registry lookups — defs and agents `[ev: retro devguide415-horizons-authoring-security Δ9]`

- **Defs** are name/value strings declared in the part's `moduleManifest { defines { define("mymod.x", "…") } }`
  (devguide `build.txt:342-344,593`; the older `registry.txt:93` still shows `<defs>` in `module-include.xml`)
  and collapsed by the registry into ONE global map, so prefix every name with `<moduleName>.` (devguide 4.15.6.20
  `registry.txt:85-105`). Read one with `Sys.getRegistry().getDef(name, fallback)`; the one-argument form is
  `getDef(name, null)` (`javax/baja/registry/Registry.java:109-119`).
- **Agents** are queried by type filter: `file.getAgents(null).filter(AgentFilter.is(BExporter.TYPE))`. A query with
  no registered agent throws `javax.baja.agent.NoSuchAgentException`, so wrap the lookup in
  `try … catch (NoSuchAgentException)` (`registry.txt:37-50`). Registering one: § 3.

### 4.5 API stability — `com.tridium.*` is private API `[ev: retro n4-course-build-deps-permissions-signing Δ3]` `[ev: retro devguide415-horizons-authoring-security Δ9]`

- The devguide labels public APIs Stable, Evaluation or Development (`apiInfo.txt:16-22`) and says "com.tridium APIs
  should never be used by developers, and no compatibility is guaranteed" (`apiInfo.txt:54`). No label is
  machine-readable; the signals a check can read are the package prefix and `@Deprecated`.
- Prefer the `javax.baja.*` equivalent, or write the function yourself (an exporter: `types/logic-authoring.md`
  § File export, where `BHistoryToCsv` is the course's own private-API example, Lab Instructions p83).
- When a `com.tridium.*` use is unavoidable (the NDriver framework, `com.tridium.ndriver.**`, is the sanctioned
  case), record each class in the module README and pin that dependency's `vendorVersion` floor to the 3-part
  release you built and tested against (§ 1, L7). The WARN for other imports is owed Go lint G65
  `private-api-import` (ledger.d/2026-10-23-r8-u5-c59-c62.md).

---

## 5 · Version pinning summary

| Source | Stamped into |
|---|---|
| `defaultModuleVersion("X.Y.Z")` in GROUP `build.gradle.kts` | OEM sibling `<dependency vendorVersion="X.Y">` (major.minor only) |
| Tridium dep floor (e.g., `4.14.0`) in part `build.gradle.kts` | `<dependency vendor="Tridium" vendorVersion="4.14.0"/>` |
| `compileOnly(files("$niagara_home/bin/ext/jetty-all-compact3-*.jar"))` | Nothing in module.xml; jar path is build-host absolute → never commit to source |
| `uberjar(...)` | Nothing in module.xml; classes merged directly into the module jar |

---

## 6 · Checks worth adding

**`agent-on-target-exists`** (proposed lint WIR-G6): every `<on type="mod:Name">` in
`module-include.xml` must resolve to a `<type name="Name">` registered in the module
named `mod` (either in the same `module-include.xml` or in a declared `<dependency>`).
A typo produces a **silent UI failure** — the agent simply never appears in Workbench,
with no error at build or deploy time. `[ev: WIR-G6]`

**`uberjar-vs-api-dep-conflict`** (proposed lint WIR-G7): a library declared both as
`uberjar("group:artifact:ver")` and `api("group:artifact:ver")` creates a
**classloader conflict**: the `api()` path emits a `<dependency>` entry that the NRE
tries to satisfy with an external module, while the bundled copy is already present in
your jar. The module's own classes load before a declared dependency's (B617 §617.2), so the `api()` copy is
shadowed silently [INFER]; always choose one path. `[ev: WIR-G7; corpus B617 §617.2]`

---

## 7 · Reading a shipped module's manifest — `nh module`

Before declaring a dependency on a Tridium part, or copying its permission requests, read the shipped
`module.xml` through niagara-help instead of unzipping jars: `nh module <part|module|glob>` (`nh` =
`python3 ~/niagara-research/niagara-help/tools/niagara_help.py`) prints the dependencies with their floors, the
reverse dependencies (who requires this part), counts of types/agents/defs and the java-permissions by profile;
`--types`, `--agents`, `--permissions`, `--defs` and `--json` narrow it, and `--niagara-version <v>` (before the
subcommand) picks another indexed release. Excerpt, 4.15.6.20 index:

```text
$ nh module kitControl-rt
VENDOR: Tridium 4.15.6.20
DEPENDENCIES (11):
  alarm-rt       Tridium 4.15.6
  baja           Tridium 4.15.6
  bql-rt         Tridium 4.15.6
  control-rt     Tridium 4.15.6
  …
REVERSE DEPENDENCIES (2):
  kitControl-ux  requires 4.15.6
  kitControl-wb  requires 4.15.6
JAVA PERMISSIONS (15):
  station    java.lang.RuntimePermission "modifyThread"
  …
```

A shipped part's permission list is what Tridium requests, not a template for yours: request only what your code
needs (`types/structure.md` § `module-include.xml` vs `META-INF/module.xml`). Which class and agents a single type
has: `nh type` (`types/structure.md` § Naming).

---

**See also:** `types/structure.md` (L6/L7 dep-floor lint, module-include.xml vs module.xml),
`types/third-party-libraries.md` (Shadow/uberjar bundling, classloader delegation order),
`types/logic-authoring.md` (TYPE anchor lifecycle, `Sys.getService` usage).
