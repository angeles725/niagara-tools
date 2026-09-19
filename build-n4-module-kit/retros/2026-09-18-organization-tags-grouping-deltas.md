<!-- review-status: folded -->
# 2026-09-18 · kit · organization-tags-grouping-deltas

**Session**: corpus-mining for build-n4-module (operator: generate all possible kit deltas — organization/tags/grouping).
**Delta count**: 6

## What happened

Mined the organization/tags/grouping vein of the niagara-research corpus against the current
kit state. Source blocks: B749 (Honeywell block-organization taxonomy, 10 patterns), B750
(taxonomy applied to our modules), B758 (tags/relations/northbound), B747 (Wire Sheet as flow
surface + SUMMARY pin mechanic), B748 (interactivity/low-cognitive-load ranked playbook), B781
(grouping/relating author-side declaration surfaces).

**Already folded — skipped as duplicates:**

- B749 P2 / B737 composition into child components → `types/logic.md §Composition & organization`
  (one-bullet mention; cites B737/B749/B750).
- B749 P4 typed folders with `isChildLegal`/`isParentLegal` → `types/logic-authoring.md` line 21
  (BFolder, isChildLegal/isParentLegal, no BComponentList — cites B779).
- B781 three-posture grouping declaration (categories=nothing, relations=RelationInfo not
  BRelation subclass, hierarchy=BHierarchy+BLevelDef) → `types/logic-authoring.md §Grouping and
  relating declaration surfaces` (three-bullet section).
- B782 query/search PROVIDER surface (BQuery/NEQL + BIAgent/BQueryEngine/BColumnsProvider/
  BISearchProvider/BSystemIndexer → BITable) → `types/logic-authoring.md §Query/search/index
  surface`.
- B747 SUMMARY = wire-sheet port (one-liner in flags bullet, corpus-index.md P2 entry) →
  `types/logic.md §Tridium rt idioms` flags bullet + `corpus-index.md` P2 row (B747).
- Tag dictionary basic concept → `types/logic.md §Ship a tag dictionary` (cites B814; uses
  `getRules()/SmartTagRule[]` API — present but shallow; extended by Δ1/Δ2 below).
- B749 P3 config/state separation → `types/logic.md §Composition & organization` (one-liner:
  "tunables in a frozen `config` child, value + BStatus on the component" — brief but present;
  sharpening not proposed since the one-liner is sufficient for the pattern; no BStruct wire-map
  needed because our outputs are BLinks to driver proxy points, not field-register addresses).
- B749 P6 pre-wired palette assembly templates → folded via `module-palette-and-build-target`
  retro + B746 entry in corpus-index.md.

**Genuine gaps found — 6 proposed deltas:**

1. **B758 §758.1 constructor-seed API** — the existing kit `§Ship a tag dictionary` section
   uses the Haystack/B814 `getRules()/SmartTagRule[]` pattern. B758 documents an OLDER alternative
   used by Honeywell: `tagInfoList.add(SlotPath.escape(name), new BSimpleTagInfo(BMarker.DEFAULT))`
   in the constructor, guarded by `get(..)==null` for idempotency. These are two DISTINCT APIs
   for two different extension mechanisms of `BSmartTagDictionary`; a builder choosing the
   constructor approach needs the exact call shape.

2. **B758 §758.1 rule-based auto-tagging** — `BTagRule` + `BTagRuleCondition` let the dictionary
   classify components automatically without per-instance annotation. `BIsPointProxyTypeRule`
   + `BIsPointProxyTypeCondition` classify a driver point by its proxyExt TYPE. `BEquipmentTypeTag`
   maps a point-folder's display name to an equip-type tag via a `lookupTable` facet. The
   `getImpliedTag`/`addAllImpliedTags` engine in `BSmartTagDictionary` drives this. NOT in kit.

3. **B758 §758.2 BCustomRelation full mechanics** — the kit says "define a relation type by
   registering a RelationInfo/BCustomRelation in a tag dictionary" (one line, cites B781). Missing:
   (a) how to define `BCustomRelation` (source scope `entity|station`, target `BTypeSpec`, inbound/
   outbound relation-id facet maps), (b) runtime resolution by ORD/NEQL query → cursor →
   `new BasicRelation(id, component, inbound?)` per hit, (c) CONSUMER API:
   `entity.relations().get(Id, dir)` / `getAll(Id, dir)` — each `Relation`'s endpoint is the far
   component. A builder implementing a Haystack `equipRef`-style edge needs this recipe.

