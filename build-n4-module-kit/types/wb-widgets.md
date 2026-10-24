# Type: Workbench widget / PX view (wb) — SEED (feed as built)

Workbench-side UI: Swing `BWbView`/editor tools, or PX views/widgets. Exemplar to READ: chihuahua-wb (`BatchLinkEditor`, a proper WB view). Corpus has the PX subsystem deeply (`corpus-nav connections 197`, blocks B179–B213) + module-anatomy B636.

Not yet fully documented — seed pointers, feed via the retro step when you build one:

- **`-wb` profile** only; register the view type + an `@AgentOn` (agent-on the component type) so it opens from the tree. `requiredPermissions` on the agent = view VISIBILITY only, never security.
- **`-wb` gradle MUST include `com.tridium.convention.niagara-home-repositories` plugin:** a `-wb` subproject cannot resolve `:baja`/`:web` unless `id("com.tridium.convention.niagara-home-repositories")` is declared in its `<mod>-wb.gradle.kts`. The `-rt` gradle already declares it (chihuahua-wb relies on that); the `-wb` gradle does NOT inherit it, so declare it there too. Missing this plugin causes "Could not resolve `:baja`" at Gradle sync time. `[ev: retro apillm-headless-servlet-rt-4.14-deltas Δ3]`
- **wb-agent registration — the `@AgentOn` in `@NiagaraType` is what writes the `<agent>` entry:** at compile time javac's `@NiagaraType` annotation processor (not Slot-o-Matic) updates `module-include.xml` with the type's registry information, its agents included (devguide `slot-o-matic.txt:191-193`). The Apillm build saw no `<agent>` block from an inline-only `@AgentOn` and added one by hand `[INFER — the processor did not run in that build]`; if Workbench does not discover the agent, check that the processor ran before hand-writing the block. Verify with `grep -r '<agent>' <mod>-wb/` after adding an `@AgentOn`. `[ev: retro apillm-headless-servlet-rt-4.14-deltas Δ5]`
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
- **Author the LEAST wb the ladder allows:** rung 0 = **nothing** — the default property-sheet/wire-sheet views already render standard slots (kitControl ships 152 rt types with only 2 wb field editors); rung 1 = a `BWbFieldEditor @AgentOn(<value type>)` for ONE composite value that renders badly; rung 2 = a `BAbstractManager`/`BDeviceManager` ONLY when the component is a container of learned/discovered children; rung 3 = a `BWbComponentView` ONLY for non-tabular interaction. **Our ColdRoomPan/CompPan components sit at rung 0 — do not build a Manager.** [ev: retro corpus-index-rt-authoring-and-organization-blocks · B751]
- **Rung-2 has two authoring recipes — pick by context:** (a) **Tridium subclass recipe** — extend `BAbstractManager`/`BDeviceManager`, override `makeModel`/`makeController`/`makeLearn` (`BDriverManager.java:33-85`); use for a standalone driver that owns its own Manager container. (b) **Honeywell device-model PLUGIN recipe** — implement only `BIHonDeviceModel` (or `BIHonBacnetDeviceModel`); the plugin contributes columns, supported models, and commands (`BThermostatDeviceModel.java:22-53`); the shared `HonDeviceModel` framework (`HonDeviceModel.java:23`) discovers plugins from the registry — **zero Manager subclass code**. Prefer (b) for a device-family module that slots into an existing Manager framework; prefer (a) only when no shared framework exists. [ev: corpus B751 §751.3]
- **FieldEditor recipe (rung 1):** ctor builds the widgets → `linkTo(widget, textModified, setModified)` → override `doLoadValue`/`doSaveValue`/`doSetReadonly`; compose child editors via `BWbFieldEditor.makeFor(value)`; register with `@AgentOn(<that value type>)`. [ev: retro corpus-index-rt-authoring-and-organization-blocks · B751]
- **A Honeywell "Wizard" is usually a tabbed `BWbComponentView`, not a `BWizard`:** step-panes = tabs, backed by rt `BJob`s launched from an agent `BMenu`. Always mutate through the space (`newTransaction` / `tx.commit`); undo is inherited from the `Command`/`MgrEdit` framework, never hand-rolled. [ev: retro corpus-index-rt-authoring-and-organization-blocks · B751]
- **Rung 4+ (reference ceiling — do NOT imitate):** a "vendor programming environment" tier exists above rung 3 in the Tridium `ace` module (the ACE driver for Distech "Titan" controllers): own wire sheet + app wizard + catalog palette + opcode expression editor. This is the ceiling of the rung ladder, documented here as a concrete upper bound for scope and code-review discussions. Our modules stay at rung 0–1. Reaching rung 4 implies owning a custom Baja domain language runtime — not a practical target for our use cases. `[ev: retro wb-vendor-ux-wave3-vendor-drivers-deltas Δ5]` `[ev: corpus B1098]`

## wb/model testable seam — exemplar-backed

- **DWB1 — `-wb` is off-station testable via a `model/` lambda-injection seam (HIGH):** keep the business logic in a Baja-free `model/` package; inject the slot-availability check as a `Predicate<String>` so the model has zero Baja imports. The `BWidget` view stays the thin adapter. `chihuahua-wb`'s `LinkSlotNameUtil` + 33 pure `@Test` cases are the proven pattern. **This upgrades `-wb` from seed to exemplar-backed for the model layer.** [ev: corpus B762]
- **Dual-surface `@AgentOn` registration:** write `@NiagaraType(agent={@AgentOn(types={"mod:Type"}, requiredPermissions="r")})` on the view/FE; javac's `@NiagaraType` annotation processor (not Slot-o-Matic) emits `<type><agent><on type=…/></agent></type>` (devguide `slot-o-matic.txt:193`); multi-type `types={…}` = one view over several source types. `[ev: corpus B780]`

## bajaux data-channel dialects

> Applies only when you choose the **bajaux `@AgentOn` + `BIJavaScript`** recipe. Our servlet-SPA uses REST-poll and needs neither dialect — this section guides builders who pick the bajaux path. [ev: corpus B752 §752.2, §752.4]

Two dialects for delivering data to a bajaux `@AgentOn` view:

- **(a) `fal.serverSideCall` dialect (`BSingleton` channel):** implement `BIServerSideCallHandler` on a `@NiagaraSingleton`; annotate the view with `@AgentOn(requiredPermissions="ri")`. The server returns `BValue`/JSON; live refresh via Fox `subscriberMixIn` in the JS view. Exemplar: `BFALServerSideCallHandler.java:29-176` (EagleHawk). Use when the view needs reactive/push-style updates from a singleton service.
- **(b) `baja.rpc` dialect (`@NiagaraRpc` channel):** annotate static methods on a `BComponent` with `@NiagaraRpc(permissions="…", transports={web,box})`; the browser calls `baja.rpc({typeSpec, method, args})`. Exemplar: `BThermostatWizardRPC.java:106-114` (TC/Sylk React SPA). Use when the view calls discrete wizard-style operations.

**Anti-pattern — `permissions="unrestricted"` on dialect (b):** Honeywell TC/Sylk RPCs ship `permissions="unrestricted"` — no server-side auth check. Do NOT copy this; the browser is never the security boundary. Add a server-side `OPERATOR_WRITE` (or stricter) check on every mutating RPC, matching the pattern in `dashboard.md §RBAC`. [ev: corpus B752 §752.5]

## Good -wb artifact doctrine (DWB1 exemplar — 11 rules)

> Every line applies to any -wb profile with Java sources. [ev: corpus B809] [ev: corpus B817]

