# Type: Workbench widget / PX view (wb) — SEED (feed as built)

Workbench-side UI: Swing `BWbView`/editor tools, or PX views/widgets. Exemplar to READ: chihuahua-wb (`BatchLinkEditor`, a proper WB view). Corpus has the PX subsystem deeply (`corpus-nav connections 197`, blocks B179–B213) + module-anatomy B636.

Not yet fully documented — seed pointers, feed via the retro step when you build one:

- **`-wb` profile** only; register the view type + an `@AgentOn` (agent-on the component type) so it opens from the tree. `requiredPermissions` on the agent = view VISIBILITY only, never security.
- **`-wb` gradle MUST include `com.tridium.convention.niagara-home-repositories` plugin:** a `-wb` subproject cannot resolve `:baja`/`:web` unless `id("com.tridium.convention.niagara-home-repositories")` is declared in its `<mod>-wb.gradle.kts`. The `-rt` profile gets this transitively from `niagara-home-repositories`; the `-wb` profile does NOT inherit it automatically. Missing this plugin causes "Could not resolve `:baja`" at Gradle sync time. `[ev: retro apillm-headless-servlet-rt-4.14-deltas Δ3]`
- **wb-agent registration requires BOTH `<agent>` in `module-include.xml` AND `@AgentOn` in `@NiagaraType`:** registering `@AgentOn` inline in `@NiagaraType` alone does NOT emit the `<agent>` block into `module-include.xml` — Workbench will not discover the agent. Both are required: the `@AgentOn` annotation on the class AND an explicit `<agent>` block in `module-include.xml`. Verify with `grep -r '<agent>' <mod>-wb/` after adding an `@AgentOn`. `[ev: retro apillm-headless-servlet-rt-4.14-deltas Δ5]`
- **PX views:** authored in Workbench PxEditor (GUI), shipped as module resources + registered as an agent view on a type; the same `.px` renders in the browser via the Hx/HTML5 profile. Editable fields via kitPx field editors; background image as a module resource. (Corpus B180/B194/B202/B203.)
- **PX vs custom servlet dashboard:** PX = declarative, Workbench-authored, native. A `BWebServlet` + SPA (types/dashboard.md) = custom HTML5, no PxEditor. Pick per need; an HMI that just points a browser at a URL takes either.
- Same build (Java 8 + slotomatic) and METHODOLOGY rules.

## UX rule — user-facing module UI must be intuitive and easy (standing user directive) `[ev: retro apillm-headless-servlet-rt-4.14-deltas Δ7]`

**RULE (standing user directive):** ALL user-facing module UI — every `-wb` config view AND every `-ux` dashboard — MUST offer pickers/choosers (ORD/point/folder) and drag-drop from the Nav tree for adding referenced components, PLUS a status table showing the resolved name, current value, and status for each managed item. NEVER ship a blank-entry "Add" that forces raw property-sheet editing on the operator.

- `-wb` manager rows that point at station components: Add an in-dialog "pick from station" chooser for Add and Edit flows.
- The `BAbstractManager`+`BComponentChooser` combination (see §Manager recipe) is the proven implementation — avoid hand-coded blank-Add patterns.
- **Lint candidate `lint-wb-usability`:** a `BWbComponentView` subclass that holds user-managed child refs (slot type `BOrd` or similar) but ships no `BComponentChooser` invocation in its controller is a candidate for this check.

`[ev: user directive 2026-09-20]`

## How much wb is enough — the ladder (climb ONLY when needed)
- **Author the LEAST wb the ladder allows:** rung 0 = **nothing** — the default property-sheet/wire-sheet views already render standard slots (kitControl ships 152 rt types with only 2 wb field editors); rung 1 = a `BWbFieldEditor @AgentOn(<value type>)` for ONE composite value that renders badly; rung 2 = a `BAbstractManager`/`BDeviceManager` ONLY when the component is a container of learned/discovered children; rung 3 = a `BWbComponentView` ONLY for non-tabular interaction. **Our ColdRoomPan/CompPan components sit at rung 0 — do not build a Manager.** [ev: retro corpus-index · B751]
- **Rung-2 has two authoring recipes — pick by context:** (a) **Tridium subclass recipe** — extend `BAbstractManager`/`BDeviceManager`, override `makeModel`/`makeController`/`makeLearn` (`BDriverManager.java:33-85`); use for a standalone driver that owns its own Manager container. (b) **Honeywell device-model PLUGIN recipe** — implement only `BIHonDeviceModel` (or `BIHonBacnetDeviceModel`); the plugin contributes columns, supported models, and commands (`BThermostatDeviceModel.java:22-53`); the shared `HonDeviceModel` framework (`HonDeviceModel.java:23`) discovers plugins from the registry — **zero Manager subclass code**. Prefer (b) for a device-family module that slots into an existing Manager framework; prefer (a) only when no shared framework exists. [ev: corpus B751 §751.3]
- **FieldEditor recipe (rung 1):** ctor builds the widgets → `linkTo(widget, textModified, setModified)` → override `doLoadValue`/`doSaveValue`/`doSetReadonly`; compose child editors via `BWbFieldEditor.makeFor(value)`; register with `@AgentOn(<that value type>)`. [ev: retro corpus-index · B751]
- **A Honeywell "Wizard" is usually a tabbed `BWbComponentView`, not a `BWizard`:** step-panes = tabs, backed by rt `BJob`s launched from an agent `BMenu`. Always mutate through the space (`newTransaction` / `tx.commit`); undo is inherited from the space, never hand-rolled. [ev: retro corpus-index · B751]
- **Rung 4+ (reference ceiling — do NOT imitate):** a "vendor programming environment" tier exists above rung 3 in the Honeywell `ace` module: own wire sheet + app wizard + catalog palette + opcode expression editor. This is the ceiling of the rung ladder, documented here as a concrete upper bound for scope and code-review discussions. Our modules stay at rung 0–1. Reaching rung 4 implies owning a custom Baja domain language runtime — not a practical target for our use cases. `[ev: retro wb-vendor-ux-wave3-vendor-drivers-deltas Δ5]` `[ev: corpus B1098]`

## wb/model testable seam — exemplar-backed

- **DWB1 — `-wb` is off-station testable via a `model/` lambda-injection seam (HIGH):** keep the business logic in a Baja-free `model/` package; inject the slot-availability check as a `Predicate<String>` so the model has zero Baja imports. The `BWidget` view stays the thin adapter. `chihuahua-wb`'s `LinkSlotNameUtil` + 33 pure `@Test` cases are the proven pattern. **This upgrades `-wb` from seed to exemplar-backed for the model layer.** [ev: corpus B762]
- **Dual-surface `@AgentOn` registration:** write `@NiagaraType(agent={@AgentOn(types={"mod:Type"}, requiredPermissions="r")})` on the view/FE; Slot-o-Matic emits `<type><agent><on type=…/></agent></type>`; multi-type `types={…}` = one view over several source types. `[ev: corpus B780]`

