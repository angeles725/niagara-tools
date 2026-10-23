# Type: module structure — the layout, naming, and a conformance lint (`lint-structure.sh`)

The shape a Niagara N4 module must have on disk, and the L1–L11 checklist `lint-structure.sh` enforces. Grounded
in how Tridium/Honeywell lay out their own modules and an audit of our four. `[ev: corpus B817]`

## Layout — one `moduleName`, per-profile parts `[ev: corpus B817]`
- A module splits into per-`runtimeProfile` artifacts `<module>-{rt,ux,wb,se,doc}`: `-rt` runtime (compact3, every
  host), `-ux` browser (BajaScript), `-wb` Workbench (Swing), `-se` code that depends on Java SE APIs (JDBC, AWT,
  Swing; § The `-se` runtime profile), `-doc` help. The parts are declared in the root `niagara-module.xml`; the
  plugin writes the `<moduleParts>` element of the generated `META-INF/module.xml`, never an authored file.
  `[ev: corpus B1313 §1313.3 C2-C3]` `[ev: retro devguide415-horizons-authoring-security Δ8]`
- **Allowed part dependencies** (devguide `modules.txt:148-155`): `rt`→rt · `ux`→rt, ux · `wb`→rt, ux, wb ·
  `se`→rt, ux, wb, se · `doc`→none, and nothing depends on a `doc` part. An `api(":<mod>-wb")` in an `-rt` or `-ux`
  part's build script is a layering error.