4. **B758 §758.4 BQL consumer cursor pattern** — the kit covers the PROVIDER side (BQueryEngine/
   BColumnsProvider/BISearchProvider etc.) but NOT the CONSUMER side: the Java cursor loop that
   a dashboard servlet or service method uses to run a BQL query from code:
   ```java
   BITable<?> t = (BITable<?>) BOrd.make("station:|slot:/|bql:select … from <module:Type> where …")
                                   .get(Sys.getStation());
   TableCursor<?> c = t.cursor();
   while (c.next()) { … c.cell(column) … }
   c.close();
   ```
   `from <module:Type>` is the select-by-station-type mechanism; oBIX exposes the same as
   `/obix/bql/<query>`. NOT in kit.

5. **B748 §748.2 ranked UX/legibility checklist** — the six ranked changes that fix "desborde" in
   the wire sheet and property sheet (ordered by impact/cost: SUMMARY pin curation, child
   component composition, units/precision facets, distinct icons per type, pre-wired palette
   assembly templates, semantic tags) are NOT in the kit as a checklist. The corpus-index-rt
   retro (folded) proposed `types/organization.md` for this; neither that file nor the ranked
   list itself was created. The ordering matters: items 1 (SUMMARY curation) and 3 (facets) are
   same-day flag/facet passes; 5 (palette template) is a resource add; 2 (composition) is the
   structural one; 4 and 6 are polish/discoverability.

6. **B750 §750.2 / B758 §758.5 our-modules tag namespace recipe** — the kit's tag-dictionary
   section is generic. The corpus gives the CONCRETE recipe for our modules: namespace `angeles`;
   marker tags `room`, `evaporator`, `compressor`, `defrost`; one `BTagRule` keyed on component
   TYPE (not folder name, since our components are well-typed). Place the dictionary instance
   under `TagDictionaryService` in `serviceStarted()` (auto-install). After this, NEQL
   `station:|slot:/|neql:select * where tag::room` works with zero integrator effort. NOT in kit.

## Evidence

- B758 §758.1: `tagInfoList.add(SlotPath.escape(name), new BSimpleTagInfo(BMarker.DEFAULT))`;
  guard `get(...)==null`; `BIsPointProxyTypeRule`+`BIsPointProxyTypeCondition.test = entity
  instanceof BControlPoint && proxyExt.getType().is(proxyExtType)`;
  `BSmartTagDictionary:101-167` drives `getImpliedTag` `[ev: corpus B758 §758.1]`
- B758 §758.2: `BCustomRelation.java:59-198`; `BOrd.make(query).get()→BITable→cursor → new
  BasicRelation(id, component, inbound?)`; `entity.relations().get(Id, dir)`;
  `BContainmentRelation:88,108,133` `[ev: corpus B758 §758.2]`
- B758 §758.4: `BAnalyticService.java:1663-1685`; `BBqlLobbyAgent` at `/obix/bql/`
  `[ev: corpus B758 §758.4]`
- B748 §748.2: six-row ranked table (SUMMARY curation ÷ trivial, composition ÷ medium, facets ÷
  low, icons ÷ low, palette templates ÷ low, tags ÷ medium); sequencing note (1+3 same-day, 5
  resource, 2 structural, 4+6 polish) `[ev: corpus B748 §748.2]`
- B750 §750.2 from P9 + B758 §758.5: namespace `angeles`; markers room/evaporator/compressor/
  defrost; additive overlay, not nesting; `serviceStarted()` auto-install
  `[ev: corpus B750 §750.2]` `[ev: corpus B758 §758.5]`
- Kit current state — `types/logic.md §Ship a tag dictionary` (B814 `getRules()/SmartTagRule[]`
  present; constructor-seed/rule-based pattern ABSENT); `types/logic-authoring.md §Grouping
  and relating declaration surfaces` (one-liner on Relations; BCustomRelation mechanics ABSENT);
  `types/logic-authoring.md §Query/search/index surface` (PROVIDER side only; consumer cursor
  ABSENT); `types/organization.md` does NOT exist

## Proposed kit deltas (propose-never-apply)