## bajaux data-channel dialects

> Applies only when you choose the **bajaux `@AgentOn` + `BIJavaScript`** recipe. Our servlet-SPA uses REST-poll and needs neither dialect — this section guides builders who pick the bajaux path. [ev: corpus B752 §752.2]

Two dialects for delivering data to a bajaux `@AgentOn` view:

- **(a) `fal.serverSideCall` dialect (`BSingleton` channel):** implement `BIServerSideCallHandler` on a `@NiagaraSingleton`; annotate the view with `@AgentOn(requiredPermissions="ri")`. The server returns `BValue`/JSON; live refresh via Fox `subscriberMixIn` in the JS view. Exemplar: `BFALServerSideCallHandler.java:29-176` (EagleHawk). Use when the view needs reactive/push-style updates from a singleton service.
- **(b) `baja.rpc` dialect (`@NiagaraRpc` channel):** annotate static methods on a `BComponent` with `@NiagaraRpc(permissions="…", transports={web,box})`; the browser calls `baja.rpc({typeSpec, method, args})`. Exemplar: `BThermostatWizardRPC.java:168-176` (TC/Sylk React SPA). Use when the view calls discrete wizard-style operations.

**Anti-pattern — `permissions="unrestricted"` on dialect (b):** Honeywell TC/Sylk RPCs ship `permissions="unrestricted"` — no server-side auth check. Do NOT copy this; the browser is never the security boundary. Add a server-side `OPERATOR_WRITE` (or stricter) check on every mutating RPC, matching the pattern in `dashboard.md §RBAC`. [ev: corpus B752 §752.2]

## Good -wb artifact doctrine (DWB1 exemplar — 10 rules)

> Every line applies to any -wb profile with Java sources. [ev: corpus B809] [ev: corpus B817]

1. **Profile isolation**: keep all -wb code in the `-wb` Gradle profile; never import `-wb` classes from `-rt` or `-ux`. [ev: corpus B809] [ev: corpus B817]
2. **Model/view split (DWB1)**: isolate business logic in a Baja-free `model/` package; the view injects Baja access as a `Predicate<String>` or similar lambda — zero Baja imports in `model/`. [ev: corpus B809] [ev: corpus B817]
3. **Thin view adapter**: the `BWbComponentView`/`BWbFieldEditor` subclass does layout + delegation only; it never contains business logic that belongs in `model/`. [ev: corpus B809] [ev: corpus B817]
4. **Off-load UI-thread traversal**: `doInvoke` bodies must NOT call `getNavChildren`/`getNavNodes`/`BqlQuery` directly on the Swing EDT; use `invokeLater` or `BJobService` for any nav-tree walk. [ev: corpus B809] [ev: corpus B817]
5. **Justify broad `@AgentOn`**: `@AgentOn(types="baja:Component")` (or any super-type) requires a comment explaining WHY the agent must attach to every component of that type. [ev: corpus B809] [ev: corpus B817]
6. **`requiredPermissions` = view only**: never use the agent's `requiredPermissions` as a security gate; it controls VISIBILITY, not authorisation — authorisation belongs in the action/servlet. [ev: corpus B809] [ev: corpus B817]
7. **Pure `@Test` in `srcTest`**: the `model/` package must be testable without a running station; aim for ≥20 pure unit tests covering the predicate-injection seam. [ev: corpus B809] [ev: corpus B817]
8. **No `getNavChildren` in `doInvoke`**: if a search or tree-walk is needed, extract it into a named method called via `invokeLater`; the `doInvoke` body stays a one-liner. [ev: corpus B809] [ev: corpus B817]
9. **Non-empty scaffold gate**: ship a `-wb` jar only when it has ≥1 `.class` OR ≥1 palette `<p n=` entry; an all-empty scaffold (`verify-module.sh` `wb-scaffold` WARN) means nothing was compiled or registered. [ev: corpus B809] [ev: corpus B817]
10. **Declare every transitive dep**: every `<dependency>` in `META-INF/module.xml` MUST appear as `api(":X")` or `nre(":X")` in the profile `.gradle.kts`; phantom deps (`verify-module.sh` `phantom-dep` WARN) disappear silently after Gradle updates. [ev: corpus B809] [ev: corpus B817]
11. **Fail-closed authorization for WB controls:** a control whose authorization mixin or PIN is absent must be DISABLED **and** HIDDEN by default — never open-by-default. Default authorization slots/PINs to `-1` (no assignment = no access). On `loadValue()`, check the pin; if the user lacks the required permission call `widget.setEnabled(false)` **and** `widget.setVisible(false)`. A missing mixin ≠ "unprotected" — it equals "no access". `[ev: retro honeywell-wb-rt-wb-deltas Δ3]` `[ev: corpus B1079]`

**DWB1 exemplar — chihuahua-wb `model/` tree** (commit `175eee8`, `angeles725/chihuahua`):
```
chihuahua-wb/src/com/angeles/chihuahua/wb/
  model/
    DirectionButtonUtil.java   — Baja-free direction label logic
    DirectionLabelUtil.java    — Baja-free label text util
    LinkSlotNameUtil.java      — slot name parsing, 33 pure @Test cases
    PendingLink.java           — Baja-free pending-link value object
    PendingLinkBuilder.java    — builder for PendingLink
    SearchResultUtil.java      — Baja-free search-result formatter
  BBatchLinkEditor.java        — thin BWbComponentView adapter
```
The `model/` package has zero Baja imports; all station access is injected via `Predicate<String>` at construction time. 33 `@Test` cases run without a station. [ev: corpus B809] [ev: corpus B817]

## Manager recipe — `BAbstractManager` wiring reference [ev: retro wb-manager-framework-deltas Δ1]

> Rung-2 detail (extends the ladder above): reach here only when the component is a container of discovered/learned children. Our ColdRoomPan/CompPan panels sit at rung 0 — do not build a Manager unless the ladder says so.

### 6-object anatomy and boolean gates

Every `BAbstractManager` subclass is wired by six support objects plus three boolean gates on the container class:

| Object | Role |
|--------|------|
| `BAbstractManager` | The Swing panel — top of the stack; owns `BTable` + toolbar; calls `makeModel`/`makeController`/`makeLearn` |
| `MgrModel` | Table model — owns `cols[]` (the database/display columns) |
| `MgrController` | Actions — New, Delete, Edit, Discover; owns `promptForNew()` / `promptForEdit()` |
| `MgrLearn` | Discover/learn contract — returns its own `cols[]` **independent** of `MgrModel.cols[]` (separate column sets); called from the Discover toolbar button |
| `MgrColumn` | Column descriptor — see taxonomy below |
| `BMgrEditDialog` | Cell-edit bridge — `ColumnInput` inner class routes cell edits to the slot pathway (`BAbstractManagerFE` is NOT a class; "FE" is a driver-naming convention only) |

