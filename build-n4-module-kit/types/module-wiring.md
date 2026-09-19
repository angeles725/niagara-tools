# Module wiring — Gradle deps, type registration, agent-on, cross-module references

How a module declares dependencies on other modules, registers its own types, and
calls into types across module boundaries. `[ev: code nmodsreflow gradle.kts; corpus B802; real module.xml]`

---

## 1 · Gradle dependency configurations → `module.xml` impact

| Gradle declaration | `module.xml` effect | Notes |
|---|---|---|
| `nre(":nre")` | Sets `nre="true"` attribute on `<module>` — NO `<dependency>` entry | Marks the module as NRE-level; every production module should carry this |
| `api(":baja")` / `api(":web-rt")` | Emits `<dependency name="baja" vendor="Tridium" vendorVersion="X.Y.Z"/>` | Transitive: a module that `api()`s yours also sees baja; 3-part floor required (L7) |
| `api(project(":mymod-rt"))` | Emits `<dependency name="mymod-rt" vendor="<OEM>" vendorVersion="<major.minor>"/>` | Sibling part dep; vendor and version stamped from `defaultModuleVersion` in the GROUP `build.gradle.kts`, NOT from the part's own `.kts` `[ev: corpus B802; nmodsreflow-ux module.xml]` |
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

---

## 2 · Own-type registration (`module-include.xml` → `module.xml`)

You author `module-include.xml`; the gradle plugin generates `META-INF/module.xml` from it.
**Never hand-author `META-INF/module.xml`** — it will be overwritten (lint L6). `[ev: corpus B817]`

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
all go through `BX.TYPE`. `[ev: corpus B802 §802.3]`

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

`requiredPermissions` governs who sees the agent in Workbench: `"r"` = read-only user;
`"rw"` = operator; `""` = admin only. `[ev: code nmodsreflow-rt module.xml; nmodsreflow-ux module.xml]`

A ux view that opens on `ReflowService` uses the same pattern:

```xml
<type class="com.niagaramods.nmodsreflow.ux.BReflow" name="Reflow">
  <agent requiredPermissions="r">
    <on type="nmodsreflow:ReflowService"/>
  </agent>
</type>
```

### Path B — `@AgentOn` annotation (slot-o-matic generates the XML)

```java
@NiagaraType(agent = @AgentOn(types = "mymod:MyService", requiredPermissions = "r"))
public class BMyCommands extends BComponent { ... }
```

Slot-o-matic expands this into the identical `<agent><on type>` XML above.
Both paths produce the same runtime behaviour — choose whichever is cleaner for
the type. XML-first is typical for command types that carry no view class; `@AgentOn`
fits plain Java-first authoring. `[ev: corpus B802 §802.5; nmodsreflow module.xml]`

---

## 4 · Cross-module type references at runtime

### 4.1 ORD form — `moduleName:TypeName`

```java
// Resolve the type object by its module-qualified name
Type t = Sys.loadType("nmodsreflow:ReflowService");
```

The 2-part colon form is the storable, version-stable ORD for types.
The `moduleName` comes from `niagara-module.xml`, not the jar name. `[ev: corpus B35]`

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
`module.xml`. The scheme handler's class must be visible to the caller's classloader
(i.e., the module carrying it must be in the `<dependency>` chain). `[ev: corpus B35 §35.2]`

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
your jar. The runtime picks one unpredictably; always choose one path. `[ev: WIR-G7]`

---

**See also:** `types/structure.md` (L6/L7 dep-floor lint, module-include.xml vs module.xml),
`types/third-party-libraries.md` (Shadow/uberjar bundling, classloader delegation order),
`types/logic-authoring.md` (TYPE anchor lifecycle, `Sys.getService` usage).