| Δ | Delta | Target file / § | Token |
|---|---|---|---|
| Δ1 | Add constructor-seed API variant under `§Ship a tag dictionary`: `subclass BSmartTagDictionary + set namespace + in ctor: tagInfoList.add(SlotPath.escape(name), new BSimpleTagInfo(BMarker.DEFAULT))` per marker tag, guarded by `get(..)==null` (idempotent re-runs). Contrast with the existing B814 `getRules()/SmartTagRule[]` approach: both extend `BSmartTagDictionary`; the tagInfoList path is the Honeywell "seed in ctor" pattern; the getRules path is the Haystack discovery pattern. | `types/logic.md §Ship a tag dictionary` sub-bullet | `[ev: corpus B758 §758.1]` |
| Δ2 | Add rule-based auto-tagging sub-bullet: `BTagRule` + `BTagRuleCondition` let the dictionary classify components by type/condition without per-instance tags. The `getImpliedTag`/`addAllImpliedTags` engine in `BSmartTagDictionary` iterates `getTagRules()` and applies matching rules. `BEquipmentTypeTag` maps a folder's display name to an equip-type tag via a `lookupTable` facet. Add a worked shape: implement `BTagRule.test(entity)` → true/false, return tag names/values from `getTags()`. | `types/logic.md §Ship a tag dictionary` sub-bullet | `[ev: corpus B758 §758.1]` |
| Δ3 | Expand the one-line Relations posture in `§Grouping and relating declaration surfaces` with the full `BCustomRelation` recipe: declare a `BCustomRelation extends BRelationInfo` with source scope (`entity`/`station`), target `BTypeSpec`, inbound/outbound relation-id facet maps; override `addRelations(entity, cx)` to resolve targets via an ORD/NEQL query (`BOrd.make(query).get(ctx) → BITable → cursor → emit new BasicRelation(id, component, inbound)` per hit); consume with `entity.relations().get(Id, dir)` / `getAll(Id, dir)` — each `Relation`'s endpoint is the far component. Register the custom relation in the tag dictionary's constructor alongside the tag defs. | `types/logic-authoring.md §Grouping and relating declaration surfaces` | `[ev: corpus B758 §758.2]` |
| Δ4 | Add `§BQL from code — consumer cursor pattern` (a self-contained memorizable recipe): `BITable<?> t = (BITable<?>) BOrd.make("station:|slot:/|bql:select … from <module:Type> where …").get(Sys.getStation()); TableCursor<?> c = t.cursor(); while (c.next()) { … c.cell(column) … } c.close();` — note `from <module:Type>` is the select-by-registered-type mechanism; always close the cursor. oBIX exposes the same query as `/obix/bql/<query>`. Contrast with the PROVIDER surface in `§Query/search/index surface` (implementing `BQueryEngine`/`BColumnsProvider`): the cursor pattern here is the CALLER side used by a servlet or service method that runs a BQL query. | `types/logic-authoring.md` new §BQL from code — consumer cursor | `[ev: corpus B758 §758.4]` |
| Δ5 | Add ranked UX/legibility checklist (or create `types/organization.md` as proposed in corpus-index-rt retro): six changes ordered by impact/cost — (1) SUMMARY pin curation (trivial · flag-only: real I/O = SUMMARY, internals = non-summary, engine callbacks = HIDDEN; declutters the wire sheet immediately), (2) compose flat slots into child components (medium · structural), (3) units/precision facets on every temp/pressure/percent slot (low · facet-only), (4) distinct icon per block type (low · one SVG resource), (5) pre-wired palette assembly templates (low · resource-only), (6) semantic tag dictionary (medium). Sequencing note: 1+3 are same-day; 5 is a resource add; 2 is the structural one (one careful pass per module); 4+6 are polish. Add this as an expansion of `§Composition & organization` or a new file depending on length. | `types/logic.md §Composition & organization` expansion OR new `types/organization.md` | `[ev: corpus B748 §748.2]` |
| Δ6 | Add our-modules application sub-section under `§Ship a tag dictionary`: namespace `angeles`; marker tags: `room`, `evaporator`, `compressor`, `defrost`; one `BTagRule` keyed on component TYPE (e.g. `entity instanceof BEvaporatorUnit`); place the dictionary under `TagDictionaryService` in `serviceStarted()` (auto-install, not drag-on-commissioning). After this, NEQL `station:|slot:/|neql:select * where tag::room` reaches all cold room components with zero integrator effort. Add a note: this overlay is additive and deploy-safe (no schema change, no containment change). | `types/logic.md §Ship a tag dictionary` our-modules sub-section | `[ev: corpus B750 §750.2]` `[ev: corpus B758 §758.5]` |

## Lessons

- **B758 carries TWO distinct `BSmartTagDictionary` extension points** (constructor-seed vs
  `getRules()`/`SmartTagRule[]`); the kit already folded the B814/getRules variant. Mining the
  Honeywell pattern (B758) surfaced the older tagInfoList constructor approach — both are valid;
  one being in the kit doesn't mean the other is covered.
- **The CONSUMER side of BQL is absent from the kit even though the PROVIDER side is present.**
  A builder implementing a dashboard servlet that queries by type will reach for `BOrd.make()`
  and find no recipe. PROVIDER and CONSUMER are symmetric concerns; index them together.
- **B748's ranked impact÷cost ordering is the missing bridge between "what to do" (B749/B750)
  and "where to start" (the kit)**. Without that ranking, a builder who reads the Honeywell
  patterns knows five changes but not which to tackle first on a live module. The ordering is
  load-bearing guidance, not decoration.
- **Corpus-index-rt retro proposed `types/organization.md` (folded 2026-09-03) but the file
  was never created.** B748's ranked checklist is the natural content for that file.

---

**Status**: PENDING — INDEX row appended:
`| 2026-09-18-organization-tags-grouping-deltas.md | kit | 2026-09-18 | pending | 6 |`