Three boolean gates on the `BAbstractManager` subclass drive pane visibility:

| Gate method | Controls |
|-------------|---------|
| `isLearnable()` | Shows/hides the Discover button |
| `isTaggable()` | Shows/hides the Tags pane |
| `isTemplatable()` | Shows/hides the Template pane |

`[ev: corpus B1088]`

### `MgrColumn` taxonomy reference card [ev: retro wb-manager-framework-deltas Δ2]

| Column type | When to use | Key attribute |
|-------------|-------------|---------------|
| `MgrColumn.Name` | Display name of the child component | — |
| `MgrColumn.Type` | Tridium type spec of the child (e.g. `"mod:MyPoint"`) | `typeSpec` |
| `MgrColumn.Prop` | A **flat** slot on the child component (accessible directly via `slot.asProperty()`) | `prop` name |
| `MgrColumn.PropPath` | A **nested** slot via a dot-path (e.g. `"proxyExt.precision"`); traverses the child component tree | `propPath` dot string |

Use `Prop` for direct slots; use `PropPath` for mixin or extension slots. The discover table (`MgrLearn.cols[]`) and database table (`MgrModel.cols[]`) carry **independent** column arrays — do not share or reuse the same `MgrColumn` instances across both.

`[ev: corpus B1089]`

**One `MgrColumn.Prop` per key `@NiagaraProperty` on the proxy ext (PD-02):** each key property on a `BProxyExt` subclass (e.g. `deviceAddress`, `pointOffset`, `scaling`) maps to exactly ONE `MgrColumn.Prop`; the column reads/writes that single slot — no compound columns or manual ORD construction. This ensures the Add/Edit dialog pre-fills the slot value without extra coding. `[ev: retro wb-vendor-ux-rt-wb-pattern-deltas Δ2]`

**`addDefault*Columns()` + `addCustom*Columns()` template hook (PD-16):** structure a manager's column definition as two protected methods: `addDefaultColumns(List<MgrColumn>)` (supplies the built-in columns every subclass gets) and `addCustomColumns(List<MgrColumn>)` (empty by default; subclasses override to extend). Call both from `makeModel()`. This lets a subclass add columns without replacing the default set. `[ev: retro wb-vendor-ux-rt-wb-pattern-deltas Δ16]`

**`isDefault` blank-suppression for polymorphic manager columns (PD-17):** when a manager row type can be one of several subtypes (e.g. digital vs analog vs enum proxy), some columns are irrelevant for certain row types. Override `MgrColumn.isDefault(BComponent row)` to return `false` for rows where the column does not apply — the manager hides the cell rather than showing a blank or a cast exception. `[ev: retro wb-vendor-ux-rt-wb-pattern-deltas Δ17]`

**Boolean-flag column-switch template (PD-22):** when a manager must show entirely different column sets based on a runtime flag (e.g. client mode vs slave mode), hold two `MgrColumn[]` arrays as private static finals and dispatch in `makeColumns()` via an `isX()` hook:

```java
private static final MgrColumn[] CLIENT_COLUMNS = { /* … */ };
private static final MgrColumn[] SLAVE_COLUMNS  = { /* … */ };

@Override
protected MgrColumn[] makeColumns() {
    return isClientMode() ? CLIENT_COLUMNS : SLAVE_COLUMNS;
}
protected boolean isClientMode() { return false; } // subclass overrides
```

Never collapse the two sets into one wide array with conditional visibility — swapping whole arrays is cheaper and cleaner. `[ev: retro wb-vendor-ux-rt-wb-pattern-deltas Δ22]`

### `BAbstractManager` vs `BWbComponentView` — decision table [ev: retro wb-manager-framework-deltas Δ3]

| Criterion | Use `BAbstractManager` | Use `BWbComponentView` + `BTable` |
|-----------|------------------------|-----------------------------------|
| Children discovered/learned from the network | YES | No |
| Need Discover/New/Delete toolbar | YES | No |
| Need paging, column sort, inline cell-edit via `BMgrEditDialog` | YES | Optional |
| Non-tabular interaction (forms, wizards, status dashboards) | No | YES |
| Simple container with hand-curated children (our Apillm importers/exporters) | No | YES (hand-built BTable suffices) |
| Standalone driver that owns its Manager container | YES (Tridium subclass recipe) | No |

Our Apillm importer/exporter managers use the hand-built `BWbComponentView`+`BTable` recipe: no discovery, no learn, hand-curated children — the full `BAbstractManager` framework adds no value there.

**`BWbComponentView` for non-table specialized UIs — terminals, schedule editors (PD-08):** use `BWbComponentView` (not a `BAbstractManager`) for any UI that is not a table of discovered/learned rows — VT100 terminals, job schedule editors, wizard flows, or any non-tabular interaction. These views live at the same rung-3 as a manager but serve fundamentally different UX patterns. `[ev: retro wb-vendor-ux-rt-wb-pattern-deltas Δ8]` `[ev: corpus B1058]`

`[ev: corpus B1091]`

### Plugin-extensible device-type manager via SPI (Honeywell recipe) [ev: retro honeywell-wb-rt-wb-deltas Δ1]

When multiple device families share one manager container, use the **plugin SPI** instead of subclassing `BAbstractManager` for each family. Implement only `BIHonDeviceModel` (or `BIHonBacnetDeviceModel`); the plugin contributes columns, supported model specs, and commands. The shared framework discovers plugins at runtime via `NiagaraRegistryUtil.getImplementersOfTypeSpec(BIHonDeviceModel.TYPE)` — zero Manager subclass code per device-family module.

Key responsibilities of a `BIHonDeviceModel` implementer:
- `getSupportedModelSpecs()` — returns the `BTypeSpec[]` this plugin handles.
- `createColumns()` — returns the `MgrColumn[]` the plugin contributes to the shared table.
- `createCommands()` — returns any toolbar commands scoped to devices this plugin owns.

Register each plugin via a plain `<type>` in `module.xml`; no `@AgentOn` is required. Prefer this recipe over a Manager subclass when a shared framework already exists and you are adding a new device family on top of it; prefer the Tridium subclass recipe (rung-2(a) above) only when no shared framework exists. `[ev: corpus B1077]`

### Static State for manager view continuity [ev: retro honeywell-wb-rt-wb-deltas Δ8]

When a manager must survive a close/reopen without losing subscription or discovery state (e.g. a scan still in progress), hold that domain state in a `static` inner class scoped to the manager class:

