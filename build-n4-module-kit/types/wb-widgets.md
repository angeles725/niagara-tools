# Type: Workbench widget / PX view (wb) — SEED (feed as built)

Workbench-side UI: Swing `BWbView`/editor tools, or PX views/widgets. Exemplar to READ: chihuahua-wb (`BatchLinkEditor`, a proper WB view). Corpus has the PX subsystem deeply (`corpus-nav connections 197`, blocks B179–B213) + module-anatomy B636.

Not yet fully documented — seed pointers, feed via the retro step when you build one:

- **`-wb` profile** only; register the view type + an `@AgentOn` (agent-on the component type) so it opens from the tree. `requiredPermissions` on the agent = view VISIBILITY only, never security.
- **PX views:** authored in Workbench PxEditor (GUI), shipped as module resources + registered as an agent view on a type; the same `.px` renders in the browser via the Hx/HTML5 profile. Editable fields via kitPx field editors; background image as a module resource. (Corpus B180/B194/B202/B203.)
- **PX vs custom servlet dashboard:** PX = declarative, Workbench-authored, native. A `BWebServlet` + SPA (types/dashboard.md) = custom HTML5, no PxEditor. Pick per need; an HMI that just points a browser at a URL takes either.
- Same build (Java 8 + slotomatic) and METHODOLOGY rules.

## How much wb is enough — the ladder (climb ONLY when needed)
- **Author the LEAST wb the ladder allows:** rung 0 = **nothing** — the default property-sheet/wire-sheet views already render standard slots (kitControl ships 152 rt types with only 2 wb field editors); rung 1 = a `BWbFieldEditor @AgentOn(<value type>)` for ONE composite value that renders badly; rung 2 = a `BAbstractManager`/`BDeviceManager` ONLY when the component is a container of learned/discovered children; rung 3 = a `BWbComponentView` ONLY for non-tabular interaction. **Our ColdRoomPan/CompPan components sit at rung 0 — do not build a Manager.** [ev: retro corpus-index · B751]
- **Rung-2 has two authoring recipes — pick by context:** (a) **Tridium subclass recipe** — extend `BAbstractManager`/`BDeviceManager`, override `makeModel`/`makeController`/`makeLearn` (`BDriverManager.java:33-85`); use for a standalone driver that owns its own Manager container. (b) **Honeywell device-model PLUGIN recipe** — implement only `BIHonDeviceModel` (or `BIHonBacnetDeviceModel`); the plugin contributes columns, supported models, and commands (`BThermostatDeviceModel.java:22-53`); the shared `HonDeviceModel` framework (`HonDeviceModel.java:23`) discovers plugins from the registry — **zero Manager subclass code**. Prefer (b) for a device-family module that slots into an existing Manager framework; prefer (a) only when no shared framework exists. [ev: corpus B751 §751.3]
- **FieldEditor recipe (rung 1):** ctor builds the widgets → `linkTo(widget, textModified, setModified)` → override `doLoadValue`/`doSaveValue`/`doSetReadonly`; compose child editors via `BWbFieldEditor.makeFor(value)`; register with `@AgentOn(<that value type>)`. [ev: retro corpus-index · B751]
- **A Honeywell "Wizard" is usually a tabbed `BWbComponentView`, not a `BWizard`:** step-panes = tabs, backed by rt `BJob`s launched from an agent `BMenu`. Always mutate through the space (`newTransaction` / `tx.commit`); undo is inherited from the space, never hand-rolled. [ev: retro corpus-index · B751]

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