- Help ships as a `doc` PART of the code module (the devguide's form, `deployingHelp.txt:8`) or as a separate
  `doc*` module; both are legitimate (§ -doc help profile authoring recipe).
- Author-side source tree: `<MOD>-rt.gradle.kts` + `module-include.xml` + `module.lexicon` + `module.palette` +
  `src/com/<vendor>/<MOD>/…` (+ `srcTest/` for tests, `rc/` for `-ux`). `[ev: corpus B817]`
- **One Java package belongs to one module part.** Code the `-rt` and `-wb` parts both need is either duplicated
  or moved into its own utility module (`types/utility-lib.md`); a package is never split across parts or modules.
  N4 loads the first copy and shadows the other; Niagara 5 refuses split packages. Check:
  `toolbelt/split-package-check.sh` (`types/issues-and-gotchas.md` D4c). `[ev: corpus B1417 §1417.3 a-b]` `[ev: retro course-recordings Δ1]`

## Naming — `javax.baja.*` = framework API, `com.<vendor>.*` = your code `[ev: corpus B817]`
- Tridium declares PUBLIC framework types under `javax.baja.<domain>` (e.g. `javax.baja.control.BControlPoint`) and
  keeps implementation under `com.tridium.<module>`. Honeywell/OEM modules use `com.<vendor>.<module>` throughout
  (ours: `com.angeles.<Module>`) — a leaf/OEM module NEVER declares `javax.baja.*` types.
- One public `@NiagaraType` per `.java` file; `BXxx` class naming. `[ev: corpus B817]`
- A `com.tridium.*` import is private API (`types/module-wiring.md` § 4.5).

## `module-include.xml` vs `META-INF/module.xml` `[ev: corpus B817]`
- You AUTHOR `module-include.xml` (the `<type>` list). The gradle plugin GENERATES `META-INF/module.xml` from it.
- **After renaming or deleting a type, delete `module-include.xml` (or its stale `<type>` line) and rebuild.** The
  build adds lines for new `@NiagaraType` classes but never rewrites or removes an existing line, so a renamed
  type keeps a dangling registration (METHODOLOGY § Schema / upgrade safety). `[ev: corpus B1417 §1417.3 e]` `[ev: retro course-recordings Δ1]`
- A hand-authored `META-INF/module.xml` in source is WRONG (L6): it will be overwritten and drifts from the plugin
  output. Edit `module-include.xml`; never the generated manifest. `[ev: corpus B817]`
- **`module-permissions.xml` — developer source-only permission declaration (JVM sandbox layer):** a module that needs JVM-level capabilities (network access, reflection, JMX) declares SEMANTIC group names in `src/module-permissions.xml` using `<niagara-permission-groups type="…"> / <req-permission> / <name>NETWORK_COMMUNICATION</name> / <purposeKey>…</purposeKey> / <parameters>…`. This is a distinct concern from the RBAC bit checks in `dashboard.md §RBAC` and from `module-include.xml`; it governs what the module's JVM code is ALLOWED to do, not what a station user is allowed to do. [ev: corpus B18 §18.4.1] Six of the groups (REFLECTION, ACCESS_CLASS, MBEAN_PERMISSION, THIRD_PARTY_PERMISSION, PROTECTION_DOMAIN, HSM_SIGNING) also need a signed, trusted module, or the station drops them with a WARNING (`build-verify.md` § Signing per deploy target). [ev: corpus B1312 §1312.3 C1]
- **Source→runtime inlining:** the Gradle signing plugin (`NiagaraPermissionGroupFactory`) expands the high-level group names into concrete `<java-permission class="…">` entries and INLINES them into the built JAR's `META-INF/module.xml` `<permissions>` element. There is NO `module-permissions.xml` in the deployed artifact (confirmed zero across all corpus JARs). To inspect deployed grants: `unzip -p <mod>-rt.jar META-INF/module.xml | grep -A20 permissions`. [ev: corpus B18 §18.4.2-18.4.3; B777 §777.4]
- **Profile scoping** — each `<niagara-permission-groups>` block carries `type="station"`, `"workbench"`, or `"all"`. A `type="workbench"` group is never granted on the station daemon. A permission needed by the NRE MUST be declared under `type="station"` or `"all"`; declaring it only under `type="workbench"` silently leaves the daemon ungrated. Verify the deployed partition: `unzip -p <mod>-rt.jar META-INF/module.xml | grep -B2 'java-permission'`. [ev: corpus B18 §18.4.7]
- **Start from `templates/module-permissions.xml`** (the wizard shape: `<permissions>` root, three
  `<niagara-permission-groups>` blocks `all`/`workbench`/`station`, devkit-wb 4.14.0.162
  `gradle/includes/niagara/module-permissions.xml.vm:1-19`, with commented NETWORK_COMMUNICATION, REFLECTION,
  MANAGE_EXECUTION and `<opt-permission>` requests as in course `Labs/Lab9 Multithreaded/module-permissions.xml`).
  `<name>` and `<purposeKey>` are both required; `purposeKey` may be a lexicon key of the module, else the text is
  shown as-is (devguide 4.15.6.20 `security/requestingPermissions.txt:40-80`). `[ev: retro n4-course-authoring-files-and-libraries Δ2]`
- **When a privileged call fails** with `AccessControlException`: read the `java.*` permission it names, find the
  group that grants it in devguide `security/requestingPermissions.txt`, add the request, rebuild. For a feature
  that can run without it, request it as `<opt-permission>` and catch `AccessControlException` where it is used
  (`requestingPermissions.txt:51-54,142`).
- **Security Manager card** (devguide 4.15.6.20 `security/securityManager.txt`; 4.13/4.14 not verified):
  - Debug: the `security.niagaraPolicy` log at FINE/FINER/FINEST (DebugService or Logger Configuration, `:44`), or
    start with `-@Djava.security.debug=access,failure` (`:58-62`).
  - Disabling it is development-only: it needs the `smDeveloperMode` licence feature plus
    `niagara.security.manager.disable` (`:86-98`); from 4.9 the denials are logged to a file while the app runs
    on (`:112`).
  - File I/O needs no permission under `Sys.getStationHome()` and `Sys.getNiagaraSharedUserHome()` (`:122-124`);
    write there, not to an absolute path.
  - What a module was granted: Spy › securityInfo › Policy Spy (`requestingPermissions.txt:653`); the startup
    console listing is turned off with `niagara.security.logPermissionGroups=false` (`:596`).
  `[ev: retro devguide415-horizons-authoring-security Δ5]` `[ev: corpus B1312 §1312.1]`

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

### Lexicon key catalogue (what L4's keys must look like)

The framework reads only these forms; any other key is dead for that purpose. `[ev: retro n4-course-lexicon-logging-testing Δ1]`
`[ev: retro devguide415-lexicons-ux-i18n Δ1]`

| Key | Shown as | Read by |
|---|---|---|
| `<slot>=` (bare slot name) | the slot's display name | `NSlot.java:42` `Lexicon.make(module, cx).get(this.name)` |
| `<Type>.displayName=` | the type's display name (palette, New menu) | `NTypeInfo.java:178` `get(getTypeName() + ".displayName")` |
| `<tag>=` (bare enum tag) | the frozen enum value's display text | `EnumType.java:50` `get(entry.tag)` |
| `%lexicon(<module>:<key>)%` in a facet value | e.g. `TRUE_TEXT`/`FALSE_TEXT` text | `BFormat.java:77,452` (a BFormat call, any key) |

All [CERT]: the first three under `organized/baja/baja/vineflower/com/tridium/sys/{schema,registry}/`, `BFormat.java` under
`organized/docSource/docSource-doc/extracted/baja/javax/baja/util/`; cross-read in corpus B1305 §1305.1 and B1236 §1236.3.
A `Type.slot` key is never read for the slot, and a bare `<Type>=` key does nothing for the type (it falls back to
`toFriendly(<Type>)`, `NTypeInfo.java:183`). `toolbelt/slot-coverage.sh` checks both: the parse mode
counts a type only through `<Type>.displayName` and WARNs a bare `<Type>=`; `per-slot` counts only bare slot keys and
prints a `DOTTED` row for a `Type.slot` key that is the only key of an annotated slot. A coverage check counts only
the key forms the framework looks up; a form it ignores is reported, never counted. `[ev: retro r8-t9g-actions Δ2]`

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

7. **New-module bring-up — align to a sibling of the same SDK:** a fresh scaffold's
   `gradle.properties` JDK/node block is COMMENTED (auto-detect). Before the first build, copy
   `niagara_home`, `niagara_user_home`, `nodeHome`, and `org.gradle.java.installations.paths`
   from an existing sibling module that deploys to the same station and SDK family — all modules
   in the group must target one consistent `niagara_home`. For a **ux/JS module** (`-ux` profile)
   the node and JDK path lines are REQUIRED (points 5–6 above); a logic-only module with no `-ux`
   profile may leave them commented. Also align `gradlePluginVersion` and `settingsPluginVersion`
   in `settings.gradle.kts` to the sibling's SDK family (each Niagara install ships exactly one
   plugin version — see `build-verify.md`). `[ev: corpus B1016]`

