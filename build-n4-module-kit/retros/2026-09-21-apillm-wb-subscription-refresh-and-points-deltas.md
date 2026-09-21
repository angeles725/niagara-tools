<!-- review-status: pending -->
# 2026-09-21 · kit · apillm-wb-subscription-refresh-and-points-deltas

**Session**: Apillm client module (Cliente/LLM/Apillm) — live commissioning of the rt+wb N4↔DB bidirectional gateway on station LLM. A user-visible WB manager bug (stale table until re-open) was root-caused live, fixed, and deepened in the research corpus (B1140–B1142).
**Delta count**: 5

## What happened
The `BApillmManager` / `BApillmImporterManager` views (`BWbComponentView` + `TableModel`, hand-built — not the `BAbstractManager` framework) showed their rows with UNSUBSCRIBED default slot values (Target=null, Status=unresolved, Recursive=no) until the view was re-navigated. Root cause: the model was built once in `doLoadValue()` with no refresh hook, and the subject descendants load asynchronously. This is the empirical confirmation of the already-folded Δ17 rule (`types/wb-widgets.md:253`), but Δ17 named only a candidate lint and no concrete API. Fixing it live (register/refresh) surfaced the exact WB API, a subscribe-depth rule, and a distinct second gap: table columns that render the live value/status of an ORD OUTSIDE the subject subtree stay unresolved even after the fix. Separately, closing the `points-rt-wb` research focus produced two citable reference blocks (rt read path, wb create) that belong in the kit's authoring guides.

## Evidence
- WB refresh API confirmed in Tridium docSource: `registerForComponentEvents(BComponent, int depth)` + override `handleComponentEvent(BComponentEvent)`; `BWbView.activated()`/`deactivated()`; `unregisterForAllComponentEvents()`. NOT a raw `javax.baja.sys.Subscriber` (that is the RT pattern). `[ev: corpus B1140]`
- `BAbstractManager` opts OUT of auto-subscribe (`autoRegisterForComponentEvents=false`) and drives refresh via `BMgrTable.reload()` → `subscribe(rows)` at `model.getSubscribeDepth()`; `BWbComponentView` default `autoRegisterForComponentEvents=true` subscribes only the subject at depth 0 and its `handleComponentEvent()` is a no-op. `[ev: corpus B1140 — BAbstractManager.java:131, BMgrTable.java:226-263, BWbComponentView.java:192-194,303]`
- Depth matters by nesting: servlet→exports FOLDER→export needs `registerForComponentEvents(exportsFolder, 2)`; importer→importMap (direct child) needs depth 1. `[ev: Apillm BApillmManager/BApillmImporterManager fix 2026-09-21]`
- External-ORD gap: `BApillmImporterManager` cols "Last Value | Status" resolve `ApillmImportMap.targetOrd` → an EXTERNAL point (`slot:/Folder1/NumericWritable`) outside the importer subtree, so `registerForComponentEvents(importer,1)` never subscribes it; those columns stay stale. Same shape for point-exports in `BApillmManager`. `[ev: Apillm-wb/BApillmImporterManager.java cols; corpus B1140-G1]`
- rt point READ path: `BControlPoint.getOutStatusValue()`; typed `BStatusNumeric.getValue()` etc.; `BStatus.isValid()` = not(DISABLED|FAULT|DOWN|STALE|NULL) vs `isOk()` = bits==0 (ALARM is valid but not ok); `BOrd.get()` throws `UnresolvedException` on dangling. `[ev: corpus B1141]`
- wb CREATE point: `container.add(uniqueName, point, null)` (direct); `MgrController.promptForNew` → `MgrEdit.commit()` (Transaction) (dialog); `BEnumRange.make(tags[])` ordinals=indices. `[ev: corpus B1142]`

## Proposed kit deltas (propose-never-apply)
| Δ | Delta | Target file / § | Token |
|---|---|---|---|
| Δ1 | Give the `-wb` refresh rule its CONCRETE API and tighten the `lint-wb-refresh` candidate: the fix is `registerForComponentEvents(subject, depth)` in `doLoadValue()` + override `handleComponentEvent(BComponentEvent)` (rebuild+relayout) + `activated()` (rebuild) + `deactivated()`→`unregisterForAllComponentEvents()`. NOT a raw `sys.Subscriber`. Lint should look for `registerForComponentEvents`+`handleComponentEvent`, not a "Subscriber field". | `types/wb-widgets.md` §"view refresh rules"; `toolbelt/lint-wb-refresh.sh` (candidate) | `[ev: corpus B1140]` |
| Δ2 | Document the subscribe-DEPTH rule: depth = distance from the subscribed root to the slots the table renders. Folder-nested children need depth≥2 (servlet→exports-folder→export = 2); a direct child = 1. Pairs with `getDeviceManagerSubscribeDepth` (Δ20/driver-authoring). | `types/wb-widgets.md` §"view refresh rules" | `[ev: corpus B1140]` |
| Δ3 | NEW rule + lint candidate `lint-wb-external-ord-value`: a manager column that shows the live value/status of an ORD OUTSIDE the subject subtree must ALSO lease/subscribe each RESOLVED target (`loadSlots()` for the one-shot read + `registerForComponentEvents(target,0)` for later refresh); subject-subtree depth alone leaves it unresolved. | `types/wb-widgets.md` §"view refresh rules"; new `toolbelt/lint-wb-external-ord-value.sh` | `[ev: corpus B1140-G1; Apillm BApillmImporterManager]` |
| Δ4 | Add an rt control-point READ-path reference card: `getOutStatusValue()`/typed `getValue()`; `BStatus.isValid()` vs `isOk()` (ALARM is valid, not ok); `BOrd.get()`→`UnresolvedException` on dangling; `BDynamicEnum` ordinal/tag/displayTag. | `types/logic-authoring.md` (new §point-read) | `[ev: corpus B1141]` |
| Δ5 | Add a wb CREATE-point recipe: direct `container.add(uniqueName, point, null)`; dialog `MgrController.promptForNew`→`MgrEdit.commit()` (Transaction); `BEnumRange.make(tags[])` ordinals=indices / `make(ordinals,tags)` / `make(frozen,ordinals,tags)`; `ObjectUtil.uniqueName` for slot names. | `types/wb-widgets.md` (new §create-point-from-wb) | `[ev: corpus B1142]` |

## Lessons
- The WB refresh rule (Δ17) was folded WITHOUT its concrete API and without a working lint — a rule that names a symptom but not the exact call gets "fixed" wrong. Fold the API with the rule.
- A hand-built `BWbComponentView`+`TableModel` manager owns BOTH its subject-subtree subscription AND any external ORD it renders; these are two separate subscriptions, easy to miss the second.
- `registerForComponentEvents`/`handleComponentEvent`/`activated`/`deactivated` is the WB-client refresh seam; `sys.Subscriber` is RT-only. Do not cross them.
- Closing a research focus at `gaps_remaining:0` without a named child-gap section over-declares "complete" — name what is left (this session's wb-manager-framework close missed the subscription mechanism entirely).

---
**Status**: PENDING — INDEX row appended: `| 2026-09-21-apillm-wb-subscription-refresh-and-points-deltas.md | kit | 2026-09-21 | pending | 5 |`