1. **Profile isolation**: keep all -wb code in the `-wb` Gradle profile; never import `-wb` classes from `-rt` or `-ux`. `[INFER — kit doctrine]`
2. **Model/view split (DWB1)**: isolate business logic in a Baja-free `model/` package; the view injects Baja access as a `Predicate<String>` or similar lambda — zero Baja imports in `model/`. [ev: corpus B809] [ev: corpus B817]
3. **Thin view adapter**: the `BWbComponentView`/`BWbFieldEditor` subclass does layout + delegation only; it never contains business logic that belongs in `model/`. [ev: corpus B809] [ev: corpus B817]
4. **Off-load UI-thread traversal**: `doInvoke` bodies must NOT call `getNavChildren`/`getNavNodes`/`BqlQuery` directly on the Swing EDT; run any station traversal off the UI thread — a `BSimpleJob.run()`, or the view-work queue `WbViewEventWorker.getInstance().invokeLater(r)` as Tridium's `BAbstractManager` does (workbench-wb vineflower `BAbstractManager.java:448`; that queue is a background worker thread, `WbViewEventWorker.java:31-37`, NOT the UI thread) — and marshal the UI update back to the UI thread with `BWidget.invokeLater(r)` (bajaui-wb vineflower `BWidget.java:871-873` → `UiEnv.get().invokeLater`). [ev: corpus B809] [ev: corpus B817]
   > **Corrected (2026-10-24):** this rule said "marshal the UI update back with `WbViewEventWorker.invokeLater()`" (citation audit batch 6, 4ee1b853); that method only enqueues on a background worker, so it never reaches the UI thread. Found while fixing chihuahua `BBatchLinkEditor` (client branch fix/r10-batch-link-editor-ui-thread).
5. **Narrow `@AgentOn`**: target the narrowest specific type(s), never `@AgentOn(types="baja:Component")`. [ev: corpus B809] [ev: corpus B817]
6. **`requiredPermissions` = view only**: never use the agent's `requiredPermissions` as a security gate; it controls VISIBILITY, not authorisation — authorisation belongs in the action/servlet. [ev: corpus B752 §752.5] [ev: corpus B809]
7. **Pure `@Test` in `srcTest`**: the `model/` package must be testable without a running station; aim for ≥20 pure unit tests covering the predicate-injection seam `[INFER]`. [ev: corpus B762]
8. **No `getNavChildren` in `doInvoke`**: if a search or tree-walk is needed, run it in a `BSimpleJob` (rule 4); the `doInvoke` body only submits the job. [ev: corpus B809] [ev: corpus B817]
9. **Non-empty scaffold gate**: ship a `-wb` jar only when it has ≥1 `.class` OR ≥1 palette `<p n=` entry; an all-empty scaffold (`verify-module.sh` `wb-scaffold` WARN) means nothing was compiled or registered. [ev: corpus B817 §817.9 L9]
10. **Declare what you reference, no phantoms**: `module.xml` declares the companion `-rt`, `workbench-wb`, `bajaui-wb` and every module whose types the code uses — a MISSING companion `-rt` is the hard failure (`ClassNotFound` at Workbench start); a dep listed but never referenced is a phantom (`verify-module.sh` `phantom-dep` WARN, advisory). [ev: corpus B809 §809.5]
11. **Fail-closed authorization for WB controls:** a control whose authorization mixin or PIN is absent must be DISABLED **and** HIDDEN by default — never open-by-default. An ABSENT mixin is the closed case; a pin of `-1` is not — in the Galileo exemplar `-1` means any level passes. On `loadValue()`, check the pin; if the user lacks the required permission call `widget.setEnabled(false)` **and** `widget.setVisible(false)`. A missing mixin ≠ "unprotected" — it equals "no access". `[ev: retro honeywell-wb-rt-wb-deltas Δ3]` `[ev: corpus B1079]`

**DWB1 exemplar — chihuahua-wb `model/` tree** (commit `175eee8` [deferred: unverifiable — no local chihuahua checkout and no corpus block names the commit], `angeles725/chihuahua`):
```
chihuahua-wb/src/com/angeles/chihuahua/wb/
  model/
    DirectionButtonUtil.java   — Baja-free direction label logic
    DirectionLabelUtil.java    — Baja-free label text util
    LinkSlotNameUtil.java      — slot name generation, Predicate-injected
    PendingLink.java           — Baja-free pending-link value object
    PendingLinkBuilder.java    — builder for PendingLink
    SearchResultUtil.java      — Baja-free search-result formatter
  BBatchLinkEditor.java        — thin BWbComponentView adapter
```
The `model/` package has zero Baja imports; station access is passed per call as a `Predicate<String>` (`LinkSlotNameUtil.generate(Predicate<String>, String)`). 33 `@Test` cases across the six classes run without a station. [ev: corpus B762]

## Manager recipe — `BAbstractManager` wiring reference [ev: retro wb-manager-framework-deltas Δ1]

> Rung-2 detail (extends the ladder above): reach here only when the component is a container of discovered/learned children. Our ColdRoomPan/CompPan panels sit at rung 0 — do not build a Manager unless the ladder says so.

### 6-object anatomy and boolean gates

Every `BAbstractManager` owns six support objects, each built once in `init()` from a protected factory method, plus three read-only gates derived from them:

| Object | Factory | Role |
|--------|---------|------|
| `MgrModel` | `makeModel()` → `new MgrModel(this)` | Database-side table model; owns the columns from its `makeColumns()` |
| `MgrController` | `makeController()` → `new MgrController(this)` | Commands — New, Edit, Delete, Discover; `promptForNew()` lives here |
| `MgrLearn` | `makeLearn()` (null by default) | Discover/learn pane — its own `makeColumns()`, **independent** of `MgrModel`'s columns |
| `MgrTagDictionary` | `makeTagDictionary()` (null when the target has no `TagDictionaryService`) | Tags pane |
| `MgrTemplate` | `makeTemplate()` (null by default) | Device-template pane |
| `MgrState` | `makeState()` → `new MgrState()` | State cached across a close/reopen |

The other classes you meet: `MgrColumn` (column descriptor — see taxonomy below) and `BMgrEditDialog` (the Add/Edit dialog; one inner `ColumnInput` per edit column, `BMgrEditDialog.java:225-227`; `BAbstractManagerFE` is NOT a class, "FE" is a driver-naming convention only).

The three gates are `public final` — you cannot override them; return null from the factory instead (`BAbstractManager.java:354-395`):

| Gate method | Is | Controls |
|-------------|----|---------|
| `isLearnable()` | `getLearn() != null` | Shows/hides the Discover button |
| `isTaggable()` | `getMgrTagDictionary() != null` | Shows/hides the Tags pane |
| `isTemplatable()` | `getMgrTemplate() != null` | Shows/hides the Template pane |

`[ev: corpus B1088]`

### `MgrColumn` taxonomy reference card [ev: retro wb-manager-framework-deltas Δ2]

| Column type | When to use | Key attribute |
|-------------|-------------|---------------|
| `MgrColumn.Name` | Display name of the child component | — |
| `MgrColumn.Type` | Tridium type spec of the child (e.g. `"mod:MyPoint"`) | `typeSpec` |
| `MgrColumn.Prop` | A **flat** slot on the child component (accessible directly via `slot.asProperty()`) | `prop` name |
| `MgrColumn.PropPath` | A **nested** slot via a property path (e.g. `{proxyExt, precision}`); traverses the child component tree | `Property[]` path |

Use `Prop` for direct slots; use `PropPath` for mixin or extension slots. The discover table (`MgrLearn.cols[]`) and database table (`MgrModel.cols[]`) carry **independent** column arrays — do not share or reuse the same `MgrColumn` instances across both.

`[ev: corpus B1089]`

**One `MgrColumn.Prop` per key `@NiagaraProperty` on the proxy ext (PD-02):** each key property on a `BProxyExt` subclass (e.g. `deviceAddress`, `pointOffset`, `scaling`) maps to exactly ONE `MgrColumn.Prop`; the column reads/writes that single slot — no compound columns or manual ORD construction. This ensures the Add/Edit dialog pre-fills the slot value without extra coding. `[ev: retro wb-vendor-ux-rt-wb-pattern-deltas Δ2]`