## JDK-pinning block in `gradle.properties` (REQUIRED for every module)

**`gradle.properties` MUST carry the JDK-pinning block** (`org.gradle.java.installations.paths`, `org.gradle.jvm.toolchain.auto-detect=false`, `org.gradle.jvm.toolchain.auto-download=false`). A module without this block triggers `lint-structure` FAIL on every build. The scaffold MUST emit the pinning block (commented-out or with a placeholder) so a fresh checkout does not immediately fail the lint. For the full pinning recipe and what each key does, see §L10 doctrine above. `[ev: retro apillm-headless-servlet-rt-4.14-deltas Δ16]`

## Slotomatic checksum — use `0`, not `(auto)`

When hand-adding a `@NiagaraProperty` to an already-generated class (outside Slotomatic's auto-region), set the auto-region checksum to the plain integer `0`, NOT the literal placeholder `(auto)`. The `(auto)` text causes a cosmetic Slotomatic parse error (the build still passes but emits a noisy WARNING about an unrecognized checksum token); `0` causes Slotomatic to regenerate the checksum cleanly on the next build. If Slotomatic's region comment shows `/* auto */ (auto)` after a hand-edit, replace it with `/* auto */ 0`. `[ev: retro apillm-headless-servlet-rt-4.14-deltas Δ24]`

## Slot-o-Matic runs — read the output, recover a failed file `[ev: retro course-recordings Δ5]`
- **Read the Slot-o-Matic output on every run:** a file it cannot process is logged and skipped, the task still
  succeeds, and the build fails later (or not at all) in `compileJava`. `[ev: corpus B1417 §1417.3 s]`
- **Never write code inside the generated `BEGIN … END BAJA AUTO GENERATED CODE` region:** the next run replaces
  the region and the code is gone. Hand-written code goes outside it. `[ev: corpus B1417 §1417.3 r]`
- **"Failed to process" / "Could not process":** delete the whole generated region and rerun; it regenerates from
  the annotations (an empty pre-written region is METHODOLOGY K15). `[ev: corpus B1418 §1418.5 a]`
- **Annotation types are Java names:** a lower-case `type = "string"` breaks the run; use `String`/`BString` as the
  property card says, and remove unused auto-imports the IDE added. `[ev: corpus B1418 §1418.5 d]`
- **"java file must belong to a package"** means the `package` line was lost in a copy-paste. `[ev: corpus B1418 §1418.5 b]`

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

### What the scaffold emits today: two hand steps
- **Replace the scaffolded `module-permissions.xml` before adding a request.** Both fixtures, and so every
  `scaffold-module.sh` module, ship `<?xml …?><modulePermissions/>`. The gradle plugin copies that root into
  `META-INF/module.xml` as it is (niagara-plugins 7.6.22 `NiagaraModulePlugin` adds `module-permissions.xml` to
  the include files, by javap; a scaffolded client jar's manifest holds `<modulePermissions/>`), and the NRE reads
  only a `<permissions>` element (baja 4.14.0.162 vineflower `com/tridium/sys/module/NModule.java:442-452`). A
  `<req-permission>` put inside `<modulePermissions>` is never read, and the call fails with
  `AccessControlException`. Start from `templates/module-permissions.xml` (§ `module-permissions.xml` above). The
  fixture fix is owed as G98 `fixture-permissions`. `[ev: retro n4-course-authoring-files-and-libraries Δ2]`
- **No `moduleTest-include.xml` is emitted yet.** Before the first `BTestNg` class, copy
  `templates/moduleTest-include.xml` (course shape: a bare `<types>` root) into the part. Do not paste the stub
  block above as written: its comment comes before the `<?xml` declaration, so it is not well-formed XML (expat:
  "XML or text declaration not at start of entity"). The emission is owed as G97 `fixture-moduletest-include`.
  `[ev: retro testing-parity-btestng Δ6]`

## Palette assembly templates `[ev: corpus B746 §746.3]`

A `module.palette` `<p>` entry can nest child `<p>` elements with property overrides to form
pre-configured, commissioning-ready **assembly templates**. Dragging one component from the palette
drops the correct sub-tree with every flag already set — no manual wiring required.

A palette assembly is NOT a Niagara template. A template is a `.ntpl` file deployed from the Template sidebar or
the Device Manager, with its own manifest, deploy, status and upgrade rules (`types/logic-authoring.md` § template
bullet, `types/distribution.md` § 8); a palette drop is a plain copy that nothing tracks afterwards.
`[ev: corpus B1325 §1325.2 C2]` `[ev: retro devguide415-r2-packaging-platform-signing Δ4]`

### Authoring workflow

1. **Build the assembly correctly in a scratch station** — create the component, add its children
   (`BDefrostController`, sub-evaporators, etc.), and configure every slot to its intended default.
2. **Copy it into the palette:** in Workbench, open the module's palette view, drag the assembled
   component into it, then **Save** (`File › Save All`). The Workbench serialises the live component
   subtree into BOG XML inside `module.palette`.
3. **Alternatively, hand-edit `module.palette`:** nest child `<p>` elements inside the parent `<p>` and set
   each slot override as a child element `<p n="<slot>" v="<value>"/>` (see the format card below).

### Format card `[ev: retro n4-course-authoring-files-and-libraries Δ3]`

A palette is a BOG document, decoded like a `.bog` (`bajaObjectGraph`). Tridium's own palettes follow it
(`kitControl-rt.jar!module.palette:1-11,130-135`, 4.15 corpus `organized/kitControl/kitControl-rt/extracted/`):

- Root `<p m="b=baja" t="b:UnrestrictedFolder">`; nested folders `<p n="<Folder>" t="b:UnrestrictedFolder">`.
- A module symbol is declared once, `m="<x>=<moduleName>"`, on the first element that uses it (an earlier
  sibling or an ancestor carries it for the rest); entries use `t="<x>:<TypeName>"` — the registered type name,
  never a Java class name. The decoder rejects `t=` without a symbol (`Invalid typespec`) and an undeclared
  symbol (`Undeclared module symbol`). [CERT] `javax/baja/io/ValueDocDecoder.java:1071-1073,1091` (baja vineflower)
- `n=` is the palette label (slot-escaped: `$20` for a space); its position among the attributes does not
  matter (Workbench saves it last).
- Slot overrides are CHILD elements: `<p n="<slot>" v="<value>"/>` for a frozen slot, plus `t=` and `f=` for
  a dynamic slot (`<p n="inA" f="sx" t="b:Boolean" v="false"/>`, kitControl :131).
- List only draggable component types — not helper `BWorker`/`BStruct`/POJO types held as properties.
  Restricted services (`BIRestrictedComponent`) only under `b:UnrestrictedFolder` [INFER: course guidance, not traced].

### Example: defrost-enabled cold-room template

This template bakes out the B731 trap (`hasDefrost=false` + `airDefrost=true` → defrost silently
disabled) by setting both flags correctly in the palette entry so every drag-drop starts correct:

```xml
<p m="b=baja" t="b:UnrestrictedFolder">
  <!-- b:UnrestrictedFolder, never b:Folder: b:Folder is access-controlled and causes "access denied"
       palette expansion for engineers without the folder's category permission. [ev: corpus B746 §746.1] -->
  <p n="ColdRoom" m="cr=ColdRoomPan" t="cr:ColdRoomPan">
    <p n="hasDefrost" v="true"/>
    <p n="airDefrost" v="true"/>
    <!-- a pre-built child component: its own entry, its overrides as children -->
    <p n="defrost" t="cr:DefrostController">
      <p n="interval" v="PT4H"/>
      <p n="duration" v="PT30M"/>
    </p>
  </p>
</p>
```

### Key rules

- Every folder (root or nested) uses `t="b:UnrestrictedFolder"`, never `t="b:Folder"`: the latter is
  access-controlled and causes silent "access denied" palette expansion in Workbench for engineers who lack
  the folder's category permission. `verify-module.sh` WARNs (`palette-root`) on every `b:Folder`.
  `scaffold-module.sh` copies `fixtures/MinimalPan`, whose palette root is `b:UnrestrictedFolder`, so a
  freshly scaffolded module carries no `palette-root` WARN. `[ev: corpus B746 §746.1]`
  `[ev: retro apillm-headless-servlet-rt-4.14-deltas Δ4]`
- **Type spec = `<alias>:<TypeName>` without the `B`** (`cr:ColdRoomPan` for `BColdRoomPan`). A Java class
  name has no alias and fails as `Invalid typespec`; a `B`-prefixed name under a declared alias fails with "Type not
  found". Declare each alias once, `m="<alias>=<module>"` on the first `<p>` that uses it; later entries reuse the
  alias. `[ev: corpus B1418 §1418.5 e]` `[ev: corpus B1419 §1419.4 d]` `[ev: corpus B780]` `[ev: retro course-recordings Δ2]`
- `verify-module.sh` FAILs (`palette-parse`) an unterminated `<!--`, FAILs (`palette-type`) a `t=` that is a Java class name or whose symbol no `m=` declared,
  and counts as entries only `<p>` elements with `n=` and a non-folder `t=` in any attribute order, skipping
  `v=` overrides — a Workbench-saved palette is no longer reported empty. A check over a framework file reads
  it the way the framework does, never by a literal text pattern.
  `[ev: retro n4-course-handson-slotomatic-authoring Δ6]` `[ev: retro r8-t9g-actions Δ1]`
- Override values are applied at drag-drop time; they are not persisted unless the component itself
  persists them.
- Assembly templates are **pure resource additions** — no code change, no class risk.
- Every child `<p n=…>` that should appear in the Workbench drag-palette at the top level needs its
  own `n=` attribute; nested children are part of the parent template only, not individually draggable.

## -doc help profile authoring recipe `[ev: corpus B759 §759.6]`

A `-doc` part has its own runtime profile (`runtimeProfile="doc"`) and no types. The devguide puts help
"in a part with a runtime profile of doc" of the code module (`deployingHelp.txt:8`); a separate `doc*`
module is also legitimate. Its sole job is to ship HTML help pages that Workbench's F1 help system finds in
the module's own `doc/` directory. `[ev: corpus B1313 §1313.3 C1]` `[ev: retro devguide415-horizons-authoring-security Δ7]`

Our modules ship none today — that is architecturally correct. This recipe documents the path for
any future help investment.

### File layout

```
<Mod>-doc/
├── <Mod>-doc.gradle.kts         # runtimeProfile = "doc"
├── module-include.xml           # <types/> — intentionally empty [ev: corpus B784 §784.1]
├── module.lexicon               # help.guide.base is optional (see below)
└── src/
    └── doc/
        ├── toc.xml              # JavaHelp 1.0 TOC descriptor
        └── <mod>-<TypeName>.html  # Guide-on-Target HTML, one per type
```

### toc.xml structure (JavaHelp 1.0 TOC DTD)

```xml
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE toc PUBLIC "-//Sun Microsystems Inc.//DTD JavaHelp TOC Version 1.0//EN"
  "http://java.sun.com/products/javahelp/toc_1_0.dtd">
<toc version="1.0">
  <tocitem text="My Module Guide" target="mymod-overview" image="topic">
    <tocitem text="BColdRoomPan" target="mymod-BColdRoomPan"/>
    <tocitem text="BDefrostController" target="mymod-BDefrostController"/>
  </tocitem>
</toc>
```

The `target=` value maps to a filename in `src/doc/` without the `.html` suffix.

### HTML naming conventions

| Purpose | File name |
|---------|-----------|
| Guide-on-Target (type help) | `doc/<mod>-<TypeName>.html` |
| On-View (view help) | `doc/<mod>-<ViewTypeName>.html` |

Example: help for `BColdRoomPan` in module `coldroomrt` → `doc/coldroomrt-BColdRoomPan.html`.

### Lexicon key (optional — usually omit it)

Guide on Target looks for the HTML file in the `doc/` directory of the type's own module when the lexicon
defines no `help.guide.base` (`deployingHelp.txt:28`), so a vendor module omits the key. Set it only to point at
ANOTHER module that holds the help; core Tridium modules point it at `docUser`, which a vendor module must never
copy (its types would open Tridium's guide, not ours). The type name is appended to the base to form the
full HTML path.

### Build plugins

Two Gradle plugins cooperate:

| Plugin | Role |
|--------|------|
| `com.tridium.niagara-doc` | Packages help HTML + TOC into the `-doc` jar |
| `com.tridium.bajadoc-module` | Generates API Javadoc pages from type annotations |

Both are applied in `<Mod>-doc.gradle.kts`. The `niagara-doc` plugin is the mandatory one for
F1-accessible help; `bajadoc-module` adds the API documentation layer.

`[ev: corpus B759 §759.6]`

## Nav files `[ev: corpus B35 §35.5.3]` `[ev: retro devguide415-nav-subscriptions-tags-ords Δ5]`

A `.nav` file is a static nav tree a user sees instead of the component tree. **Nothing discovers it**: a file
is used only when a user's `navFile` ORD property points at it (`BUser.navFile`, `javax/baja/user/BUser.java:621`)
or the user's profile names it as the mobile nav file. Keep it under the station's Files and assign the ORD on
the user or the user prototype. No Tridium module ships one: 0 `.nav` files across 728 jars (corpus B1353 §1353.1),
so an `rc/nav/` convention inside a module jar is not a framework feature.
`[ev: retro devguide415-r2-runtime-contracts-formats Δ5]` `[ev: corpus B1317 §1317.2]`

### XML schema

```xml
<?xml version="1.0" encoding="UTF-8"?>
<nav version="1.0">
  <node name="My Module" ord="station:|slot:/Services/MyService" icon="module://mymod/icons/x16/service.png">
    <node name="Status"  ord="station:|slot:/Services/MyService/status"/>
    <node name="Config"  ord="station:|slot:/Services/MyService/config"/>
  </node>
</nav>
```

- The root is `<nav version="1.0">` with exactly one root `<node>`: the decoder throws `Root element must be "nav"`,
  `Only version 1.0 is supported` or `Missing root <node>` otherwise.
- Each `<node>` reads `name` (label), `ord` (target, any scheme) and optional `icon`; nodes nest, in document order.
- [CERT] `javax/baja/nav/NavFileDecoder.java:127-138,166-168` (niagara-help 4.15.6.20 source)

### Cache

`NavFileDecoder` caches each parsed file by its absolute ORD and re-parses it when the file's `lastModified`
changes (`NavFileDecoder.java:51-71`), so an edited file is picked up on its next load; there is no
session-lifetime staleness to design around. `[ev: corpus B1307 §1307.2 C2]`

## The `-se` (Java SE) runtime profile

A `-se` part is needed when code requires Java SE APIs that Niagara's Compact 3 profile does not
provide: `java.awt` / `javax.swing`, `java.sql` (JDBC), or native libraries linked via JNI.
It is built against Java 8 SE — NOT synonymous with "JACE-exclusive".
`[ev: devguide modules.txt]`

### Load profile on each host type

| Host | Profiles loaded at startup |
|------|---------------------------|
| Station daemon on a Windows-based host (Supervisor) launched with `-rp:se` | `{rt, se}` — rt is always prepended |
| JACE | never `-se`: the platform guide says `se` jars "can run on Windows-based hosts only" and a JACE runs a Java 8 compact3 VM |
| Workbench | `{rt, ux, wb}` (+ `se` when `-rp:se` passed) |

A `-wb` part is **invisible** to any station daemon regardless of `-rp` flags. Keep JACE logic in `-rt`.
`[ev: corpus B630]` `[ev: corpus B1341 §1341.2 C1]` (guide `Platform/aPlat4PlatformAdminConfigureRuntimeProfiles.txt:24,30`;
the JACE row is doc-proven, not run on a JACE) `[ev: retro devguide415-r2-packaging-platform-signing Δ8]`
`[ev: retro r8-t9i-packaging Δ2]`

### Dependency trap — `-wb` deps in a `-se` module.xml

A `-se` `module.xml` may legitimately declare dependencies on `-wb` modules (they satisfy the
compiler). Example: `alarm-se` at 4.14.0 lists `alarm-wb`, `bajaui-wb`, and `export-wb` as
dependencies. Those classes are **not loaded** on a headless daemon (`-rp:se` loads only `{rt,se}`),
so any `-se` code path that calls into a `-wb` class at runtime on a Supervisor daemon throws
`ClassNotFoundException`. `[ev: code alarm-se module.xml; corpus B630]`

**Rule:** guard every `-wb` class reference in a `-se` part behind a runtime `Sys.getProfile()`
check, or move the wb-dependent code into the `-wb` part and keep `-se` wb-free.

### Headless AWT trap

A Supervisor station daemon runs headless. Display-dependent classes
(`JFrame`, `JDialog`, any component that opens a native window) fail at runtime with
`HeadlessException`. Headless-safe AWT is fine: `java.awt.print`, `Font`, image I/O,
`java.awt.AWTPermission` (clipboard, window-without-warning-banner) — `obix-se` uses exactly
these. `[ev: code obix-se module.xml; code alarm-se BPrinterRecipient.java vs obix-se Shell.java]`

### `plat moduleinstall` mechanics (JACE install)

`plat moduleinstall` stops **all** stations, overwrites the jar in place — no atomic rename,
no backup, no rollback. If `NIAGARA_HOME` is read-only it falls back to
`NIAGARA_USER_HOME/modules`. **Back up `modules/` before any production update.**
`[ev: corpus B633]`

### Adding a `-se` sibling to an existing module

Three places to touch (beyond the new `-se` source tree):

1. `<Mod>-se.gradle.kts` — `runtimeProfile.set("se")`
2. `niagara-module.xml` (module root) — add `se` to the `runtimeProfiles` list
3. nothing in `module-include.xml`: it holds only `<types>`, and the plugin writes `<moduleParts>` into the
   generated `module.xml` from `niagara-module.xml` (devguide `upgradingBuild.txt:95,115`)

`[ev: corpus B784]` `[ev: corpus B1313 §1313.3 C2]`

### JACE resource limits

JACE hardware (1 GB DDR3, ARM, QNX) never loads `-se`, so code a JACE needs stays in `-rt` and within
compact3: no OrientDB, Swing UI components or heavy JNI libraries there. `[ev: corpus B473]`

### Real `-se` modules for reference (Tridium 4.14)

| Module | Notable Java SE usage |
|--------|-----------------------|
| `alarm-se` | Printer recipients — `java.awt.print`, `javax.print` |
| `obix-se` | Swing shell (`Shell.java`) — display-optional, JACE-unsafe |
| `orientSystemDb-se` | Embedded OrientDB — heavyweight JNI, Supervisor-only |
| `test-se` | SE-profile test fixtures |

Theme modules and lexicon-only / `-doc` profiles are a separate resource-only module class;
see `types/theme.md` for theme modules and the `-doc help profile authoring recipe` section
above for doc-profile modules.

## Deployed version publication `[ev: retro dashboard-deployment-profiles Δ6]`
"Which build runs on this station" must be observable without opening Workbench or the Software Manager:
- Each module exposes a read-only version on its service component: `defaultModuleVersion` plus the build
  time (millis), as a `SUMMARY|READONLY` slot (the Reflow pattern, corpus B153). It is never OPERATOR-writable and
  is re-derived from the module itself at every start.
- A module with a `-ux` servlet also serves it as `GET /api/version` and the SPA shows it in an About/footer
  (`dashboard.md` § Deploy on a JACE).
- An off-station collector (e.g. an oBIX poller feeding a database) reads the slot per site and flags drift
  against the latest packaged release (the package's `SOURCE.txt`). `[ev: corpus B153 §153]`

## Recommendations for our modules (impact ÷ cost) `[ev: corpus B817]`
R1 chihuahua — populate rt+ux `module.lexicon` (10 types unlocalized; cheap, operator-visible). R2 DashboardPan-wb
— delete the empty skeleton OR fill it + add its JUnit dep. R3 DashboardPan — test `DashboardReader` (14-baja, the
sole data engine, untested) via a `BTestNg` station test (B815 recipe). R4 ColdRoomPan/CompPan — isolate the pure
model in a `model/` sub-package + add its junit moduleTest dep (L11). R5 ColdRoomPan — add the missing
`BFanMode`/`fanRunMode` lexicon keys. `[ev: corpus B817]`
