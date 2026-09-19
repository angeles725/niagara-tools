<!-- review-status: folded -->
# 2026-09-18 · kit · ux-wb-writesurface-rbac-deltas

**Session**: corpus-mining for build-n4-module (operator: generate all possible kit deltas).
**Delta count**: 4

## What happened

Mined B751 (wb ladder + recipes), B752 (ux three serving recipes / bajaux dialects / PX bindings /
RBAC contrast), B753 (playbook for our modules), B763 (write-surface five gates / pure RBAC seam).
Cross-checked every finding against `types/wb-widgets.md`, `types/dashboard.md`, `corpus-index.md`,
BUILD-STATE.md, and the retros `dashboard-servlet-write-surface-and-reader-authority`,
`r14-second-login`, `hidden-actions-not-invocable-and-runtime-anchor-verification`.

**Prior coverage confirmed (no duplicate proposed):**
- wb ladder rungs 0–3, FieldEditor recipe → `wb-widgets.md` ✓ [C4-PR2 fold]
- three serving recipes decision (servlet / bajaux / PX) → `dashboard.md §serving recipe` ✓
- OPERATOR_WRITE bit (not role name), fail-closed → `dashboard.md §RBAC` ✓
- DWS1 five gates + DWS2 pure RBAC seam (`canWrite(boolean)`) → `dashboard.md` ✓
- CSRF `X-Requested-With` guard inside pure `route()` → `dashboard.md` ✓
- vendor bajaux `permissions="unrestricted"` — do not copy → `dashboard.md` ✓
- U5 re-grading, per-Ord lock/423 gap (issue #49) → BUILD-STATE.md ✓
- `requiredPermissions` = view visibility only, never security → `wb-widgets.md` rule 6 ✓

Four net-new deltas remain, all from bajaux/PX internals that were indexed in `corpus-index.md`
but never extracted into a type guide.

## Evidence

- B751 §751.3 — Honeywell `BIHonDeviceModel`/`BIHonBacnetDeviceModel` plugin framework:
  `BThermostatDeviceModel.java:22-53` contributes columns + supported models + commands with
  zero Manager code; `HonDeviceModel.java:23` is the column engine that discovers plugins from
  the registry. Two ways to author rung 2: (a) subclass `BAbstractManager` + override
  `makeModel/makeController/makeLearn` (Tridium recipe — `BDriverManager.java:33-85`); (b)
  contribute only a device-model plugin that a shared Manager framework discovers (the Honeywell
  recipe — no Manager subclass required). [ev: corpus B751 §751.3]
- B752 §752.2 — bajaux data-channel dialects: (a) `fal.serverSideCall({typeSpec, methodName,
  value})` → `BSingleton implements BIServerSideCallHandler @AgentOn(requiredPermissions="ri")`,
  returns BValue/JSON, live refresh via Fox `subscriberMixIn`
  (`BFALServerSideCallHandler.java:29-176`, EagleHawk); (b) `baja.rpc({typeSpec, method, args})`
  → `@NiagaraRpc(permissions="…", transports={web,box})` static methods on a `BComponent`
  (`BThermostatWizardRPC.java:168-176`, TC/Sylk React SPA). Vendors use
  `permissions="unrestricted"` on path (b) — confirmed anti-pattern; our servlet-SPA does not
  use either dialect (REST-poll), but a builder choosing bajaux `@AgentOn` needs the decision.
  [ev: corpus B752 §752.2]
- B752 §752.3 — PX binding taxonomy across 270 `.px` files in the corpus: `BoundLabelBinding
  (ord=…, statusEffect="color") + ObjectToString(format="%out.value%")` for a read label;
  `SetPointBinding(ord=…, widgetEvent="actionPerformed", widgetProperty="selected")` for
  write-back; `ActionBinding(ord=…, widgetEvent="actionPerformed")` for invoke; cross-space `|`
  in ords, `|view:<module>:<ViewName>` for hyperlinks; absolute `layout="x,y,w,h"` geometry.
  (`VENOM_VAV_003n.px:32-35`, `Smart_IO.px:134`.) [ev: corpus B752 §752.3]
- B752 §752.1 + B753 §753.2 — PX complement to servlet-SPA: if an integrator wants to
  hand-author an equipment schematic without touching the SPA, a `.px` bound to the facade's
  slot ords is a zero-Java complement; the facade's `OPERATOR` slots are already the write
  surface; no Java build required; rides the Workbench PX runtime. Complementary, NOT a
  migration target. [ev: corpus B752 §752.1, B753 §753.2]

## Proposed kit deltas (propose-never-apply)

| Δ | Delta | Target file / § | Token |
|---|---|---|---|
| Δ1 | Under rung-2 ("BAbstractManager/BDeviceManager ONLY for containers of discovered children"), add the Honeywell device-model PLUGIN alternative: when a device family shares a Manager framework, contribute only a `BIHonDeviceModel` (columns + supported models + commands); the framework discovers plugins from the registry — zero Manager subclass code. Label the Tridium subclass recipe (a) and the Honeywell plugin recipe (b); recommend (b) for a device-family module, (a) for a standalone driver. | `types/wb-widgets.md §How much wb is enough` (after rung-2 entry) | [ev: corpus B751 §751.3] |
| Δ2 | Add §bajaux data-channel dialects: two channels for any bajaux `@AgentOn` view — (a) `serverSideCall` → `BSingleton implements BIServerSideCallHandler @AgentOn(ri)`, server returns BValue/JSON, live refresh via Fox `subscriberMixIn`; (b) `baja.rpc` → `@NiagaraRpc(permissions="…", transports={web,box})` static methods on a `BComponent`. Flag that vendors use `permissions="unrestricted"` on path (b) — do not copy. Note that our servlet-SPA uses neither (REST-poll); this section is for builders who choose the bajaux `@AgentOn` recipe. | `types/wb-widgets.md` new §bajaux data-channel dialects | [ev: corpus B752 §752.2] |
| Δ3 | Fill the `types/wb-widgets.md §PX` TODO with the PX binding taxonomy: `BoundLabelBinding(statusEffect="color") + ObjectToString` for read labels; `SetPointBinding(widgetEvent, widgetProperty)` for write-back setpoints; `ActionBinding` for invoke; cross-space `\|` in ords, `\|view:<module>:<ViewName>` for hyperlinks; absolute `layout="x,y,w,h"` geometry. Include a minimal read-label + write-setpoint snippet from the corpus exemplar (`VENOM_VAV_003n.px:32-35`). | `types/wb-widgets.md §PX authoring` (fills the existing TODO) | [ev: corpus B752 §752.3] |
| Δ4 | In `dashboard.md §ux — serving recipe`, after "engineer-authored equipment graphics → PX", add one sentence: PX is also a zero-Java COMPLEMENT to an existing servlet-SPA — an integrator adds a `.px` bound to the facade slot ords for equipment schematics; the facade's OPERATOR slots are already the write surface; no servlet change needed. This prevents treating "PX vs servlet" as mutually exclusive. | `types/dashboard.md §ux — servlet + SPA / Pick the right serving recipe` | [ev: corpus B752 §752.1, B753 §753.2] |

## Lessons

- Most of the write-surface and RBAC canon from B751–B763 was already folded in prior campaigns.
  The four net-new deltas come from bajaux internals (dialects, PX binding detail, Honeywell
  plugin pattern) — correctly indexed in `corpus-index.md` as B752 "bajaux data-channel
  dialects" and "PX bindings", but never extracted into a type doc.
- B763 §763.7 said "for the §18 retro" explicitly; those deltas (DWS1/DWS2) were already
  folded by the time this retro was written. What remained was the deeper B752/B751 internals.
- `wb-widgets.md §PX` has a standing TODO — Δ3 fills it directly from corpus evidence.
- Δ4 is a one-sentence clarification that prevents "PX vs servlet" from reading as an
  either/or choice, which is wrong for a module that already has a servlet-SPA.

---
**Status**: PENDING — INDEX row appended: `| 2026-09-18-ux-wb-writesurface-rbac-deltas.md | kit | 2026-09-18 | pending | 4 |`
