# Retro (PROPOSED delta — propose-never-apply) — a curated corpus index for build-n4-module: the RT-authoring + organization blocks the skill should take into account

- **Date**: 2026-09-03
- **Origin**: a niagara-research `/research-sdd` run that reconstructed the RT block model, the Wire Sheet, the Honeywell module-organization taxonomy, the WB/UX authoring taxonomy, and the module-authoring axes — versioning/upgrade-safety, bits, build/signing, integration, tags, lexicon (corpus blocks **B729–B760**). Operator asked: survey the documentation and tell build-n4-module which docs serve it, which blocks, what logic to take into account.
- **Update 2026-09-04a**: extended to cover the WB/UX authoring campaign (**B751–B753**) — feed `types/wb-widgets.md` [seed] and `types/dashboard.md` [mature].
- **Update 2026-09-04b**: extended to cover the module-authoring campaign (**B754–B760**) — versioning/upgrade-safety, the bit models, the build/signing toolchain, integration, tags/exposure, lexicon/doc, and the consolidated actionable audit. Added to the index table and the rules below; the versioning-safety rules belong in `METHODOLOGY.md`, and B760 is the actionable punch-list any of our-module work should start from.
- **Status**: PROPOSED. This retro adds no rule and edits no methodology; it proposes a curated block index + specific wiring so a human can accept it. Following the skill's own step 6 (append PROVEN lessons as PROPOSED deltas).

---

## Finding

The kit already tells the builder "**corpus-nav FIRST**" and references a handful of blocks (B166, B179, B180, B194, B202, B203, B213, B636, B724, **B729, B730**). But the niagara-research corpus now carries a COMPLETE authoring body — the RT campaign **B729–B746**, the Wire-Sheet + organization work **B747–B750**, the WB/UX authoring taxonomy **B751–B753**, and the module-authoring axes **B754–B760** (versioning/upgrade-safety, bits, build/signing, integration, tags, lexicon, and a consolidated actionable audit) — and only **B729/B730** of it are wired into the kit. That is the single largest block of directly-relevant documentation the skill is not yet pointing the builder at. The WB/UX blocks matter especially for the two type guides the kit marks weakest: `types/wb-widgets.md` (seed) and `types/dashboard.md` (mature).

"corpus-nav FIRST" is necessary but not sufficient: lexical search finds a block only if the builder already knows the term. A CURATED INDEX (block → what it gives → when to read it) turns the corpus from "searchable if you know the word" into "the reading list for building a module."

Several of these lessons ALREADY have kit retros (so they are partly covered): B729 → `self-firing-timer-needs-started-*`; B739 → `slot-type-change-rompe-bog-*`; B741 → `qa-stack-pure-tests-*`; B746 → `module-palette-and-build-target`; the B740 cross-module-enum lesson is already in `types/logic.md` ("Linking across custom modules"). The gap is the rest, and the lack of one place that maps them.

## Proposed deliverable: `$KIT/corpus-index.md` (a new file), referenced from SKILL step 2 and `types/logic.md`

A curated map. Priority = how load-bearing it is for a correct build (P0 = read before building, P1 = read for the relevant layer, P2 = read when the feature/decision comes up).

