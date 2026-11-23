# Type: utility / library module — two shapes, one split convention

A module whose job is to be depended on, not dragged onto a palette. Two distinct shapes;
choose by what it wraps. `[ev: code clUtils-rt module.xml; code jtds-rt module.xml]`

---

## 1 · Shape A — pure 3rd-party library bundle (e.g. jtds/JDBC)

Wraps an OSS jar with zero Niagara types. The module is a classpath carrier only.
The JDBC driver for Niagara's `rdbMySQL`/`rdbOracle` is not bundled this way: it goes in `<niagara_home>/jre/lib/ext` (`types/cloud-connector.md` § 11).

```xml
<!-- jtds-rt META-INF/module.xml (other <module> attributes elided) -->
<module name='jtds-rt' autoload='true' installable='true'>
  <dependencies>
    <dependency name='baja' vendor='Tridium' vendorVersion='4.0'/>
    <!-- ONLY baja — no driver-rt, no control-rt -->
  </dependencies>
  <types/>   <!-- empty — no @NiagaraType, no palette, no lexicon -->
  <permissions>
    <java-permissions type='workbench'>
      <java-permission name='getClassLoader' class='java.lang.RuntimePermission'/>
    </java-permissions>
    <java-permissions type='station'>
      <java-permission name='*:1-100000' action='accept,connect,listen,resolve'
                       class='com.tridium.nre.security.NiagaraSocketPermission'/>
      <java-permission name='modifyThread'     class='java.lang.RuntimePermission'/>
      <java-permission name='modifyThreadGroup' class='java.lang.RuntimePermission'/>
      <java-permission name='shutdownHooks'    class='java.lang.RuntimePermission'/>
    </java-permissions>
  </permissions>
</module>
```

`[ev: code jtds-rt module.xml]`

| Rule | Reason |
|------|--------|
| `autoload='true'` | NRE loads it before any consumer so `Class.forName(...)` resolves |
| `<types/>` empty | No `@NiagaraType`, no palette, no lexicon needed |
| Only `baja` dep | The wrapped lib has no Niagara API dependency |
| `getClassLoader` RuntimePermission | JDBC `DriverManager.getConnection` triggers class loading |
| `NiagaraSocketPermission` station | The lib opens TCP connections to the DB server |

Consumer: `Class.forName("net.sourceforge.jtds.jdbc.Driver")` + `DriverManager.getConnection(...)`.
No concrete import from the lib jar — the boundary is classpath only.

A library module declares its kind in the module map, or `lint-structure` L5 FAILs its `-rt` part (§ 6).

---

## 2 · Shape B — helper-type library (e.g. clUtils)

Registers agent/abstract helper types that auto-attach to driver or framework types. **No palette**
(agent helpers are framework-wired, not dragged; declare the kind, § 6). **Yes lexicon** (L4 requires ≥ 1 `key=value` per type).

```xml
<!-- clUtils-rt META-INF/module.xml (other <module> attributes elided) -->
<module name="clUtils-rt" autoload="true" installable="true">
  <dependencies>
    <dependency name="baja"         vendor="Tridium" vendorVersion="4.15"/>
    <dependency name="driver-rt"    vendor="Tridium" vendorVersion="4.15"/>
    <dependency name="alarm-rt"     vendor="Tridium" vendorVersion="4.15"/>
    <!-- all deps the helper functionality actually needs — NOT just baja -->
  </dependencies>
  <types>
    <type class="com.tridium.clUtils.device.BDeviceInfoHelper" name="DeviceInfoHelper">
      <agent>
        <on type="driver:Device"/>   <!-- binds to the base type — widest coverage -->
      </agent>
    </type>
    <!-- abstract helpers with no agent binding -->
    <type class="com.tridium.clUtils.util.BProxyPointHelper"    name="ProxyPointHelper"/>
    <type class="com.tridium.clUtils.util.BHistoryImportHelper" name="HistoryImportHelper"/>
  </types>
  <permissions>
    <java-permissions type="station">
      <java-permission action="connect,resolve"
                       class="com.tridium.nre.security.NiagaraSocketPermission" name="*:1-100000"/>
      <java-permission class="java.lang.RuntimePermission" name="modifyThread"/>
    </java-permissions>
  </permissions>
</module>
```

`[ev: code clUtils-rt module.xml]`

The base helper keeps a static map: on first use it asks the registry for every concrete subtype
(`Sys.getRegistry().getConcreteTypes(...)`) and each registers itself through the protected static `registerHelper()`;
nothing runs at module load (`organized/clUtils/clUtils-rt/vineflower/com/tridium/clUtils/device/BDeviceInfoHelper.java:30-55`).
Agent types are registered on their target type through the registry — no station wiring by the operator.