```java
public class BMyDeviceManager extends BAbstractManager {

    // Survives manager close/reopen: the static class lives with the classloader,
    // not with the Swing panel instance.
    private static class State {
        volatile boolean discoveryInProgress;
        final List<DiscoveredDevice> found = Collections.synchronizedList(new ArrayList<>());
    }
    private static final State STATE = new State();

    @Override
    protected void doLoadValue(BObject value, Context cx) {
        // read STATE.found — may already be populated by a background scan
    }
}
```

**Anti-pattern:** do NOT make Swing *widgets* (e.g. a `JList` or `BTable` model) static — only DOMAIN STATE. Widget singletons across two simultaneously open manager windows cause threading violations. The static field must hold only value-typed or thread-safe domain state. `[ev: corpus B1078]`

### Dialog pre-population — live RT objects and toRow() [ev: retro wb-vendor-ux-rt-wb-pattern-deltas]

**`toRow()` result carries ALL config slots so the add-dialog is pre-populated (PD-03):** the discover/learn path's `toRow()` result object MUST carry ALL config slots (address, name, type, scaling, ranges) — not just the discovery key. The `MgrController.promptForNew()` dialog reads these slots to pre-fill the Add dialog; a sparse `toRow()` forces the operator to fill fields manually that the driver already knows. `[ev: retro wb-vendor-ux-rt-wb-pattern-deltas Δ3]` `[ev: corpus B1055]`

**Dialogs accept live RT objects, not a re-fetch (PD-07):** when the Edit or Add dialog is pre-populated from an already-resolved RT component, pass the live object directly to the dialog rather than re-resolving from an ORD. Re-fetching inside the dialog introduces a race (the state may change between the manager render and the dialog open) and adds a latency flash. Accept the live RT component in the dialog constructor. `[ev: retro wb-vendor-ux-rt-wb-pattern-deltas Δ7]` `[ev: corpus B1055]`

**COV/capability-bit `toRow()` pre-population (PD-21):** when a protocol device advertises capabilities as a bitfield (e.g. `servicesSupported` in BACnet, capability flags in Modbus), decode the bitfield inside `toRow()` and set the corresponding pre-filled fields on the result object. The manager row then shows pre-checked capability boxes at discover time without a second round-trip. Bit decode belongs in `toRow()`, not in the Add dialog. `[ev: retro wb-vendor-ux-rt-wb-pattern-deltas Δ21]` `[ev: corpus B1073]`

### Minimal non-driver custom-manager template [ev: retro wb-manager-framework-deltas Δ4]

Use this template when a **service or container** (not a driver) needs a manager view — e.g. a service that owns a set of configuration records with an Add/Delete UI. For the driver-centric recipe with full `makeLearn()` + network discovery, see `envCtrlDriver` (B956).

Key constraints for a non-driver custom manager:
- `@AgentOn` the **container/service** type, NOT a `BNetwork`/`BDevice`.
- Use `MgrColumn.Prop` only (no `PropPath`, no Discover) — keep it flat.
- Do NOT override `makeLearn()`; return `null` and `isLearnable() = false`.
- Transaction: always mutate via `MgrController.promptForNew()` → `MgrEdit.commit()` → `Mark.moveTo(container)` — never call `BComponent.add()` directly.

```java
// -wb profile only
@NiagaraType(agent = @AgentOn(types = {"myMod:MyService"}, requiredPermissions = "r"))
public class BMyServiceManager extends BAbstractManager {
    @Override protected MgrModel      makeModel()      { return new MyModel(); }
    @Override protected MgrController makeController() { return new MgrController(this); }
    @Override protected MgrLearn      makeLearn()      { return null; }   // no discovery

    @Override public boolean isLearnable()   { return false; }
    @Override public boolean isTaggable()    { return false; }
    @Override public boolean isTemplatable() { return false; }

    private static class MyModel extends MgrModel {
        MyModel() {
            cols = new MgrColumn[] {
                new MgrColumn.Name(),                          // display name column
                new MgrColumn.Prop("enabled", "Enabled"),     // flat boolean slot
                new MgrColumn.Prop("priority", "Priority"),   // flat int slot
            };
            newTypes = new Type[] { BMyRecord.TYPE };         // allowed New types
        }
    }
}
```

`[ev: corpus B1091]`

## Vendor-grade -wb UX patterns (cross-vendor survey PD-01..PD-10)

> These rules come from the wb-vendor-ux cross-vendor survey (B1054–B1061). Apply to any driver or service that ships a `-wb` profile.

- **Every `BBasicNetwork`/driver module ships a `@AgentOn` WB device manager (WB-presence rule, PD-01):** a driver that adds a `BBasicNetwork` subclass MUST register at least a minimal `BAbstractManager` or `BWbComponentView` via `@AgentOn` on the network or device type. Relying solely on the default property-sheet view forces operators into raw slot editing — the vendor-grade bar is a table view at minimum. For the full manager recipe see `§Manager recipe`. `[ev: retro wb-vendor-ux-rt-wb-pattern-deltas Δ1]` `[ev: corpus B1057]`

- **Every long RT operation is a `BOrd` action on a job-bar (PD-04):** any operation whose RT side takes >1 s (firmware upload, device scan, config push) MUST follow the `submit → sync → resolve → registerForEvents → jobBar.load` recipe. The `BOrd` points at the `BJob`; `jobBar.load(ord)` attaches the progress UI; the button is disabled while `jobBar.isRunning()`. Never block the Swing EDT or poll in a `javax.swing.Timer`. See `types/logic-authoring.md §BSimpleJob + Fox file-channel streaming` for the RT job authoring side. `[ev: retro wb-vendor-ux-rt-wb-pattern-deltas Δ4]` `[ev: corpus B1060]`

- **Prune inapplicable inherited commands in `makeCommands()`/`getAgents()` (PD-05):** a WB view inherits all parent-type commands and agents by default. Override `getAgents()` to filter out agents that do not apply to the current type (e.g. the default property-sheet agent when a custom manager is the preferred view). Override `makeCommands()` to remove toolbar commands that would cause errors or have no effect on the current component. Remove, do not just disable, to keep the UX clean. `[ev: retro wb-vendor-ux-rt-wb-pattern-deltas Δ5]` `[ev: corpus B1057]`

