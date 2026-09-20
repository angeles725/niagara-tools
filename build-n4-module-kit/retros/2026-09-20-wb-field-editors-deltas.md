<!-- review-status: pending -->
# 2026-09-20 · kit · wb-field-editors-deltas

**Session**: niagara-research focus `wb-field-editors` (B1084–B1087, commit 8a79fbaac) — the `BWbFieldEditor` framework: class hierarchy, `@AgentOn` registry, `makeFor()` resolution, `BOrdFE` anatomy and the null-ord file-chooser gotcha, `BIPopupEditor` / `dialog()` / `FIELD_EDITOR` facet, and the custom FE recipe with `BComponentChooser` + `BComponent.add()`.

**Delta count**: 3

## What happened
Reconstructed the WB slot-editing seam end-to-end: the `BWbFieldEditor` agent registry (type-hierarchy-aware `makeFor()`), why `BOrdFE` defaults to a file-system dialog for null `BOrd` slots, how the `targetType` facet overrides that without writing a custom editor, the `dialog()` static method + `FIELD_EDITOR` facet + `BWbProfile` filter, and the full recipe for a custom station-component chooser using `BComponentChooser` with a type-filtering `RefFilter` and `MgrController.promptForNew()` for point-creation from WB. Remittances: `BFlexAddressFE` covered as subject in B1074; `BWbComponentView` covered in B1057/B1058/B1070.

Kit-relevant angle: our modules mostly don't need custom field editors (rung-0/1 per MBP3), but when they do — e.g. a station-component picker in an importer-style manager like Apillm's — the zero-code `targetType` facet fix, the `BComponentChooser` recipe, and the `getNewTypes()`/`MgrController.promptForNew()` point-creation flow all belong in the how-to guide's `-wb` section and the `wb-widgets` reference.

## Evidence
- B1084 (`BWbFieldEditor` hierarchy: `makeFor()` resolution order facet → agents → profile → default; `@AgentOn` type-hierarchy walk; `BWbFieldEditorBinding` lifecycle: `targetChanged()` → `loadValue()`, `save()` → `parent.set()`). `[ev: corpus B1084]`
- B1085 (`BOrdFE` anatomy: `defaultBrowse.info = BFileOrdChooser` at `doLoadValue()` entry; null-ord stays on file dialog; `targetType` facet check at lines 254–257 before scheme-based resolution; `BComponentChooser @AgentOn baja:SlotScheme/HandleScheme/Component`; `BIOrdChooser` SPI). `[ev: corpus B1085]`
- B1086 (`BIPopupEditor.getEditor()`; `BWbFieldEditor.dialog()` — `makeFor()` + `loadValue()` + `BDialog.open()`; `IDialogContentProvider.getDialogContent()` N4.13+; `FIELD_EDITOR` facet bypasses agent registry; `ProfileFilter` checks `agent.getAppName()`). `[ev: corpus B1086]`
- B1087 (custom FE recipe: `@AgentOn` + `doLoadValue/doSaveValue/doSetReadonly`; `BComponentChooser.prompt()` with explicit `displayFilter`/`selectFilter` `RefFilter` lambda; `MgrController.promptForNew()` → `MgrEdit.commit()` → `Mark.moveTo()` point-creation flow; `BComponent.add(name, value, cx)` direct API; `MgrModel.getNewTypes()` type picker; PD-FE1/PD-FE2/PD-FE3 proposed in §1087.7). `[ev: corpus B1087]`

## Proposed kit deltas (propose-never-apply)
| Δ | Delta | Target file / § | Token |
|---|---|---|---|
| Δ1 | PD-FE1 document the `targetType` facet pattern in the authoring guide: when an rt `BOrd` slot should browse the station component tree instead of the file system, add `@BFacets(values={"targetType=baja:Component"})` to the property declaration — zero `-wb` code required, `BOrdFE` selects `BComponentChooser` automatically even for null ords | `docs/how-to-create-an-n4-module.md` §-wb / `types/wb-widgets.md` | `[ev: corpus B1085]` |
| Δ2 | PD-FE2 document the point-creation-from-WB flow for importer-style managers: override `getNewTypes()` on the `BAbstractManager` subclass to return the desired writable types, then rely on `MgrController.promptForNew()` → `MgrEdit.commit()` → `Mark.moveTo(container)` — do NOT call `BComponent.add()` manually; the manager's own plumbing handles the transaction and station relay | `types/wb-widgets.md` | `[ev: corpus B1087]` |
| Δ3 | PD-FE3 document the filtered station-component-chooser recipe for when a slot needs only a subtype (e.g. only `BNumericWritable`): subclass `BComponentChooser`, override the chooser constructor to pass a type-checking `selectFilter = (parent, slot) -> slot.isProperty() && parent.get(slot.asProperty()).getType().is(BTargetType.TYPE)`, and register via `@AgentOn(types={"myModule:MyOrdAlias"})` on a custom ord type — NEVER override the global `baja:Ord` agent | `types/wb-widgets.md` | `[ev: corpus B1087]` |

## Lessons
- The `BOrdFE` null-ord gotcha is by design for generic `BOrd` slots: a slot whose value is null has no scheme → no `BIOrdChooser` override logic fires → `BFileOrdChooser` remains the default. The fix is the `targetType` facet, not a custom FE (B1085 §1085.8).
- `FIELD_EDITOR` facet (B1086 §1086.5) is the right override when you cannot change the slot type — e.g. a `BString` slot needing a specialized editor in one specific context. For `BOrd` slots pointing at station components, prefer `targetType` facet over `FIELD_EDITOR` (no custom Java required).
- `BWbFieldEditor.dialog()` (B1086 §1086.3) is the correct plumbing for any WB code that needs to edit a value in a standalone modal — used by bacnet-wb's address-editing dialogs; pairs with the PD-WMF deltas (wb-manager-framework-deltas).
- Remittances confirmed: `BFlexAddressFE` compound editor (B1074) is the advanced variant of the Δ3 recipe; `BWbComponentView` (B1057/B1058/B1070) is the "view" twin, used when you need a full panel rather than a field-editor widget.

---
**Status**: PENDING — 3 proposed deltas; full detail in corpus B1084–B1087.