---

## 3 · Protocol-specialization split convention

A base library (`<base>-rt`) holds generic helpers with **no protocol dep**. Each protocol
specialization (`<base><Protocol>-rt`) adds its driver deps and **narrows** `<on type>`:

```
clUtils-rt            ← no bacnet-rt / niagaraDriver-rt
                         <on type="driver:Device">              ← widest

clUtilsBacnet-rt      ← adds bacnet-rt + niagaraDriver-rt + fox-rt + schedule-rt + clUtils-rt dep
                         <on type="bacnet:BacnetDevice">        ← narrowed

clUtilsNiagara-rt     ← adds niagaraDriver-rt + niagaraVirtual-rt + fox-rt + schedule-rt + clUtils-rt dep
                         <on type="niagaraDriver:NiagaraStation"> ← narrowed
```

`[ev: code clUtilsBacnet-rt module.xml; code clUtilsNiagara-rt module.xml]`

**Why split:** the base stays installable on stations without the driver module. A `bacnet-rt` dep
in the base would force it onto every platform even when BACnet is absent. Each specialization
ships only where the matching driver is installed.

---

## 4 · Consumer wiring

In the consumer's `module-include.xml`:

```xml
<dependency name="clUtils-rt" vendor="Tridium" vendorVersion="4.15"/>
```

This is a **runtime classpath dep**. The Gradle `api(":<lib>-rt")` in `build.gradle.kts` covers
compile-time visibility; the `module.xml` `<dependency>` makes the NRE load the dep jar before
yours at runtime. Both are required — see `types/module-wiring.md` for the api()/module.xml duality.

---

## 5 · When to build a utility-lib vs. inlining

| Situation | Decision |
|-----------|----------|
| Shared infra reused by 2+ of your modules | Extract into `<base>-rt` utility-lib |
| Wrapped OSS jar (JDBC, JSON, CSV parser) | Shape A — empty `<types/>` |
| Protocol-agnostic helpers | Shape B base + optional protocol split |
| Logic used only in one module | Inline — separate module adds install/version friction |

**Gotcha:** `autoload='true'` loads the module at station startup regardless of whether any
station component currently uses it. Keep utility-libs free of heavy static initialisers — they
run on every station start even on platforms that have no active consumer.

---

## 6 · Structure lint — declare `type: library`

`lint-structure` L5 FAILs an `-rt` part with no `module.palette` (`L5: module.palette missing for -rt profile`, followed by how to declare a library): on a
component module that is the empty-palette footgun. `[ev: retro doc-coverage-audit Δ22]` Both shapes above ship no palette on purpose, and the lint never
infers a library from the missing file. The module says it in the header of `<module-root>/MODULE-MAP.md`
(`types/structure.md` § Module map):

```
# MODULE-MAP — acmeJdbc

modularity: strict
type: library

| Concern | Part | Files | Tests | Change here when |
|---|---|---|---|---|
| plumbing | rt | acmeJdbc-rt/acmeJdbc-rt.gradle.kts; acmeJdbc-rt/module-include.xml; acmeJdbc-rt/module-permissions.xml | — | build, packaging or the wrapped library's version |
```

With `type: library` an `-rt` part may ship no `module.palette`. A palette it does ship is judged like any other
(no named entry is still an L5 FAIL), and every other rule applies unchanged: Shape B still needs its lexicon (L4)
and, in a strict module, its `model/` classes and tests (L3). Pinned by the `kind-library-*` golden cases of
`go/internal/lint/rules/structure` and `TestDeclaredKindMapPassesModuleMap`.

**Start from the scaffold:** `toolbelt/scaffold-module.sh <ModuleName> <out-dir> --vendor <v> --type library` emits Shape A
(the `-rt` part of `<types/>`, a permissions file and the gradle root, from `fixtures/MinimalLibrary`) with the
`type: library` line already in its `<module-root>/MODULE-MAP.md`; `lint-structure`, `lint-module-map` and the permissions lint pass it
clean (`tests/scaffold-module-map.bats` TC-MAP-LIB). The fixture has not been built with gradle (no Niagara install
in the session that added it): run `build.sh` on the first scaffold before relying on it. `[ev: retro doc-coverage-audit Δ10]`

---

**See also:** `types/module-wiring.md` (api()/module.xml mechanics), `types/structure.md`
(permissions blocks, module-include.xml authorship), `types/driver-authoring.md` (`<on type>` binding).