**`addDefault*Columns()` + `addCustom*Columns()` template hook (PD-16):** structure a manager's column definition as two protected methods: `addDefaultColumns(List<MgrColumn>)` (supplies the built-in columns every subclass gets) and `addCustomColumns(List<MgrColumn>)` (empty by default; subclasses override to extend). Call both from `makeModel()`. This lets a subclass add columns without replacing the default set. `[ev: retro wb-vendor-ux-rt-wb-pattern-deltas Δ16]`

**`isDefault` blank-suppression for polymorphic manager columns (PD-17):** when a manager row type can be one of several subtypes (e.g. digital vs analog vs enum proxy), some columns are irrelevant for certain row types. Override `MgrColumn.isCellValid(MgrEditRow row)` to return `false` for rows where the column does not apply — the manager greys the cell out rather than showing a blank or a cast exception (`MgrColumn.java:220-228`; there is no `isDefault`). `[ev: retro wb-vendor-ux-rt-wb-pattern-deltas Δ17]`

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
| Container whose hand-curated children you create/edit/delete in a table (our Apillm importers/exporters) | YES (B1091: migrate to it) | Only for a pure display view |
| Standalone driver that owns its Manager container | YES (Tridium subclass recipe) | No |

Our Apillm importer/exporter managers are hand-built `BWbComponentView`+`BTable` today; B1091 recommends migrating any create/edit/delete manager view to `BAbstractManager` + `MgrColumn.Prop` (free `MgrEdit` dialog, column reordering, CSV export, tags) and keeping the hand-built recipe for pure display views.

**`BWbComponentView` for non-table specialized UIs — terminals, schedule editors (PD-08):** use `BWbComponentView` (not a `BAbstractManager`) for any UI that is not a table of discovered/learned rows — VT100 terminals, job schedule editors, wizard flows, or any non-tabular interaction. These views live at the same rung-3 as a manager but serve fundamentally different UX patterns. `[ev: retro wb-vendor-ux-rt-wb-pattern-deltas Δ8]` `[ev: corpus B1058]`

`[ev: corpus B1091]`

### Plugin-extensible device-type manager via SPI (Honeywell recipe) [ev: retro honeywell-wb-rt-wb-deltas Δ1]

When multiple device families share one manager container, use the **plugin SPI** instead of subclassing `BAbstractManager` for each family. Implement only `BIHonDeviceModel` (or `BIHonBacnetDeviceModel`); the plugin contributes columns, supported model specs, and commands. The shared framework discovers plugins at runtime through Honeywell's own helper `NiagaraRegistryUtil.instance().getImplementersOfTypeSpec(String typeSpec)` (`com.honeywell.devicemanager.util`, called from `HonDeviceCache.java:120`) — zero Manager subclass code per device-family module.

Key responsibilities of a `BIHonDeviceModel` implementer:
- `getSupportedModels()` — returns the `List<String>` of device model names this plugin handles.
- `getSupportedColumns()` / `getSupportedDiscoveryColumns()` — return the `List<HonColumn>` / `List<HonDiscoveryColumn>` the plugin contributes to the shared tables.
- `getSupportedCommands(BWidget)` — returns the `List<HonMgrCommand>` scoped to devices this plugin owns; `getDeviceType()` names the device `Type` (`BIHonDeviceModel.java`).

Register each plugin via a plain `<type>` in `module.xml`; no `@AgentOn` is required. Prefer this recipe over a Manager subclass when a shared framework already exists and you are adding a new device family on top of it; prefer the Tridium subclass recipe (rung-2(a) above) only when no shared framework exists. `[ev: corpus B1077]`

### Static State for manager view continuity [ev: retro honeywell-wb-rt-wb-deltas Δ8]

When a manager must survive a close/reopen without losing discovery state (e.g. the last learn result or a scan job still in progress), use the framework's `MgrState` seam: override `makeState()` to return a `static` inner `State extends MgrState` (the Honeywell Modbus manager's `State extends DeviceState` keeps `lastLearn` and `lrnJob`). `saveState()` caches it on deactivation under `MgrState.toKey(manager)` (the manager type) and `restoreState()` calls `restore(manager)` on the next load (`BAbstractManager.java:533-559`).

```java
public class BMyDeviceManager extends BAbstractManager {

    // Cached by the framework across close/reopen, keyed by the manager type.
    static class State extends MgrState {
        BObject[] lastLearn;   // domain state only — never widgets

        @Override public void save(BAbstractManager manager)    { super.save(manager);    /* copy from the manager */ }
        @Override public void restore(BAbstractManager manager) { super.restore(manager); /* copy back */ }
    }

    @Override protected MgrState makeState() { return new State(); }
}
```

**Anti-pattern:** do NOT put Swing *widgets* (e.g. a `BTable` model) in the state — only DOMAIN STATE. Widget singletons across two simultaneously open manager windows cause threading violations. `[ev: corpus B1078]`

### Dialog pre-population — live RT objects and toRow() [ev: retro wb-vendor-ux-rt-wb-pattern-deltas]

**`toRow()` result carries ALL config slots so the add-dialog is pre-populated (PD-03):** the discover/learn path's `toRow()` result object MUST carry ALL config slots (address, name, type, scaling, ranges) — not just the discovery key. The `MgrController.promptForNew()` dialog reads these slots to pre-fill the Add dialog; a sparse `toRow()` forces the operator to fill fields manually that the driver already knows. `[ev: retro wb-vendor-ux-rt-wb-pattern-deltas Δ3]` `[ev: corpus B1057]`

**Dialogs accept live RT objects, not a re-fetch (PD-07):** when the Edit or Add dialog is pre-populated from an already-resolved RT component, pass the live object directly to the dialog rather than re-resolving from an ORD. Re-fetching inside the dialog introduces a race (the state may change between the manager render and the dialog open) and adds a latency flash. Accept the live RT component in the dialog constructor `[INFER]`. `[ev: retro wb-vendor-ux-rt-wb-pattern-deltas Δ7]`

**COV/capability-bit `toRow()` pre-population (PD-21):** when a protocol device advertises capabilities as a bitfield (e.g. `servicesSupported` in BACnet, capability flags in Modbus), decode the bitfield inside `toRow()` and set the corresponding pre-filled fields on the result object. The manager row then shows pre-checked capability boxes at discover time without a second round-trip. Bit decode belongs in `toRow()`, not in the Add dialog. `[ev: retro wb-vendor-ux-rt-wb-pattern-deltas Δ21]` `[ev: corpus B1073]`

### Minimal non-driver custom-manager template [ev: retro wb-manager-framework-deltas Δ4]

Use this template when a **service or container** (not a driver) needs a manager view — e.g. a service that owns a set of configuration records with an Add/Delete UI. For the driver-centric recipe with full `makeLearn()` + network discovery, see `envCtrlDriver` (B956).

Key constraints for a non-driver custom manager:
- `@AgentOn` the **container/service** type, NOT a `BNetwork`/`BDevice`.
- Use `MgrColumn.Prop` only (no `PropPath`, no Discover) — keep it flat.
- Do NOT override `makeLearn()`: the default returns `null`, so the final `isLearnable()` is false.
- Transaction: the standard New flow is `MgrController.promptForNew()` → `MgrEdit.commit()` → `MgrModel.addInstances()` (`Mark.moveTo(container, null)`); a one-step programmatic add calls `container.add(...)` (Path A, §Create-point-from-wb).