| Block | What it gives the builder | Feeds kit file | Priority | Already wired? |
|---|---|---|---|---|
| **B744** | The ONE consolidated "what an RT block is" index (parts, format, rules, what it shows/does/discovers) — the entry point that links every other block | `types/logic.md` (start-here) | **P0** | no |
| **B737** | Engine thread + watchdog, and **composition**: group concerns into CHILD components instead of a flat slot wall (the fix for our 25-slot `BEvaporatorUnit`) | `types/logic.md` (new "Composition" section) | **P0** | no |
| **B730** | The rt idioms catalog (execute/changed discipline, timers, flags, degrade-honestly) | `types/logic.md` | **P0** | **yes** |
| **B729** | Timer lifecycle: arm in `started()` + `atSteadyState()`, `clockChanged` | `types/logic.md` | **P0** | **yes** (retro) |
| **B739** | Schema evolution — **ADD, never retype** an existing slot (retype breaks the `.bog`, station won't boot). Outage-prevention | `METHODOLOGY.md` (schema check) | **P0** | **yes** (retro) |
| **B740** | Cross-module links use a plain `double`, never a shared frozen enum (`Missing class …HoaMode` outage) | `types/logic.md` | **P0** | **yes** (in logic.md) |
| **B734** | Point-type taxonomy (4×2, point vs writable/priority array, proxy vs local, extensions) — decide component vs point | `types/logic.md` | P1 | no |
| **B735** | Slots/facets/links mechanics + **why SUMMARY/HIDDEN/BIUnlinkable curate the Link picker and the wire sheet** | `types/logic.md` + `METHODOLOGY.md` (flags) | P1 | partial |
| **B736** | The `BStatus` 8-bit model — set/propagate fault/down/stale/null/overridden honestly | `METHODOLOGY.md` (status check) | P1 | partial |
| **B745** | Units — `BUnit`/`UnitDatabase`, the `units` facet, how to put °C/kPa/% on a slot | `METHODOLOGY.md` (facets) | P1 | partial |
| **B738** | Practical how-to: add a proxyExt, facets, `propagateFlags`, an icon/SVG to a block | `types/logic.md` / how-to | P1 | no |
| **B746** | Module palette authoring (BOG XML) + **pre-wired assembly templates** so commissioning is drag-one-thing | `types/*` + palette retro | P1 | **yes** (retro) |
| **B749** | **The Honeywell block-organization taxonomy — the 10 recurring patterns** (see "logic to take into account" below) | new `types/organization.md` or `corpus-index.md` | **P0** | no |
| **B750** | The taxonomy applied to OUR modules — 5 actionable gaps + deploy-safe sequence | `types/organization.md` | P1 | no |
| **B747** | The Wire Sheet as a flow surface: **pin = SUMMARY exactly**; live values render with facets + status color | `types/logic.md` (pins) | P2 | no |
| **B748** | Interactivity/low-cognitive-load playbook (curate pins, compose, icons, palette, tags) | `types/organization.md` | P2 | no |
| **B732** | Authoring real alarms — `BAlarmSourceExt` is a point extension; the offnormal/fault algorithm family | `types/logic.md` (when adding alarms) | P2 | no |
| **B733** | Modulating (0-10V) outputs, `kitControl.BLoopPoint` PID, the math block family | `types/logic.md` (when adding PID/AO) | P2 | no |
| **B741** | The 4-layer QA/test stack; what's testable in WSL | `build-verify.md` | P1 | **yes** (retro) |
| **B743** | Testing timer-arming/lifecycle — the scheduler seam + `BTestNgStation` | `build-verify.md` | P2 | no |
| **B751** | **WB authoring: the "how much wb is enough" ladder** (rung 0 nothing → 1 FieldEditor → 2 Manager → 3 custom View) + the Manager/View/FieldEditor/Command recipes + the Honeywell device-model plugin | `types/wb-widgets.md` (seed → grow it) | **P0 for a -wb build** | no |
| **B752** | **UX authoring: the three serving recipes** (servlet-SPA / bajaux @AgentOn view / PX), the two bajaux data-channel dialects, PX bindings, and the RBAC contrast | `types/dashboard.md` (mature) | **P0 for a -ux build** | no |
| **B753** | The WB/UX playbook applied to OUR modules — our components sit at wb rung 0; keep the servlet-SPA + `OPERATOR_WRITE` RBAC | `types/wb-widgets.md` + `types/dashboard.md` | P1 | no |
| **B754** | **Module versioning + the saved-data survival matrix** — what schema changes are SAFE / LOSSY / OUTAGE over an existing `.bog` (generalizes B739); no per-module migration hook | `METHODOLOGY.md` (schema/upgrade check) | **P0 for any schema change** | no |
| **B755** | **The bit models you set** — slot `Flags`, `BStatus`, `BPermissions`, `BVersion` with exact values (SUMMARY=8, HIDDEN=4, OPERATOR=256, OPERATOR_WRITE=2, …) | `METHODOLOGY.md` (flags) + `types/logic.md` | P1 | partial (flags in METHODOLOGY) |
| **B756** | **Build + version-targeting + signing** — vendor stamp, target-the-lowest-station, plugin family, .jar vs .dist, project-CA + STORED repack | `build-verify.md` | **P0 for build/deploy** | **yes** (build-verify.md is the operational form) |
| **B757** | Station integration — authoring a `BAbstractService` (register-by-placement) + the nav tree (any component is a nav node free) | `types/logic.md` (when adding a Service) | P2 | no |
| **B758** | Tags/relations authoring + northbound data exposure (tag dictionary, oBIX agent, Fox/BOX, BQL-from-code cursor) | `types/dashboard.md` (when exposing data) | P2 | no |
| **B759** | Lexicon/i18n + the -doc/help profile — `module.lexicon` key=type/slot, `toFriendly` fallback; **CompPan-rt lexicon is EMPTY** | `METHODOLOGY.md` (lexicon check) | P1 | partial |
| **B760** | **The consolidated actionable audit** — what's already correct + the 8-item ranked punch-list for our modules + the versioning discipline. START HERE for our-module work | reference (the punch-list) | **P0 for our-module work** | no |
| **B731** / **B742** | Our own modules' audit + the consolidated deploy-safe refactor backlog | reference | P2 | no |

## The LOGIC the skill must take into account (the non-obvious rules)

These are the rules a builder gets wrong without the docs — the "qué lógica" the operator asked for. Most are the distilled Honeywell patterns (B749) confirmed against our own outages:

1. **Compose into child components; don't sprawl flat slots.** One child `BComponent` per concern/domain (timing, outputs, hoa, freeze). This is B737 AND exactly how every Honeywell module distributes (B749 P2). Above ~12–15 slots, flat is wrong.
2. **Separate config from live-state.** Put tunables in a frozen `config` child; keep value + `BStatus` on the component (Honeywell `honIOBase` pattern, B749 P3 / B750). Our field outputs are BLinks to proxy points, so we skip Honeywell's third "wire-map `BStruct`" plane.
3. **SUMMARY = the wire-sheet pin, exactly** (`SlotBarGlyph.java:56 Flags.isSummary`). Curate pins: SUMMARY on real I/O, HIDDEN on internals, non-summary for interim state (B735/B747). This is what keeps a block from "desbordando."
4. **ADD, never retype a slot with saved data** (B739) — retype breaks the `.bog`, the station won't boot. A real outage. The single most important schema rule.
5. **Cross-module value = plain `double`, never a shared enum type** (B740) — a shared enum forces a hard inter-module dependency and left a `Missing class …HoaMode` on the live JACE.
6. **A self-armed timer arms in `started()`, not only `atSteadyState()`** (B729) — else it never fires on a late mount (the defrost-never-armed bug).
7. **Alarms NOTIFY, they never STOP control** — do not wire alarm-limit slots into the control decision (already in `types/logic.md`; grounded in B732).
8. **Ship pre-wired palette assembly templates** (B746, B749 P6) — Honeywell's whole Venom/IRM reuse layer is palette-only BOG trees. Bakes in correct nesting + flags (avoids the `hasDefrost=false → never defrosts` trap).
9. **Tag components for BQL discoverability** as a semantic overlay, not by nesting (B749 P9).
10. **Grouping folders are TYPED and self-validate** via `isParentLegal`/`isChildLegal` (B749 P4) — the tree rejects a wrong child.

### WB/UX rules (from B751–B753)

11. **Author the LEAST wb the ladder allows** (B751): rung 0 = nothing (the default property/wire-sheet views render standard slots — kitControl ships 152 rt types with only 2 wb FEs). Climb only when needed: rung 1 a `BWbFieldEditor @AgentOn(value type)` for ONE composite value that renders badly; rung 2 a `BAbstractManager`/`BDeviceManager` ONLY if the component is a container of learned/discovered children; rung 3 a `BWbComponentView` ONLY for non-tabular interaction. **Our ColdRoomPan/CompPan components sit at rung 0 — do not build a Manager.**
12. **FieldEditor recipe** (B751): ctor builds widgets → `linkTo(widget, textModified, setModified)` → override `doLoadValue`/`doSaveValue`/`doSetReadonly`; child editors via `BWbFieldEditor.makeFor(value)`. Register with `@AgentOn(that value type)`.
13. **A Honeywell "Wizard" is usually a tabbed `BWbComponentView`, not a `BWizard`** (B751) — step-panes = tabs, backed by rt `BJob`s launched from an agent `BMenu`. Mutation always through the space (`newTransaction`/`tx.commit`); undo is inherited, never hand-rolled.
14. **Pick the right UX serving recipe** (B752): a bespoke dashboard + custom JSON API → **servlet-SPA** (`BWebServlet` self-registers by `getServletName()`); a view bound to a component type → **bajaux `@AgentOn` + `BIJavaScript`** (`getJsInfo` → `module://…/rc/X.js`); engineer-authored equipment graphics → **PX** (XML bound to slot ords, no Java). Our DashboardPan is servlet-SPA — the correct choice; do not migrate to bajaux.
15. **Enforce REAL server RBAC — the vendors don't** (B752): gate every write on `BPermissions.OPERATOR_WRITE` (the permission BIT, not role-name matching), fail-closed (no user → 401, lacks-write → 403), as the FIRST line of the handler. The Honeywell React SPAs ship `permissions="unrestricted"` RPCs with no server check — do NOT copy that. Add a CSRF `X-Requested-With` gate + an audit trail (our chihuahua/DashboardPan pattern).
16. **Servlet-SPA footguns** (B752): keep routing in a PURE `route()` function (Niagara-free, WSL-unit-testable); serve statics from `rc/` with a traversal guard; **never set a user Home Page to a servlet path** (a raw path is not a resolvable ORD → `SyntaxException: Missing scheme name` → every login fails — bookmark the URL instead); don't hardcode the service ORD if it may be relocated.

### Versioning / upgrade-safety + bits rules (from B754–B760)

17. **The saved-data survival matrix — the single most important upgrade rule** (B754): a `.bog` binds to a class BY NAME, ungated (no version, no type-hash). A schema change either routes to `warningAndSkip` (survivable, data silently dropped/shunted) or an unwrapped throw (station won't boot). **SAFE**: ADD a slot, REORDER, change default/flags/facets, ADD an enum tag. **LOSSY-SAFE** (boots, data lost): REMOVE/RENAME a slot, complex-retype. **OUTAGE (won't boot)**: RETYPE a frozen SIMPLE slot whose saved `v=` can't parse into the new primitive (the B739 case), or REMOVE/RENAME a frozen ENUM TAG some `.bog` stored. So: **ADD, never retype/remove**; never remove/rename an enum tag; if a slot's shape must change, ADD-new + migrate-in-`started()` + leave-old-deprecated (there is NO decoder convert hook and NO per-module migration callback).
18. **Bump `vendorVersion` + back up `config.bog` on every schema-change deploy** (B754/B756): nothing gates on `vendorVersion` at decode, but it is the audit trail and the dependency minimum; and survivable changes still DROP data silently (only a printed warning count) — "it booted" ≠ "the data survived".
19. **Target the LOWEST station you must support** (B756): a 4.14 JACE REJECTS a jar whose manifest stamps `baja 4.15`. `gradle.properties` can lie; verify the stamped `baja` with `--target-version`. Re-sign JACE-bound jars under the project CA (`angelessigner`) via a STORED repack (the WSL deflater mismatch).
20. **Know the bit models you set** (B755): `Flags` — SUMMARY=8 (pin), HIDDEN=4 (all UI), READONLY=1, TRANSIENT=2 (NOT persisted — don't put it on config you want saved), OPERATOR=256, DEFAULT_ON_CLONE=64, FAN_IN=1024. `BStatus` — DISABLED=1/FAULT=2/DOWN=4/ALARM=8/STALE=16/OVERRIDDEN=32/NULL=64/UNACKED=128, ok=0. `BPermissions` — gate writes on the `OPERATOR_WRITE=2` BIT, never a role name.
21. **Fill every module's `module.lexicon`** (B759): key = bare type name and bare slot name (module-global, so shared slot names collide); a missing key silently renders the raw camelCase via `toFriendly`. **CompPan-rt's lexicon is currently EMPTY** — every compressor slot shows raw camelCase. This is the cheapest legibility fix.
22. **Author the LEAST integration surface** (B757): our components need no custom Service/nav code — a `BComponent` is a nav node for free, and a Service registers by being placed under `/Services`. Add tags/oBIX/relations (B758) only when discoverability or a standards-based northbound client actually needs them.

## Proposed deltas (propose-never-apply — for human review)

1. **Create `$KIT/corpus-index.md`** = the table above (the curated reading list), and add one line to SKILL step 2 and to `types/logic.md`: "Before building, skim `corpus-index.md` — the P0 blocks (B744, B737, B730, B729, B739, B740, B749) are the reading list."
2. **Add a "Composition & organization" section to `types/logic.md`** pointing at B737 + B749/B750 (the file today is flat-slot oriented; it teaches idioms but not the tree shape). Include the one-line rule: distribute by containment with fixed roles; one child per domain; config separate from state.
3. **Optionally add `$KIT/types/organization.md`** if the organization body (B749/B750) grows past a section — the 10 patterns + the applied playbook are a distinct concern from per-layer idioms.
4. **Add to `METHODOLOGY.md`** two check items already grounded in P0 blocks: (a) "Compose concerns into child components above ~12–15 slots (B737)"; (b) the schema-safety line "ADD slots, never retype one with saved data (B739)" (currently only a retro).
5. **Grow the two weakest type guides from the WB/UX blocks**: fold B751 (the "how much wb" ladder + FieldEditor/Manager/View recipes) into `types/wb-widgets.md` (today a seed), and B752 (the three serving recipes + the RBAC rule) + B753 (our servlet-SPA is the chosen recipe) into `types/dashboard.md` (today mature but pre-dating the RBAC census). Add the "author the least wb the ladder allows" one-liner to SKILL step 3.
6. **Add the versioning/upgrade-safety section to `METHODOLOGY.md`** (from B754/B756): the SAFE/LOSSY/OUTAGE matrix as a pre-deploy check ("ADD, never retype/remove a frozen slot with saved data; never remove/rename an enum tag; bump vendorVersion; back up config.bog"), plus the "target the lowest station" build rule (B756 — `build-verify.md` already carries the mechanics; cross-link it).
7. **Add a lexicon check to `METHODOLOGY.md`** (from B759): "every exported type/slot has a `module.lexicon` key or it renders raw camelCase" — and flag that **`CompPan-rt/module.lexicon` is empty** as a concrete open item.
8. **Point SKILL step 2 at B760 for our-module work**: "when extending ColdRoomPan/CompPan/DashboardPan, start from B760's punch-list (what's already correct + the ranked fixes)."

No file above is edited by this retro — these are proposals. B739/B729/B741/B746 lessons already have retros; this consolidates the map and adds the missing composition/organization/wb/ux/versioning/reference blocks.

## Evidence
- Kit inspected: `SKILL.md`, `METHODOLOGY.md`, `types/logic.md`, `types/wb-widgets.md`, `types/dashboard.md`, `build-verify.md`, `retros/` (2026-09-03/04). Existing block refs: B166/B179/B180/B194/B202/B203/B213/B636/B724/B729/B730.
- Corpus blocks surveyed: B729–B760 (niagara-research), read/cited this session; organization taxonomy from a 5-sweep Honeywell census (B749); WB/UX authoring from a 4-sweep census (B751–B753); module-authoring (versioning/bits/build/integration/tags/lexicon) from a 6-sweep census (B754–B760).
- Cross-checked which lessons already have kit retros to avoid duplicate proposals.