- **Every `BOrd` ref slot needs a station-scoped WB picker — never the file-space default (PD-06):** any `-wb` form or manager that lets the operator configure a `BOrd` pointing at a station component MUST present a station-component chooser (`BComponentChooser` or the `targetType` facet; see `§Field editors — PD-FE1`). The default `BOrd.NULL` dialog opens a file-space chooser (`C:\`) — useless in a station context. This extends the `lint-wb-file-chooser` candidate from `§-wb station-space picker`. `[ev: retro wb-vendor-ux-rt-wb-pattern-deltas Δ6]` `[ev: corpus B1061]`

- **Cross-cutting WB concerns use a `BWbService`, not per-component agents (PD-10):** when a `-wb` concern spans multiple component types in a session (e.g. a shared job scheduler, a session-keyed credential cache, or a global drag-and-drop coordinator), implement it as a `BWbService extends BAbstractService` registered under `/Services`. Per-component agents that share state across component instances via `static` fields is the anti-pattern — it creates invisible coupling and GC pressure. `[ev: retro wb-vendor-ux-rt-wb-pattern-deltas Δ10]` `[ev: corpus B1060]`

## -wb station-space picker, table repaint, and view refresh rules

- **Station-component pickers in a `-wb` manager MUST scope to the STATION COMPONENT SPACE (`slot:`), NOT the file system:** `BWbFieldEditor.dialog(this, title, BOrd.NULL)` defaults to the `file:` space and opens a File Chooser browsing `C:/` (or the WSL root) — NOT the station component tree. To browse the station, use `BComponentChooser` filtered to `BControlPoint` (or the appropriate base type). The vendor-driver `PointManager` pattern (corpus B1055 Andover) is the reference. For a `BOrd` slot, the zero-code fix is the `targetType` facet (see §Field editors PD-FE1). **Lint candidate `lint-wb-file-chooser`:** flag a `-wb` class that calls `BWbFieldEditor.dialog(this, …, BOrd.NULL)` with no `targetType` or `BComponentChooser` nearby. `[ev: retro apillm-headless-servlet-rt-4.14-deltas Δ8]` `[ev: corpus B1055]`
- **`-wb` table repaint gotcha — `table.relayout()` after in-place model mutation:** after mutating a `BTable`'s model in place (e.g. clearing and repopulating its rows), calling `table.setModel(sameModelInstance)` is a NO-OP in some BJ builds — the view shows stale data. Call `table.relayout()` (or rebuild the model instance then call `relayout()`) to force a full repaint. Surfaced as "Toggle Recursive shows old value" — the HTTP serving was correct; only the WB view was stale. `[ev: retro apillm-headless-servlet-rt-4.14-deltas Δ15]`
- **A `-wb` manager/view showing live component state MUST refresh on activation AND subscribe to its component:** a view that only rebuilds its model in `doLoadValue()` / on-command shows STALE state after a station restart or an external change until the user re-opens the view. Add: (a) call the model-rebuild in `doLoadValue()` AND in a topic/`Subscriber.event()` callback that fires on component changes; (b) call the reload on `componentShown()`/`doActivated()` (re-entry point when a WB tab is switched back); (c) register the subscriber in `doLoadValue()` and unregister in `doUnloadValue()`. Pairs with B1070 `DynamicTableModel` and corpus B29 `serviceStarted` registration. **Lint candidate `lint-wb-refresh`:** a `BWbComponentView` subclass with a `doLoadValue()` that sets a model but no `Subscriber` or topic-listener in the same class. `[ev: retro apillm-headless-servlet-rt-4.14-deltas Δ17]` `[ev: corpus B29]`

## WB user preferences — per-user JSON under userHome/ [ev: retro honeywell-wb-rt-wb-deltas Δ2]

For remembered WB state (visible columns, selected commands, last-used values) that is per-user and must survive a manager close/reopen, store a small JSON file under `userHome/<username>/<module>/prefs.json`. No rt slots are needed — the WB view reads/writes the file directly on load/save, keeping the slot schema clean.

```java
// Persist per-user WB prefs — manager open → read; manager close → write
File home  = new File(Sys.getUserHome(), getUsername());   // Sys.getUserHome() → userHome/
File prefs = new File(home, "myModule/prefs.json");
// write: new ObjectMapper().writeValue(prefs, myPrefsObject);
// read:  myPrefsObject = new ObjectMapper().readValue(prefs, MyPrefs.class);
```

Guard every read with `try/catch` — the file may not exist on the first open. This pattern avoids adding transient or per-user state to the rt component model. `[ev: corpus B1077]`

## Integer `visibilityPin`/`actionPin` authorization for PX widgets and manager rows [ev: retro honeywell-wb-rt-wb-deltas Δ4]

Add a `visibilityPin` (int, default `-1`) and an `actionPin` (int, default `-1`) as `@NiagaraProperty` slots on a PX widget or manager-row type. At render time:
- If `visibilityPin` check fails → `widget.setVisible(false)`.
- If `actionPin` check fails → `widget.setEnabled(false)`.
- Default of `-1` means no pin assigned → control is closed (fail-closed rule above).

```java
// Example pin check (integer maps to a BPermissions mask or role ordinal)
boolean hasVis = session.getPermissions().has(visibilityPin);
boolean hasAct = session.getPermissions().has(actionPin);
widget.setVisible(hasVis);
widget.setEnabled(hasVis && hasAct);
```

Reusable across manager rows and PX widgets without per-type auth boilerplate. `[ev: corpus B1079]`

## Field editors — station-component pickers, the null-ord gotcha, and point-creation from WB

> Rung-1 detail (extends the FieldEditor recipe above): most modules need NO custom field editor (rung 0). Reach here only when an rt slot must pick a station component, or an importer-style Manager must create points from WB. [ev: retro wb-field-editors-deltas]

- **Facet-driven field-editor selection — pick the FE from the slot's facets (PD-15):** instead of hard-coding a field editor class for a slot, store a facet key (e.g. `BFacets.make("fieldEditor", BString.make("module:TimeZoneSelectionFE"))`) on the property; the WB framework selects the matching registered FE by that facet. This allows operators to swap the FE per slot without code changes, and avoids a WB-profile compile dependency just to name a field-editor class. `[ev: retro wb-vendor-ux-rt-wb-pattern-deltas Δ15]` `[ev: corpus B1067]`

- **`targetType` facet — zero-code station-component picker for a `BOrd` slot (PD-FE1):** a `BOrd` slot whose value is null has no scheme, so `BOrdFE` stays on the file-system chooser (`BFileOrdChooser`) — the null-ord gotcha, by design (a null value has no `BIOrdChooser` scheme override to fire). To browse the station component tree instead, put a `targetType` facet on the property (`facets=BFacets.make("targetType", BString.make("baja:Component"))`, or the equivalent `@Facets`); `BOrdFE` then selects `BComponentChooser` automatically, even for a null ord. No `-wb` code required — prefer this over a custom FE for `BOrd`→component slots. [ev: retro wb-field-editors-deltas Δ1] `[ev: retro apillm-headless-servlet-rt-4.14-deltas Δ20]`
  - **Δ19 SUPERSEDED:** a previous approach proposed a custom registered `BWbFieldEditor` for station-ORD slots. That approach (Δ19) is superseded by the simpler `targetType` facet (PD-FE1 above). See `[ev: retro apillm-headless-servlet-rt-4.14-deltas Δ19]` for the original proposal; the facet-based fix requires zero `-wb` code. `[ev: retro apillm-headless-servlet-rt-4.14-deltas Δ19]`
- **`targetType=baja:ControlPoint` for pick/create a control point slot (PD-FE2):** use `targetType=baja:ControlPoint` (or a relevant subtype such as `baja:NumericWritable`) in the `@BFacets` annotation on a `BOrd` property that should refer to a control point — the chooser is then filtered to points only, not every component in the station. [ev: retro wb-field-editors-deltas Δ2] `[ev: retro apillm-headless-servlet-rt-4.14-deltas Δ21]`
- **Point-creation from WB — use the Manager plumbing, not `BComponent.add()` (PD-FE2-create):** for an importer-style `BAbstractManager`, override `getNewTypes()` to return the writable types you allow, then rely on `MgrController.promptForNew()` → `MgrEdit.commit()` → `Mark.moveTo(container)`. Do NOT call `BComponent.add()` manually from the WB view — the manager's transaction plumbing owns the station relay and undo; a hand-rolled `add` bypasses both. [ev: retro wb-field-editors-deltas Δ2]
- **Filtered sub-type chooser — subclass `BComponentChooser`, NEVER override `baja:Ord` (PD-FE3):** when a slot must pick only a subtype (e.g. only `BNumericWritable`), subclass `BComponentChooser` with a `selectFilter` `RefFilter` — `(parent, slot) -> slot.isProperty() && parent.get(slot.asProperty()).getType().is(BTargetType.TYPE)` — and register it via `@AgentOn(types={"myModule:MyOrdAlias"})` on a **custom** ord type. Never override the global `baja:Ord` agent — that would hijack every ord slot in the station. The `FIELD_EDITOR` facet is the alternative when you cannot change the slot type; for `BOrd`→component prefer the `targetType` facet (PD-FE1). For a station-subtree-filtered chooser, subclass `BComponentChooser` and call `BComponentChooser.prompt(root, path, displayFilter, selectFilter)` directly, or add new target types via `parent.add(name, point, cx)` for `[BNumericWritable/BBooleanWritable/BStringWritable/BEnumWritable]`. For `BEnumWritable`, set a minimal `BEnumRange` facet so it constructs without error. [ev: retro wb-field-editors-deltas Δ3] `[ev: retro apillm-headless-servlet-rt-4.14-deltas Δ22]`
- **One-step "create the referenced component" in a manager (Δ23):** when a manager's rows reference station components (e.g. an importer mapping → writable point), the "Add…" flow SHOULD let the user CREATE the target component in one dialog — type dropdown (Numeric/Boolean/String/Enum), name, folder → `new B*Writable()` + `folder.add(uniqueName, point, cx)` + wire the ref ORD — with a "(use existing…)" fallback. NEVER force pre-creating points and hand-picking ORDs separately. For `BEnumWritable`, set a minimal `BEnumRange` facet on construction so it serializes. `[ev: retro apillm-headless-servlet-rt-4.14-deltas Δ23]` `[ev: corpus B1087]`

## Multi-part WB view — BTabbedPane+BLabelPane recipe (PD-19) `[ev: retro wb-vendor-ux-rt-wb-pattern-deltas Δ19]`

When a `-wb` view needs multiple sections (e.g. status tab + config tab + diagnostics tab), use Baja's native `BTabbedPane`+`BLabelPane` — NOT `javax.swing.JTabbedPane`. The native types participate in the Baja Swing lifecycle and fire `doLoadValue`/`doSaveValue` correctly per tab.

**Recipe (`BTabbedPane` + one `createTPage*()` per tab):**

```java
private BTabbedPane tabs;

@Override
protected void buildImpl(BWbTablePane parent) {
    tabs = new BTabbedPane();
    tabs.add(createTPageStatus(),     "Status");
    tabs.add(createTPageConfig(),     "Configuration");
    tabs.add(createTPageDiagnostics(),"Diagnostics");
    parent.add(tabs, BorderLayout.CENTER);
}

private BLabelPane createTPageStatus() {
    BLabelPane p = new BLabelPane();
    // … add widgets …
    return p;
}
```

**Rules:**
- One `createTPage*()` factory method per tab — keeps the constructor compact.
- **Conditional tab:** wrap in `if (childIsPresent())` before `tabs.add(...)` to hide a tab when the required child is absent.
- **`doLoad`/`doSave` symmetry:** override `doLoadValue()` to set widget values from the loaded component; override `doSaveValue()` to write widget values back. Keep them paired — a missing `doSaveValue()` causes silent loss of edits.
- **Status tab layout:** place a `BJobBar` in `NORTH` and a `BTable` in `CENTER` — the standard vendor layout for a status/command tab.

`[ev: corpus B1070]`

**`BTabbedPane.selection` is TRANSIENT — persist the tab by label text (PD-24):** `BTabbedPane.selection` is NOT annotated with `@NiagaraProperty`, so it is TRANSIENT — it does not survive a view close/reopen or a `doSaveValue()` call. To remember the selected tab across open/close, persist the tab's **label text** (not its index, which is unstable if tabs are added/removed) in a per-user pref file (see `§WB user preferences`) or in a `String` field. Restore in `doLoadValue()` with `tabs.setSelection(tabs.find(savedLabel))`. `[ev: retro wb-vendor-ux-rt-wb-pattern-deltas Δ24]` `[ev: corpus B1076]`

## PX authoring — binding taxonomy

PX files are XML authored in the Workbench PxEditor, shipped as module resources, and rendered in the browser via the Hx/HTML5 profile. The corpus contains 270+ `.px` files; the binding patterns below are the recurring skeleton. [ev: corpus B752 §752.3]

**Core binding types:**

| Intent | Binding type | Key attributes |
|--------|-------------|----------------|
| Read label with status tint | `BoundLabelBinding` + `ObjectToString` converter | `ord="…"`, `statusEffect="color"`, `format="%out.value%"` |
| Write-back setpoint / enum | `SetPointBinding` | `ord="…"`, `widgetEvent="actionPerformed"`, `widgetProperty="selected"` |
| Invoke an action | `ActionBinding` | `ord="…"`, `widgetEvent="actionPerformed"` |
| Hyperlink to a view | any binding | ORD ends in `\|view:<module>:<ViewName>`; `\|` = cross-space separator |

**Geometry:** all widget positions are absolute — `layout="x,y,w,h"` in px. Cross-space ords use `|` (pipe) as the space separator; e.g. `station:|slot:/Services/MyService`.

**Minimal read-label + write-setpoint snippet** (from `VENOM_VAV_003n.px:32-35`, `Smart_IO.px:134` — [ev: corpus B752 §752.3]):
```xml
<!-- Read label: displays current value, tints red on fault -->
<Label layout="10,10,100,20">
  <BoundLabelBinding ord="slot:/Setpoint" statusEffect="color">
    <ObjectToString format="%out.value% °C"/>
  </BoundLabelBinding>
</Label>
<!-- Write-back setpoint control -->
<SpinnerWidget layout="10,35,100,24">
  <SetPointBinding ord="slot:/Setpoint"
    widgetEvent="actionPerformed" widgetProperty="selected"/>
</SpinnerWidget>
```

See also: `types/dashboard.md §serving recipe` for the PX-vs-servlet decision and the PX complement use case.

See also: `docs/module-best-practices.md` (rt/ux/wb do & don't).

---

## WB view-target taxonomy — three axes [ev: retro wb-vendor-ux-wave3-vendor-drivers-deltas Δ12]

The wave-3 survey (B1094–B1105) identified a second axis that extends the rung ladder: **where** the WB view is anchored, not just how complex it is. Three named targets:

| Target axis | `@AgentOn` anchor | Lifecycle / contract | Typical `@NiagaraType` base |
|-------------|-------------------|----------------------|-----------------------------|
| **station-component / driver** | A `BComponent`, `BNetwork`, `BDevice`, or `BService` subclass in the station component space | Opened from the Workbench Nav tree; participates in the normal station lifecycle | `BWbComponentView`, `BAbstractManager`, `BWbFieldEditor` |
| **platform-service-plugin** | A `platform:*Service` agent (e.g. `PlatformServiceAgent`) resolved via `@AgentOn` | Opened from the **Platform tab**; accesses the Platform daemon via `poll`/`lease`/`savePlatformServiceProperties` — NOT the station daemon | `BPlatformServiceView` or equivalent |
| **platform-daemon-file** | A `BDaemonSessionView` with a `DaemonFileUtil` file-push contract | Opened from the **Platform tab** alongside the platform-service-plugin; pushes files to the host OS via the Platform daemon; feature tabs are NRE-version-gated | `BDaemonSessionView` |

This taxonomy extends B751's rung ladder with a second "target" axis — ladder rung says HOW MUCH WB, target axis says WHERE. All three targets can appear at any rung (rung 1 FE or rung 2 Manager). [ev: corpus B1105]

### PlatformServicePlugin WB view target [ev: retro wb-vendor-ux-wave3-vendor-drivers-deltas Δ6]

`PlatformServicePlugin` is a distinct WB view-target pattern for Platform-tab integration. It differs from the station-component pattern:

- **`@AgentOn` anchor:** the platform service agent (`platform:*Service`) — NOT a `BComponent` in the station space.
- **API contract:** the view must implement `poll()` (one-shot data refresh from the platform daemon), `lease()` (keep-alive while the view is open), and optionally `savePlatformServiceProperties()` (write config back to the daemon).
- **No station-space writes:** mutations go through the Platform daemon pathway, not through `BComponent.set()` with a station Context.
- **Commissioning note:** the Platform tab is only accessible when connected to the Platform node (not the station node); views registered here are invisible when connected to the station node only.

```java
// Skeleton — platform service view that polls the daemon for status
@NiagaraType(agent = @AgentOn(types = {"platform:MyPlatformService"}, requiredPermissions = "r"))
public class BMyPlatformServiceView extends BWbComponentView {
    @Override public void doLoadValue(BObject value, Context cx) {
        poll();   // fetch current state from the platform daemon
    }
    private void poll() { /* call platform daemon API, update widgets */ }
    private void lease() { /* keep-alive ping to the daemon while the view is open */ }
}
```

[ev: corpus B1099]

### Platform-daemon-file WB view target — BDaemonSessionView [ev: retro wb-vendor-ux-wave3-vendor-drivers-deltas Δ11]

The `BDaemonSessionView` + `DaemonFileUtil` pair is the Niagara Platform pattern for pushing host-OS configuration files from the WB Platform tab:

- **`BDaemonSessionView`** opens a session to the platform daemon and provides an API to read/write files on the host OS (not in the station BOG).
- **`DaemonFileUtil.pushFile(session, localFile, remotePath)`** copies a local temp file to the host OS path via the daemon session — the correct path for config files that live outside the station (e.g. `/etc/network/interfaces`, driver license files).
- **NRE-version-gated feature tabs:** when a feature requires a minimum NRE version, check `Sys.getNiagaraVersion()` at tab-render time and call `tab.setEnabled(false)` + tooltip explanation if the version is insufficient. Never silently hide tabs — always explain why they are disabled.

```java
// NRE-version gate on a Platform tab
Version minVersion = Version.make("4.14.0");
if (Sys.getNiagaraVersion().compareTo(minVersion) < 0) {
    advancedTab.setEnabled(false);
    advancedTab.setToolTipText("Requires Niagara 4.14 or later");
}
```

**Security note:** `BDaemonSessionView` carries MD5-digested credential storage in the `honAdvWirelessCfg` reference module (corpus B1104). When implementing a daemon-file view that stores credentials, use SHA-256 or stronger (see `types/security.md §9`). [ev: corpus B1104]

### OEM-on-stock-driver WB side notes [ev: retro wb-vendor-ux-wave3-vendor-drivers-deltas Δ3]

For the full OEM-on-stock-driver pattern (type-float slot, two-anchor manager mount, dual-FE registration, ORD-carrier navigation), see `types/driver-authoring.md §9.3`. WB-specific notes:

- **Dual-FE registration** is safe: Workbench dispatches by the exact `BTypeSpec` match; the OEM FE and the stock FE coexist without conflict as long as the OEM FE is `@AgentOn` the OEM type (not the stock base type).
- **Two-anchor manager mount:** register the shared OEM manager with `@AgentOn(types={"bacnet:BacnetDevice", "myOem:MyOemDevice"})` — the framework opens the OEM manager for both; the type-float slot discriminates which OEM plugin activates.
- **ORD-carrier navigation:** `ord|view:myOem:MyOemDeviceView` in a `<b-hyperlink>` PX widget opens the OEM view only for devices whose type matches `myOem:MyOemDevice`; the stock view remains active for untyped BACnet devices.

[ev: corpus B1096]

### Session-keyed learn state in command-driven managers [ev: retro wb-vendor-ux-wave3-vendor-drivers-deltas Δ4]

For the full `BStationMgrCommand` command SPI (registry discovery, CredentialsColumn lease+newCopy, HistoryId shorthand), see `types/driver-authoring.md §9.4`. WB-specific note for session-keyed state:

The `Static State for manager view continuity` pattern (see `§Static State` above) applies to **domain state that must survive a close/reopen**. For **learn/discovery state scoped to one session**, use a session key instead:

```java
// Per-session learn state — cleared automatically when the session ends
private static final Map<String, LearnState> SESSION_STATES =
    Collections.synchronizedMap(new WeakHashMap<>());

private LearnState getLearnState(Context cx) {
    return SESSION_STATES.computeIfAbsent(cx.getSessionId(), k -> new LearnState());
}
```

`WeakHashMap` keyed by session ID allows old sessions to be GC'd without an explicit cleanup hook. Do NOT use a `HashMap` with no eviction — it leaks one `LearnState` per Workbench session. [ev: corpus B1097]

### Thin non-driver CRUD manager template — SNMP pattern [ev: retro wb-vendor-ux-wave3-vendor-drivers-deltas Δ8]

When a **service** (not a driver) needs a table-manager CRUD view with credential support, the nSnmp module (~50–70 lines) is the reference archetype. It is smaller than the `BAbstractManager` non-driver template (`wb-widgets.md §Minimal non-driver custom-manager template`) because it needs no Discover:

Key structural rules:
- `@AgentOn` the service type (not a network or device).
- Use `MgrColumn.Prop` only — no `MgrColumn.PropPath`, no discover columns.
- Credential variant: add a `BPassword` prop column; in `MgrEdit.validate()` check `BPasswordStrength.DEFAULT` — reject credentials that do not meet the minimum strength:

```java
// Inside MgrEdit.validate() — credential strength check
BPassword pw = (BPassword) record.get(passwordProp);
if (!BPasswordStrength.DEFAULT.isSatisfiedBy(pw)) {
    throw new ValidationException("Password does not meet minimum strength requirements");
}
```

- `isLearnable() = false`, `makeLearn() = null`.
- Override `getNewTypes()` to restrict the allowed row type to the credential record type.

The ~50–70 line target is achievable only if column definitions and the MgrModel inner class are kept flat. Split into a separate `MgrModel` inner class as soon as it exceeds 3 columns. [ev: corpus B1101]

### Multi-perspective managers over one object graph [ev: retro wb-vendor-ux-wave3-vendor-drivers-deltas Δ9]

The Z-Wave module (corpus B1102) demonstrates the **multi-perspective manager** pattern: two or more `BAbstractManager` subclasses that present DIFFERENT VIEWS over the SAME underlying object graph (the `BZWaveNetwork` device subtree).

Key design rules for this pattern:

- **One source of truth:** all managers read and write the SAME `BZWaveDevice` components in the station space. Never duplicate data across manager-local state and the component tree.
- **Firmware-capability-gated columns:** show/hide manager columns based on a per-device capability flag (e.g. `device.supportsSecureCommands()`). Implement gating in `MgrModel.getColumnCount()` and `MgrModel.getColumnAt(i)` — return a shorter column array when the capability is absent. Never show a column that the device cannot support.
- **Raw-payload byte FE:** for columns that carry raw byte-array payloads (e.g. Z-Wave command-class data), register a custom `BWbFieldEditor` for `BByteArray` (or equivalent) that renders the bytes as hex with a length counter. Keep the byte FE in the `-wb` profile; never decode raw command-class data in the rt.
- **Device power-state column:** expose `devicePowerState` (mains / battery / unknown) as a dedicated `MgrColumn.Prop` with a custom `MgrCellRenderer` that renders a battery/plug icon. This column is always present regardless of firmware capability — power state is never gated.

```java
// Firmware-capability-gated column in MgrModel
@Override public int getColumnCount() {
    return device.supportsSecureCommands() ? FULL_COLUMNS.length : BASE_COLUMNS.length;
}
@Override public MgrColumn getColumnAt(int i) {
    return device.supportsSecureCommands() ? FULL_COLUMNS[i] : BASE_COLUMNS[i];
}
```

[ev: corpus B1102]

### OPC WB view layer — action-slot bridge and structured error display [ev: retro wb-vendor-ux-wave3-vendor-drivers-deltas Δ10]

WB-side complement to `types/driver-authoring.md §9.6`. The WB view layer for a protocol-adapter manager must:

- **Never import COM/native types.** The `-wb` jar must not reference `COMException`, `NativeException`, or any native-interop class. All COM/native error decode happens in the driver `-rt` layer; the WB view receives only a `DriverException` (or equivalent `BException`) with a human-readable message.
- **Lazy-browse expansion:** implement a `TreeModel` whose `getChildCount(node)` and `getChildren(node)` methods call the driver via an `invokeLater`-guarded async fetch. Show a "Loading…" placeholder node until the response arrives.
- **Security-gated state column:** add a `MgrColumn.Prop` for `securityState` with states `READY` / `FAULT` / `UNKNOWN`; render `FAULT` in red and disable write-action toolbar buttons when the state is not `READY`. The WB manager must enforce the gate — do not rely solely on the driver layer to refuse writes.

---

## -ux module JS toolchain reference `[ev: retro module-hardening-failure-modes-deltas Δ16]`

A `-ux` module that authors its own AMD/RequireJS JavaScript (a bajaux `@AgentOn` view or a custom widget type-extension) uses the Niagara-standard JS toolchain. Raw `rc/` content without this toolchain requires **ES5** — see `types/dashboard.md §JS build strategy` (UXS2/B1119 ES6/ES5 decision).

### Standard stack

| Component | Role |
|---|---|
| **`grunt-niagara`** | Grunt task library — provides `babel:dist`, `copy:dist`, `requirejs`, `karma` tasks |
| **Gradle `com.tridium.niagara-grunt` plugin** | Wires the Gradle build to `grunt`: `gruntBuild` task → runs grunt tasks in order |
| **RequireJS/AMD** | Module loader at runtime — modules are referenced by `nmodule/<module>/rc/<path>` IDs |
| **Babel** | ES6→ES5 transpilation (the `babel:dist` grunt task) |

### Gradle wiring in `<mod>-ux.gradle.kts`

```kotlin
plugins {
    id("com.tridium.niagara-module")
    id("com.tridium.niagara-grunt")        // adds gruntBuild task
}

tasks.named<GruntBuildTask>("gruntBuild") {
    tasks("babel:dist", "copy:dist", "requirejs")   // standard task order
}
```

### RequireJS AMD module ID convention

All `-ux` RC resources are addressed via RequireJS using the `nmodule/` prefix:

```javascript
// Inside a .js AMD module in this module's rc/ directory:
define(["nmodule/mymod/rc/myWidget", "nmodule/bajaui/rc/bajaui"], function(myWidget, bajaui) {
    // …
});
```

`nmodule/<module>/rc/<path>` resolves to the jar's `rc/<path>` resource of the named module. Never use a bare relative path — it breaks across module boundaries.

### When grunt IS required vs. when it must be removed

- **Keep `niagara-grunt`** when: the module AUTHORS its own AMD modules (bajaux type-extension, custom widget with its own `.js` files that need `babel:dist` + `requirejs` bundling).
- **Remove `niagara-grunt` entirely** when: the module only serves a pre-built static bundle (three.js, Chart.js, etc.) from `src/rc/` with no authored AMD modules — a leftover `niagara-grunt` dep adds a phantom `nodeHome` requirement and a no-op build step on every compile.

See `types/dashboard.md §JS build strategy` for the full decision matrix. `[ev: corpus B1132]`

[ev: corpus B1103]