```java
// -wb profile only
@NiagaraType(agent = @AgentOn(types = {"myMod:MyService"}, requiredPermissions = "r"))
public class BMyServiceManager extends BAbstractManager {
    public static final Type TYPE = Sys.loadType(BMyServiceManager.class);
    @Override public Type getType() { return TYPE; }

    @Override protected MgrModel makeModel() { return new Model(this); }
    // no makeLearn()/makeTemplate() override: both return null (no Discover, no templates)

    class Model extends MgrModel {
        Model(BAbstractManager mgr) { super(mgr); }

        @Override protected MgrColumn[] makeColumns() {
            return new MgrColumn[] {
                new MgrColumn.Name(MgrColumn.EDITABLE),                      // display name column
                new MgrColumn.Prop(BMyRecord.enabled,  MgrColumn.EDITABLE),  // flat boolean slot
                new MgrColumn.Prop(BMyRecord.priority, MgrColumn.EDITABLE),  // flat int slot
            };
        }
        @Override public MgrTypeInfo[] getNewTypes() { return MgrTypeInfo.makeArray(BMyRecord.TYPE); } // allowed New types
        @Override public Type[] getIncludeTypes()    { return new Type[] { BMyRecord.TYPE }; }
    }
}
```

`[ev: corpus B1091]`

## Vendor-grade -wb UX patterns (cross-vendor survey PD-01..PD-10)

> These rules come from the wb-vendor-ux cross-vendor survey (B1054–B1061). Apply to any driver or service that ships a `-wb` profile.

- **Every `BBasicNetwork`/driver module ships a `@AgentOn` WB device manager (WB-presence rule, PD-01):** a driver that adds a `BBasicNetwork` subclass MUST register at least a minimal `BAbstractManager` or `BWbComponentView` via `@AgentOn` on the network or device type. Relying solely on the default property-sheet view forces operators into raw slot editing — the vendor-grade bar is a table view at minimum. For the full manager recipe see `§Manager recipe`. `[ev: retro wb-vendor-ux-rt-wb-pattern-deltas Δ1]` `[ev: corpus B1057]`

- **Every long RT operation is a `BOrd` action on a job-bar (PD-04):** any operation whose RT side takes >1 s (firmware upload, device scan, config push) MUST follow the `submit → sync → resolve → registerForEvents → jobBar.load` recipe. The `BOrd` points at the `BJob`; `jobBar.load(ord)` attaches the progress UI; the button is disabled while `jobBar.isRunning()`. Never block the Swing EDT or poll in a `javax.swing.Timer`. See `types/logic-authoring.md §BSimpleJob + Fox file-channel streaming` for the RT job authoring side. `[ev: retro wb-vendor-ux-rt-wb-pattern-deltas Δ4]` `[ev: corpus B1055 §1055.6]`

- **Prune inapplicable inherited commands in `makeCommands()`/`getAgents()` (PD-05):** a WB view inherits all parent-type commands and agents by default. Override `getAgents()` on the RT network/component type (as `BCcnNetwork` does) to filter out agents that do not apply (e.g. the default property-sheet agent when a custom manager is the preferred view). In the manager's `MgrController`, override `makeCommands()`, call `super.makeCommands()` and remove the commands the protocol does not support (`match`, `quickMatch`, `newFolder`, `edit`, …). Remove, do not just disable, to keep the UX clean. `[ev: retro wb-vendor-ux-rt-wb-pattern-deltas Δ5]` `[ev: corpus B1057]` `[ev: corpus B1061 PD-05]`

