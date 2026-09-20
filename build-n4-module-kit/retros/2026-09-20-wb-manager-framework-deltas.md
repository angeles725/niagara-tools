<!-- review-status: folded -->
# 2026-09-20 · kit · wb-manager-framework-deltas

**Session**: niagara-research focus `wb-manager-framework` (B1088–B1093, commit e4121d38d) — the `BAbstractManager`/`MgrController`/`MgrModel`/`MgrColumn`/`MgrLearn`/`MgrEdit` framework that every driver device/point manager extends, and how our hand-built `BWbComponentView`+`BTable` managers compare.

**Delta count**: 4

## What happened
Reconstructed the WB manager framework end-to-end: the 6 support objects, the cell-edit→slot pathway (`BMgrEditDialog.ColumnInput`), the display/permission gate (`BMgrTable.reload`), the discovery/learn contract (`MgrLearn`), and the column taxonomy (`MgrColumn`). Our kit names "use a Manager" (MBP3) but does not document the wiring, so developers read source. Proposed deltas add the missing reference material and a service-centric template.

## Evidence
- B1088 (BAbstractManager anatomy + isLearnable/isTaggable/isTemplatable gates), B1089 (MgrColumn taxonomy), B1091 (BAbstractManager vs BWbComponentView decision), B1092 (BMgrTable + ColumnInput cell-edit pathway; `BAbstractManagerFE` does NOT exist — FE suffix is a driver naming convention), B1093 (MgrSupport/MgrLearn contract; `MgrLearn.cols[]` separate from `MgrModel.cols[]`). `[ev: corpus B1088-B1093 / commit e4121d38d]`

## Proposed kit deltas (propose-never-apply)
| Δ | Delta | Target file / § | Token |
|---|---|---|---|
| Δ1 | PD-WMF1 add a "Manager recipe" section to WB best-practices: the `BAbstractManager` 6-object anatomy + the 3 boolean gates (`isLearnable`/`isTaggable`/`isTemplatable`) that drive pane visibility | `docs/module-best-practices.md` MBP3 / `types/wb-widgets.md` | `[ev: corpus B1088]` |
| Δ2 | PD-WMF2 add a `MgrColumn` taxonomy reference card (Name/Type/Prop/PropPath, when to use each: flat property vs nested slot vs mixin) | `docs/module-best-practices.md` MBP3 / `types/wb-widgets.md` | `[ev: corpus B1089]` |
| Δ3 | PD-WMF3 add a "`BAbstractManager` vs `BWbComponentView`" decision table — when to use the driver-manager framework vs a hand-built view+BTable (our Apillm managers are the latter) | `docs/module-best-practices.md` MBP3 | `[ev: corpus B1091]` |
| Δ4 | PD-WMF4 add a minimal non-driver custom-manager template (`@AgentOn` a service/container, `MgrColumn.Prop` only, no discover) to the how-to guide's `-wb` section; the envCtrlDriver example (B956) is driver-centric | `docs/how-to-create-an-n4-module.md` §-wb | `[ev: corpus B1091]` |

## Lessons
- `BAbstractManagerFE` is NOT a class — the "FE" suffix in driver managers is a naming convention; the real cell-edit bridge is `BMgrEditDialog.ColumnInput` (B1092).
- The discover table (`MgrLearn.cols`) and database table (`MgrModel.cols`) have INDEPENDENT column sets (B1093).
- For a simple component container (like our Apillm importer/exporter), a hand-built `BWbComponentView`+`BTable` is legitimate; the full `BAbstractManager` framework is worth it when you need discovery/learn + rich columns (B1091). Pairs with wb-vendor Δ19 (multi-tab view) and PD-FE deltas.

---
**Status**: PENDING — 4 proposed deltas; full detail in corpus B1088–B1093.