- **Every `BOrd` ref slot needs a station-scoped WB picker — never the file-space default (PD-06):** any `-wb` form or manager that lets the operator configure a `BOrd` pointing at a station component MUST present a station-component chooser (`BComponentChooser` or the `targetType` facet; see `§Field editors — PD-FE1`). The default `BOrd.NULL` dialog opens a file-space chooser (`C:\`) — useless in a station context. This extends the `lint-wb-file-chooser` candidate from `§-wb station-space picker`. B1061 PD-06 states the wider rule: every `baja:Ord` ref slot needs a WB navigation helper — a manager column, a double-click to the resolved component, or a command. `[ev: retro wb-vendor-ux-rt-wb-pattern-deltas Δ6]` `[ev: corpus B1061 PD-06]` `[ev: corpus B1087 §1087.2]`

- **Cross-cutting WB concerns use a `BWbService`, not per-component agents (PD-10):** when a `-wb` concern spans multiple component types in a session (e.g. a shared job scheduler, a session-keyed credential cache, or a global drag-and-drop coordinator), implement it as a `BWbService` subtype (`BWbService extends BWbNavNodeTool implements BIService`, `BWbService.java:13`) registered in the `-wb` `module.xml`. Per-component agents that share state across component instances via `static` fields is the anti-pattern — it creates invisible coupling and GC pressure. `[ev: retro wb-vendor-ux-rt-wb-pattern-deltas Δ10]` `[ev: corpus B1061]`

## -wb station-space picker, table repaint, and view refresh rules

- **Station-component pickers in a `-wb` manager MUST scope to the STATION COMPONENT SPACE (`slot:`), NOT the file system:** `BWbFieldEditor.dialog(this, title, BOrd.NULL)` defaults to the `file:` space and opens a File Chooser browsing `C:/` (or the WSL root) — NOT the station component tree. To browse the station, use `BComponentChooser` filtered to `BControlPoint` (or the appropriate base type). The `BComponentChooser` recipe in corpus B1087 §1087.2 is the reference. For a `BOrd` slot, the zero-code fix is the `targetType` facet (see §Field editors PD-FE1). **Lint candidate `lint-wb-file-chooser`:** flag a `-wb` class that calls `BWbFieldEditor.dialog(this, …, BOrd.NULL)` with no `targetType` or `BComponentChooser` nearby. `[ev: retro apillm-headless-servlet-rt-4.14-deltas Δ8]` `[ev: corpus B1087]`
- **`-wb` table repaint gotcha — `table.relayout()` after in-place model mutation:** after mutating a `BTable`'s model in place (e.g. clearing and repopulating its rows), calling `table.setModel(sameModelInstance)` is a NO-OP in some BJ builds — the view shows stale data. Call `table.relayout()` (or rebuild the model instance then call `relayout()`) to force a full repaint. Surfaced as "Toggle Recursive shows old value" — the HTTP serving was correct; only the WB view was stale. `[ev: retro apillm-headless-servlet-rt-4.14-deltas Δ15]`
- **A `-wb` manager/view showing live component state MUST refresh on activation AND subscribe to its component:** a view that only rebuilds its model in `doLoadValue()` / on-command shows STALE state after a station restart or an external change until the user re-opens the view. Add: (a) the model rebuild in `doLoadValue()` AND in an overridden `handleComponentEvent(BComponentEvent)`, which fires on component changes; (b) the reload in `activated()` (the `BWbView` hook called when the view becomes the shell's active view); (c) the subscription through `registerForComponentEvents(...)` in `doLoadValue()` (or the default `autoRegisterForComponentEvents`), released by `unregisterForAllComponentEvents()` in `deactivated()` — there is no `doUnloadValue()`, `doActivated()` or `componentShown()` hook (`BWbView.java:125-135`). Concrete API and depth: Δ1/Δ2 below. Pairs with B1070 `DynamicTableModel` and corpus B29 `serviceStarted` registration. **Lint candidate `lint-wb-refresh`:** the tightened form in Δ1 below. `[ev: retro apillm-headless-servlet-rt-4.14-deltas Δ17]` `[ev: retro apillm-wb-subscription-refresh-and-points-deltas Δ1]` `[ev: corpus B29]`
  - **Δ1 CONCRETE API for the rule above — `registerForComponentEvents`/`handleComponentEvent`, NOT a raw `sys.Subscriber`:** the WB-client refresh seam is `BWbComponentView.registerForComponentEvents(BComponent subject, int depth)` (the single-arg `registerForComponentEvents(BComponent)` form subscribes at depth 0, the subject only) + an OVERRIDDEN `handleComponentEvent(BComponentEvent)` that rebuilds the model and calls `table.relayout()` + `BWbView.activated()` (rebuild on tab re-entry) + `BWbView.deactivated()` calling `unregisterForAllComponentEvents()`. The default `autoRegisterForComponentEvents = true` already subscribes the subject at depth 0 on `doLoadValue()`; `handleComponentEvent()` is a NO-OP by default, so the default wiring alone never refreshes. `javax.baja.sys.Subscriber` is the RT-side pattern — it does not apply to a WB view/manager; do not cross the two seams. **Lint candidate `lint-wb-refresh` (tightened):** a `BWbComponentView` subclass with a `doLoadValue()` that builds a model but the class has no `registerForComponentEvents(` call and no overridden `handleComponentEvent(` — replaces the earlier "no `Subscriber` field" formulation, which named an API that does not apply here and would never actually fire. `[ev: retro apillm-wb-subscription-refresh-and-points-deltas Δ1]` `[ev: corpus B1140 — BWbEditor.java:282-286; BWbComponentView.java:84-91,192-194,303]`
  - **Δ2 subscribe-DEPTH rule — depth is distance to the slots actually rendered, not a fixed "1":** a DIRECT child of the subscribed root (e.g. an importer subscribing its own `BApillmImportMap` mapping rows) needs depth 1; a FOLDER-NESTED child (e.g. a servlet subscribing exports through an intermediate `BApillmExports` folder: servlet → folder → `BApillmExport`) needs depth 2 — one level for the folder, one more for the export inside it. Getting the depth one level too shallow silently reproduces the Δ1 stale-defaults symptom for the deeper children only, while the direct children look fine. Pairs with `getDeviceManagerSubscribeDepth()`/`MgrModel.getSubscribeDepth()` on the driver-manager side — see `types/driver-authoring.md §getDeviceManagerSubscribeDepth() — deep slot-tree discovery`. `[ev: retro apillm-wb-subscription-refresh-and-points-deltas Δ2]` `[ev: corpus B1140]`
- **A manager column rendering an ORD OUTSIDE the subscribed subtree needs its OWN lease, not subject-subtree depth (Δ3):** subject-subtree `registerForComponentEvents(subject, depth)` only ever reaches descendants of `subject`. A column that resolves a per-row `BOrd` property to an EXTERNAL target (e.g. `BApillmImportMap.targetOrd` → an arbitrary station point outside the importer subtree, or `BApillmExport.ord` → an arbitrary export target) and reads that target's live value/status is reading a component the subject-subtree subscription never covers — it stays at its resolve-time snapshot (or "unresolved" on the first, async, resolve) no matter how the depth above is tuned. Fix: for EACH resolved external target, call `target.loadSlots()` once (whether that first read is synchronous on the WB proxy layer is still open: B1140-G1) and `registerForComponentEvents(target, 0)` (leases future refreshes for that target specifically); guard repeat resolves with `isRegisteredForComponentEvents(target)` so successive `rebuild()` calls do not re-subscribe the same target. Unregister happens automatically with the view's `unregisterForAllComponentEvents()` in `deactivated()` (§ above) — it also drops every per-target lease taken this way. **Lint candidate `lint-wb-external-ord-value`** (implemented, `toolbelt/lint-wb-external-ord-value.sh`): a manager method that resolves an ORD property to an external object (`<x>Ord().get(` / `<x>ord.get(`) and reads a live value/status off it (`.get("out")`, `getOutStatusValue(`, `BStatusValue`) with no `registerForComponentEvents(` call in the same method → WARN. `[ev: retro apillm-wb-subscription-refresh-and-points-deltas Δ3]` `[ev: corpus B1140-G1; Apillm BApillmImporterManager.java rowFor()]`

## WB user preferences — per-user JSON under userHome/ [ev: retro honeywell-wb-rt-wb-deltas Δ2]

For remembered WB state (visible columns, selected commands, last-used values) that is per-user and must survive a manager close/reopen, store a small JSON file under the Niagara user home, as Honeywell's `HonUserCommandUtil` does: `<userHome>/UserCmdPrefs/<username>/UserCmdPrefs.json`, written with Tridium's bundled `com.tridium.json`, directories created as needed. No rt slots are needed — the WB view reads/writes the file directly on load/save, keeping the slot schema clean.

```java
// Persist per-user WB prefs — manager open → read; manager close → write
File prefs = new File(Sys.getNiagaraUserHome(),      // Sys.java:63, the Niagara user home
                      "MyModulePrefs/" + userName + "/MyModulePrefs.json");
// write/read the JSON with com.tridium.json, the library HonUserCommandUtil uses
```

Guard every read — `HonUserCommandUtil` returns an empty list on any `IOException`, so the first open (no file yet) falls back to defaults. This pattern avoids adding transient or per-user state to the rt component model. `[ev: corpus B1077]`

## Integer `visibilityPin`/`actionPin` authorization for PX widgets and manager rows [ev: retro honeywell-wb-rt-wb-deltas Δ4]

Add a `visibilityPin` (int, default `-1`) and an `actionPin` (int, default `-1`) as `@NiagaraProperty` slots on a PX widget or manager-row type. At render time, read the user's `BGalileoUserPin` mixin:
- Mixin ABSENT → `widget.setVisible(false)` and `widget.setEnabled(false)` (fail-closed rule above).
- Mixin present → compare its `userPinLevel` with `visibilityPin` (fails → `setVisible(false)`) and with `actionPin` (fails → `setEnabled(false)`).
- A pin of `-1` lets any level pass — it is NOT a closed default.

```java
// Galileo-style pin check: an absent user mixin is the closed case
BValue pin = user.get(SlotPath.escape("galileoSupervisor_GalileoUserPin"));
if (pin == null) { widget.setVisible(false); widget.setEnabled(false); return; }
// else compare the mixin's userPinLevel with visibilityPin / actionPin (-1 = any level)
```

Reusable across manager rows and PX widgets without per-type auth boilerplate. `[ev: corpus B1079]`

## Field editors — station-component pickers, the null-ord gotcha, and point-creation from WB

> Rung-1 detail (extends the FieldEditor recipe above): most modules need NO custom field editor (rung 0). Reach here only when an rt slot must pick a station component, or an importer-style Manager must create points from WB. [ev: retro wb-field-editors-deltas]

- **Facet-driven field-editor selection — pick the FE from the slot's facets (PD-15):** instead of hard-coding a field editor class for a slot, build `fieldEditor` facets (e.g. `BFacets.make("fieldEditor", BString.make("driver:TimeZoneSelectionFE"))`) and pass them to `BWbFieldEditor.makeFor(value, facets)`, which selects the named FE — the bacnet manager's `TimeZoneConfigColumn` does this in code. This avoids a custom `BWbFieldEditor` subclass just to pick a field editor `[INFER]`. `[ev: retro wb-vendor-ux-rt-wb-pattern-deltas Δ15]` `[ev: corpus B1067]`

- **`targetType` facet — zero-code station-component picker for a `BOrd` slot (PD-FE1):** a `BOrd` slot whose value is null has no scheme, so `BOrdFE` stays on the file-system chooser (`BFileOrdChooser`) — the null-ord gotcha, by design (a null value has no `BIOrdChooser` scheme override to fire). To browse the station component tree instead, put a `targetType` facet on the property (`facets=BFacets.make("targetType", BString.make("baja:Component"))`, or the equivalent `@Facets`); `BOrdFE` then selects `BComponentChooser` automatically, even for a null ord. No `-wb` code required — prefer this over a custom FE for `BOrd`→component slots. [ev: retro wb-field-editors-deltas Δ1] `[ev: retro apillm-headless-servlet-rt-4.14-deltas Δ20]`
  - **Δ19 SUPERSEDED:** a previous approach proposed a custom registered `BWbFieldEditor` for station-ORD slots. That approach (Δ19) is superseded by the simpler `targetType` facet (PD-FE1 above). See `[ev: retro apillm-headless-servlet-rt-4.14-deltas Δ19]` for the original proposal; the facet-based fix requires zero `-wb` code. `[ev: retro apillm-headless-servlet-rt-4.14-deltas Δ19]`
  - **Sibling runtime gotcha — do not confuse the two:** the entry above is about a *picker* falling back to the file-system chooser on a null ord. A DIFFERENT, RUNTIME gotcha lives one layer down: a manager wizard that maps a freshly-created point's `point.getSlotPathOrd()` (which can itself be null before the component is mounted/resolvable) straight into a `BOrd`-typed `set()` call NPEs inside `BComplex.set`, not in any field editor. See `types/issues-and-gotchas.md §H3` for the symptom, root cause, and guard. `[ev: retro wb-mapping-ord-npe-and-wsl-windows-jdk Δ1]`
- **`targetType=baja:ControlPoint` for pick/create a control point slot (PD-FE2):** use `targetType=baja:ControlPoint` (or a relevant subtype such as `baja:NumericWritable`) in the `@BFacets` annotation on a `BOrd` property that should refer to a control point — the chooser is then filtered to points only, not every component in the station. `[ev: retro apillm-headless-servlet-rt-4.14-deltas Δ21]`
- **Point-creation from WB — use the Manager plumbing, not `BComponent.add()` (PD-FE2-create):** for an importer-style `BAbstractManager`, override `getNewTypes()` to return the writable types you allow, then rely on `MgrController.promptForNew()` → `MgrEdit.commit()` → `Mark.moveTo(container)`. A hand-rolled `container.add(...)` skips the type-picker dialog and the `MgrEdit` transaction, so use it only for the programmatic one-step path (Path A below), which still relays the add to the station via Fox. [ev: retro wb-field-editors-deltas Δ2]
- **Filtered sub-type chooser — subclass `BComponentChooser`, NEVER override `baja:Ord` (PD-FE3):** when a slot must pick only a subtype (e.g. only `BNumericWritable`), subclass `BComponentChooser` with a `selectFilter` `RefFilter` — `(parent, slot) -> slot.isProperty() && parent.get(slot.asProperty()).getType().is(BTargetType.TYPE)` — and register it via `@AgentOn(types={"myModule:MyOrdAlias"})` on a **custom** ord type. Never override the global `baja:Ord` agent — that would hijack every ord slot in the station. The `FIELD_EDITOR` facet is the alternative when you cannot change the slot type; for `BOrd`→component prefer the `targetType` facet (PD-FE1). For a station-subtree-filtered chooser, subclass `BComponentChooser` and call `BComponentChooser.prompt(root, path, displayFilter, selectFilter)` directly, or add new target types via `parent.add(name, point, cx)` for `[BNumericWritable/BBooleanWritable/BStringWritable/BEnumWritable]`. For `BEnumWritable`, set a minimal `BEnumRange` facet so it constructs without error. [ev: retro wb-field-editors-deltas Δ3] `[ev: retro apillm-headless-servlet-rt-4.14-deltas Δ22]`
- **One-step "create the referenced component" in a manager (Δ23):** when a manager's rows reference station components (e.g. an importer mapping → writable point), the "Add…" flow SHOULD let the user CREATE the target component in one dialog — type dropdown (Numeric/Boolean/String/Enum), name, folder → `new B*Writable()` + `folder.add(uniqueName, point, cx)` + wire the ref ORD — with a "(use existing…)" fallback. NEVER force pre-creating points and hand-picking ORDs separately. For `BEnumWritable`, set a minimal `BEnumRange` facet on construction so it serializes. `[ev: retro apillm-headless-servlet-rt-4.14-deltas Δ23]` `[ev: corpus B1087]`

## Create-point-from-wb — the consolidated recipe (Δ5)

> Consolidates the two point-CREATION paths already named piecemeal above (PD-FE2-create, Δ23) into one recipe, plus the `BEnumRange` overloads and unique-naming helper the prior bullets did not spell out. `[ev: retro apillm-wb-subscription-refresh-and-points-deltas Δ5]` `[ev: corpus B1142]`

- **Path A — direct one-step add (programmatic, no dialog):** `container.add(uniqueName, point, null)` where `container` is any `BComponent` acting as a folder (`BFolder` inherits `add()` unchanged and declares no slots of its own). This is the same call the "Add Mapping…" one-step recipe above (Δ23) and `BApillmImporterManager`/`BApillmManager` use to create the writable point AND relay it to the station JVM via Fox in one call — no `-wb` transaction plumbing needed for this path. `[ev: corpus B1142 §1142.1–1142.2]`
- **Path B — the standard "New" dialog (`MgrController`/`MgrEdit`/`Mark`):** already the PD-FE2-create recipe above — `getNewTypes()` → `MgrController.doNew()` opens the type-picker + `BMgrEditDialog` → `MgrEdit.commit()` wraps the field-editor results in a `Transaction` → `MgrModel.addInstances()` finishes with `Mark.moveTo(container, null)`. Use Path A for programmatic/one-step creation (e.g. inside a custom "Add…" command); use Path B when the user should see Tridium's standard type-picker dialog. `[ev: corpus B1142 §1142.3]`
- **`BEnumRange.make()` overloads for an enum point's facets:** `BEnumRange.make(String[] tags)` — ordinals are the array index (0, 1, 2, …); `BEnumRange.make(int[] ordinals, String[] tags)` — explicit, non-contiguous ordinals; `BEnumRange.make(Type frozenBase, int[] ordinals, String[] tags)` — a frozen base enum type extended with dynamic entries. Apply via `point.setFacets(BFacets.make(BFacets.RANGE, range))` (or `BFacets.makeEnum(range)`) BEFORE or immediately after `add()` — an unranged `BEnumWritable`/`BEnumPoint` serializes without ordinal/tag names. `[ev: corpus B1142 §1142.5; corpus B1141 §1141.4]`
- **Unique slot naming — `ObjectUtil.uniqueName(container, baseName)`:** appends a numeric suffix (`_2`, `_3`, …) until the name is free in `container`; use it whenever a point/mapping is created from a JSON key, imported label, or any other name that is not already guaranteed unique. `[INFER — workbench utility, not in docSource; ev: corpus B1142 §1142.2]`

## Multi-part WB view — BTabbedPane+BLabelPane recipe (PD-19) `[ev: retro wb-vendor-ux-rt-wb-pattern-deltas Δ19]`

When a `-wb` view needs multiple sections (e.g. status tab + config tab + diagnostics tab), use Baja's native `BTabbedPane`+`BLabelPane` — NOT `javax.swing.JTabbedPane`. The native types participate in the Baja Swing lifecycle and fire `doLoadValue`/`doSaveValue` correctly per tab.

**Recipe (`BTabbedPane` + one `createTPage*()` per tab):**

```java
private BTabbedPane tabs;

// in the view's constructor
tabs = new BTabbedPane();
tabs.addPane("Status",        createTPageStatus());       // BLabelPaneContainer.addPane(String, BWidget)
tabs.addPane("Configuration", createTPageConfig());
tabs.addPane("Diagnostics",   createTPageDiagnostics());
setContent(tabs);                                          // BWbPlugin.setContent

private BWidget createTPageStatus() {
    BEdgePane p = new BEdgePane();
    // … add widgets …
    return p;
}
```

**Rules:**
- One `createTPage*()` factory method per tab — keeps the constructor compact.
- **Conditional tab:** wrap in `if (childIsPresent())` before `tabs.add(...)` to hide a tab when the required child is absent.
- **`doLoad`/`doSave` symmetry:** override `doLoadValue()` to set widget values from the loaded component; override `doSaveValue()` to write widget values back. Keep them paired — a missing `doSaveValue()` causes silent loss of edits.
- **Status tab layout:** a `BEdgePane` with the `BJobBar` in `setNorth(...)` and the `BTable` in `setCenter(...)` — the standard vendor layout for a status/command tab.

`[ev: corpus B1070]`

**The `BTabbedPane` selection is TRANSIENT — persist the tab by label text (PD-24):** `BTabbedPane` has no selection property (only `getSelectedLabelPane()`/`selectLabelPane(...)`), so the selected tab does not survive a view close/reopen or a `doSaveValue()` call. To remember the selected tab across open/close, persist the tab's **label text** (not its index, which is unstable if tabs are added/removed) in a per-user pref file (see `§WB user preferences`) or in a `String` field. Restore in `doLoadValue()` by walking `tabs.getChildren(BLabelPane.class)` for the pane whose `getLabel().getText()` equals the saved label and calling `tabs.selectLabelPane(p)`. `[ev: retro wb-vendor-ux-rt-wb-pattern-deltas Δ24]` `[ev: corpus B1076]`

**Popup editor — one editor for the whole component** `[ev: corpus B1315 §1315.1 rank 7]` [ev: retro devguide415-horizons-authoring-security Δ10]:
a class that extends `BWbEditor` and implements `BIPopupEditor` (`workbench-wb` `javax.baja.workbench.popup`) opens
instead of the per-field Property Sheet when the component is clicked on the wiresheet (devguide `popupEditor.txt:10-12`).
Use it for a setpoint dialog that must validate fields together; other horizons: `types/horizons.md`.

## Commands — lexicon-driven `Command` / `ToggleCommand` `[ev: corpus B1317 §1317.1]` [ev: retro devguide415-r2-runtime-contracts-formats Δ4]

`new Command(owner, lexicon, keyBase)` (also the `BModule` and `Properties` overloads, and every `ToggleCommand`
constructor, which calls `super`) reads exactly four keys: `keyBase.label`, `keyBase.icon` (an ORD to a 16x16 png),
`keyBase.accelerator` and `keyBase.description` (`bajaui-wb` `javax/baja/ui/Command.java:61-64`, 4.15.6.20 source).
A missing key leaves that field null (`Lexicon.get` returns null, `javax/baja/util/Lexicon.java:326-335`); an accelerator that
`BAccelerator.make` rejects only prints "Invalid accelerator for Command <keyBase>" to stdout and the command loads
without it (`:68-79`). Ship all four keys per command in `module.lexicon` (lexicon chain: `types/lexicon.md`), and
override `doInvoke()` to do the work. Owed advisory: Go lint G91 `command-lexicon-keys`
(`ledger.d/2026-10-23-r8-u10-platform-templates.md`). [deferred: retro devguide415-r2-runtime-contracts-formats Δ4]

## PX authoring — binding taxonomy

PX files are XML authored in the Workbench PxEditor, shipped as module resources, and rendered in the browser via the Hx/HTML5 profile. The corpus contains 270+ `.px` files; the binding patterns below are the recurring skeleton. [ev: corpus B752 §752.3]

When PX is the right recipe and how a Px page is laid out: `types/dashboard.md` § ux "Px rules".

**Core binding types:**

| Intent | Binding type | Key attributes |
|--------|-------------|----------------|
| Read label with status tint | `BoundLabelBinding` + `ObjectToString` converter | `ord="…"`, `statusEffect="color"`, `format="%out.value%"` |
| Write-back setpoint / enum | `SetPointBinding` | `ord="…"`, `widgetEvent="actionPerformed"`, `widgetProperty="selected"` |
| Invoke an action | `ActionBinding` | `ord="…"`, `widgetEvent="actionPerformed"` |
| Hyperlink to a view | any binding | ORD ends in `\|view:<module>:<ViewName>`; `\|` = cross-space separator |

**Geometry:** only a `CanvasPane` lays out its children by position: it reads each child's `layout="x,y,w,h"`, where
each value is px, `%` of the pane, or `pref` (preferred size) (`organized/docSource/docSource-doc/extracted/bajaui-wb/javax/baja/ui/pane/BCanvasPane.java:355-405`, `BLayout.ABS/PERCENT/PREF`);
`GridPane`, `BorderPane`, `EdgePane` and the other panes ignore `layout` and size children by their own rules
(`columnCount`, edges). Cross-space ords use `|` (pipe) as the space separator; e.g. `station:|slot:/Services/MyService`.

**Minimal read-label + write-setpoint snippet** (from `VENOM_VAV_003n.px:36-40, :375-377` — [ev: corpus B752 §752.3]):
```xml
<!-- Read label: displays current value with status coloring -->
<BoundLabel layout="10,10,100,20">
  <BoundLabelBinding ord="slot:/Setpoint" statusEffect="color">
    <ObjectToString name="text" format="%out.value% °C"/>
  </BoundLabelBinding>
</BoundLabel>
<!-- Write-back boolean setpoint (check box) -->
<CheckBox layout="10,35,40,20">
  <SetPointBinding ord="slot:/AlarmEnable"
    widgetEvent="actionPerformed" widgetProperty="selected"/>
</CheckBox>
```

See also: `types/dashboard.md §serving recipe` for the PX-vs-servlet decision and the PX complement use case.

See also: `docs/module-best-practices.md` (rt/ux/wb do & don't).

---

## WB view-target taxonomy — three axes [ev: retro wb-vendor-ux-wave3-vendor-drivers-deltas Δ12]

The wave-3 survey (B1094–B1105) identified a second axis that extends the rung ladder: **where** the WB view is anchored, not just how complex it is. Three named targets:

| Target axis | `@AgentOn` anchor | Lifecycle / contract | Typical `@NiagaraType` base |
|-------------|-------------------|----------------------|-----------------------------|
| **station-component / driver** | A `BComponent`, `BNetwork`, `BDevice`, or `BService` subclass in the station component space | Opened from the Workbench Nav tree; participates in the normal station lifecycle | `BWbComponentView`, `BAbstractManager`, `BWbFieldEditor` |
| **platform-service-plugin** | A platform-service type (e.g. `platPower:PowerMonitorPlatformService`, `requiredPermissions="W"`) | Opened from the **Platform tab**; calls `poll()`/`lease(…)`/`savePlatformServiceProperties()` on the platform service — NOT the station | `BWbComponentView` |
| **platform-daemon-file** | A `BDaemonSessionView` with a `DaemonFileUtil` file-push contract | Opened from the **Platform tab** alongside the platform-service-plugin; pushes files to the host OS via the Platform daemon; feature tabs are NRE-version-gated | `BDaemonSessionView` |

This taxonomy extends B751's rung ladder with a second "target" axis — ladder rung says HOW MUCH WB, target axis says WHERE. All three targets can appear at any rung (rung 1 FE or rung 2 Manager). [ev: corpus B1105]

### PlatformServicePlugin WB view target [ev: retro wb-vendor-ux-wave3-vendor-drivers-deltas Δ6]

`PlatformServicePlugin` is a distinct WB view-target pattern for Platform-tab integration. It differs from the station-component pattern:

- **`@AgentOn` anchor:** the platform-service type (e.g. `platPower:PowerMonitorPlatformService`) — NOT a `BComponent` in the station space; the view itself is a plain `BWbComponentView`.
- **API contract:** the view CALLS these on the platform service — `poll()` + `lease(5)` in `doLoadValue()` (activates a polling lease on the remote daemon), then `registerForComponentEvents()`; `doSaveValue()` sets the values and calls `savePlatformServiceProperties()` to persist them to the daemon.
- **No station-space writes:** mutations go through the Platform daemon pathway, not through `BComponent.set()` with a station Context.
- **Commissioning note:** the Platform tab is only accessible when connected to the Platform node (not the station node); views registered here are invisible when connected to the station node only.

```java
// Skeleton — platform service view that polls the daemon for status
@NiagaraType(agent = @AgentOn(types = {"platform:MyPlatformService"}, requiredPermissions = "r"))
public class BMyPlatformServiceView extends BWbComponentView {
    @Override public void doLoadValue(BObject value, Context cx) {
        BMyPlatformService service = (BMyPlatformService) value;
        service.poll();       // refresh state from the platform daemon
        service.lease(5);     // keep the polling lease alive while the view is open
        registerForComponentEvents(service);
        // build or update widgets
    }
}
```

[ev: corpus B1099]

### Platform-daemon-file WB view target — BDaemonSessionView [ev: retro wb-vendor-ux-wave3-vendor-drivers-deltas Δ11]

The `BDaemonSessionView` + `DaemonFileUtil` pair is the Niagara Platform pattern for pushing host-OS configuration files from the WB Platform tab:

- **`BDaemonSessionView`** opens a session to the platform daemon and provides an API to read/write files on the host OS (not in the station BOG).
- **`DaemonFileUtil` + `FileTransferMessage` + `BDaemonFileSpace`** (the platform-daemon file channel, the same one firmware update and cert install use) carry config files to and from the host OS — the correct path for config that lives outside the station; the exemplar exchanges WiFi config as XML and BLE config as JSON. `DaemonFileUtil` has no `pushFile` method (its statics include `transfer`, `pipe`, `listFiles`, `findFile`).
- **NRE-version-gated feature tabs:** when a feature requires a minimum NRE version, check `Sys.getBajaVersion()` (`Sys.java:272`) at tab-render time. The exemplar suppresses the BLE tab below its minimum version; the kit prefers disabling it and showing why `[INFER — kit doctrine]`.

```java
// NRE-version gate on a Platform tab
Version minVersion = new Version("4.14.0");
if (Sys.getBajaVersion().compareTo(minVersion) < 0) {
    tabs.setEnabledLabelPane(advancedTab, false);   // BTabbedPane.setEnabledLabelPane
    // put "Requires Niagara 4.14 or later" in a label inside the tab
}
```

**Security note:** in the `honAdvWirelessCfg` reference module the BLE manager (`BHoneywellControllerBleManager`, a `BWbComponentView`) stores the BLE password as an MD5 hex digest in the BLE config it pushes (corpus B1104). When implementing a daemon-file view that stores credentials, use SHA-256 or stronger (see `types/security.md §9`). [ev: corpus B1104]

### OEM-on-stock-driver WB side notes [ev: retro wb-vendor-ux-wave3-vendor-drivers-deltas Δ3]

For the full OEM-on-stock-driver pattern (type-float slot, two-anchor manager mount, dual-FE registration, ORD-carrier navigation), see `types/driver-authoring.md §9.3`. WB-specific notes:

- **Dual-FE registration:** a thin subclass of the stock FE `@AgentOn` the stock type (`BHonBacnetBitStringEditor extends BBacnetBitStringFE`, overriding only the WB-context render path) next to a fresh `BWbFieldEditor` `@AgentOn` the OEM wrapper type (`BHonBacnetBitStringFE`); each FE is `@AgentOn` its own type, so they coexist.
- **Two-anchor manager mount:** one manager `@AgentOn` two tree locations (`BHonBacnetAlarmManager` on `HonBacnetAlarmDeviceExt` and `HonBacnetNotificationClassFolder`), dispatching on `instanceof` in `doLoadValue()`.
- **ORD-carrier navigation:** rt point types carry an ORD (`getAlarmViewHyperlink()`); a WB dialog reads it per point and a WB service hyperlinks to it (`BHonWbService.hyperlink(ord)`).

[ev: corpus B1096]

### Session-keyed learn state in command-driven managers [ev: retro wb-vendor-ux-wave3-vendor-drivers-deltas Δ4]

For the full `BStationMgrCommand` command SPI (registry discovery, CredentialsColumn lease+newCopy, HistoryId shorthand), see `types/driver-authoring.md §9.4`. WB-specific note for session-keyed state:

The `Static State for manager view continuity` pattern (see `§Static State` above) applies to **domain state that must survive a close/reopen**. For **learn/discovery state scoped to one session**, key the cache by the session: `BNiagaraFileManager` saves the Fox session id it learned under and invalidates the cached learn state when the id changes, so a reconnect never shows a stale learn table. [ev: corpus B1097]

### Thin non-driver CRUD manager template — SNMP pattern [ev: retro wb-vendor-ux-wave3-vendor-drivers-deltas Δ8]

When a config table (not a driver) needs a table-manager CRUD view with credential support, the nSnmp module (~50–70 lines) is the reference archetype. It is smaller than the `BAbstractManager` non-driver template (`wb-widgets.md §Minimal non-driver custom-manager template`) because it needs no Discover:

Key structural rules:
- `@AgentOn` the `*Table` type (e.g. `nSnmp:UsmUserTable`, `requiredPermissions="r"`), not a network or device.
- Use `MgrColumn.Prop` only — no `MgrColumn.PropPath`, no discover columns.
- Credential variant: add a `BPassword` prop column; in the inner `Edit`'s `MgrEdit.validate()` check the pass phrase against `BPasswordStrength.DEFAULT` — reject credentials that do not meet the minimum strength:

```java
// Inside MgrEdit.validate() — credential strength check
// isPasswordValid(char[], Consumer<Localizable>) — BPasswordStrength.java:392; there is no isSatisfiedBy
if (!BPasswordStrength.DEFAULT.isPasswordValid(passPhrase, msg -> { /* collect the reason */ })) {
    // reject: report the collected reason and keep the dialog open
}
```

- `isLearnable() = false`, `makeLearn() = null`.
- Override `getNewTypes()` to restrict the allowed row type to the credential record type.

The ~50–70 line target is achievable only if column definitions and the inner `MgrModel` (`makeColumns()` + `getNewTypes()`) are kept flat. [ev: corpus B1101]

### Multi-perspective managers over one object graph [ev: retro wb-vendor-ux-wave3-vendor-drivers-deltas Δ9]

The Z-Wave module (corpus B1102) demonstrates the **multi-perspective manager** pattern: three `BFolderManager` views (`BDeviceSceneManager`, `BNetworkSceneManager`, `BNetworkSceneExtManager`) that present DIFFERENT VIEWS over the SAME underlying scene objects from the device, one-scene and catalogue viewpoints.

Key design rules for this pattern:

- **One source of truth:** all managers read and write the SAME RT scene components (`BZWaveSceneActuatorConf`, `BZWaveNetworkSceneConfig`, `BZWaveNetworkScene`) in the station space. Never duplicate data across manager-local state and the component tree.
- **Firmware-capability-gated UI:** gate what the dialog offers on the device's capability — the inclusion dialog shows its `networkWide` check box only when `BFirmwareCapability.getSupportsExplorerFrames()` is true. Never offer an option the device cannot support.
- **Raw-payload byte FE:** for raw byte payloads (Z-Wave command-class data) the module ships its own `-wb` FE (`BZWaveBlobFE`, comma-delimited bytes). Keep the byte FE in the `-wb` profile; never decode raw command-class data in the rt `[INFER]`.
- **Battery wake-up column:** `BZWavePointManager` adds a `BZWaveProxyExt.nextWakeUpIn` column (battery devices report their next scheduled wake-up) plus `age` (value staleness).

```java
// Capability-gated option (the BZWaveDeviceManager inclusion pattern)
boolean explorer = capability.getSupportsExplorerFrames();
// open the inclusion options dialog; show its networkWide check box only when explorer is true
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
| **`grunt-niagara`** | Grunt task library — provides `babel:dist`, `copy:dist`, `requirejs`, `karma` `[INFER]` tasks |
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

- **Keep `niagara-grunt`** when: the module AUTHORS its own AMD modules (bajaux type-extension, custom widget with its own `.js` files that need `babel:dist` + `requirejs` bundling) `[INFER — kit doctrine]`.
- **Remove `niagara-grunt` entirely** when: the module only serves a pre-built static bundle (three.js, Chart.js, etc.) from `src/rc/` with no authored AMD modules — a leftover `niagara-grunt` dep adds a phantom `nodeHome` requirement and a no-op build step on every compile `[INFER — kit doctrine]`.

See `types/dashboard.md §JS build strategy` for the full decision matrix. `[ev: corpus B1132]`
