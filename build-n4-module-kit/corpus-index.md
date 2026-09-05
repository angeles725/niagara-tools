# Corpus index — the curated map of the niagara-research authoring corpus (B729–B760)

The niagara-research corpus carries a COMPLETE N4 module-authoring body — the RT campaign (B729–B746),
the Wire-Sheet + organization work (B747–B750), the WB/UX authoring taxonomy (B751–B753), and the
module-authoring axes (B754–B760: versioning/upgrade-safety, bits, build/signing, integration, tags,
lexicon, and a consolidated audit). This index points the builder at the block that answers each need.

**How to use it:** `corpus-nav FIRST` when you have a TERM to look up (`corpus-nav find "<topic>"`); use THIS
index when you want to know WHAT TO READ for the layer you are building, in priority order. Blocks live in
the niagara-research repo, not this kit — read them there.

**Priority:** **P0** = read before building · **P1** = read for the layer you are touching · **P2** = read when
that feature/decision comes up.

**Start here:** rt logic → **B744** (the "what an RT block is" index) · our-module work → **B760** (the ranked
actionable punch-list) · `-wb` build → **B751** (the wb ladder) · `-ux` build → **B752** (the serving recipes).

## P0 — read before building

| Block | What it gives the builder | Layer |
|---|---|---|
| **B756** | Build + version-targeting + signing — vendor stamp, target-the-lowest-station, plugin family, `.jar` vs `.dist`, project-CA + STORED repack | build |
| **B760** | The consolidated actionable audit — what's already correct + the ranked punch-list for our modules + the versioning discipline; START HERE for our-module work | reference |
| **B749** | The Honeywell block-organization taxonomy — the 10 recurring patterns (compose into children, curate pins, icons, palette, tags) | organization |
| **B744** | The ONE consolidated "what an RT block is" index (parts, format, rules, what it shows/does/discovers) — the entry point that links every other rt block | rt |
| **B737** | Engine thread + watchdog, and composition: group concerns into CHILD components instead of a flat slot wall (the fix for a 25-slot unit) | rt |
| **B730** | The rt idioms catalog (execute/changed discipline, timers, flags, degrade-honestly) | rt |
| **B729** | Timer lifecycle: arm in `started()` + `atSteadyState()`, `clockChanged` | rt |
| **B740** | Cross-module links use a plain `double`, never a shared frozen enum (a `Missing class …HoaMode` outage) | rt |
| **B739** | Schema evolution — ADD, never retype an existing slot (retype breaks the `.bog`, station won't boot) | schema |
| **B754** | Module versioning + the saved-data survival matrix — which schema changes are SAFE / LOSSY / OUTAGE over an existing `.bog` (generalizes B739); no per-module migration hook | schema |
| **B751** | WB authoring: the "how much wb is enough" ladder (rung 0 nothing → 1 FieldEditor → 2 Manager → 3 custom View) + the Manager/View/FieldEditor/Command recipes | wb |
| **B752** | UX authoring: the three serving recipes (servlet-SPA / bajaux `@AgentOn` view / PX), the bajaux data-channel dialects, PX bindings, and the RBAC contrast | ux |

## P1 — read for the layer you are touching

| Block | What it gives the builder | Layer |
|---|---|---|
| **B741** | The 4-layer QA/test stack; what's testable in WSL | build / test |
| **B734** | Point-type taxonomy (4×2, point vs writable/priority array, proxy vs local, extensions) — decide component vs point | rt |
| **B735** | Slots/facets/links mechanics + why `SUMMARY`/`HIDDEN`/`BIUnlinkable` curate the Link picker and the wire sheet | rt / flags |
| **B738** | Practical how-to: add a `proxyExt`, facets, `propagateFlags`, an icon/SVG to a block | rt / how-to |
| **B755** | The bit models you set — slot `Flags`, `BStatus`, `BPermissions`, `BVersion` with exact values (`SUMMARY=8`, `HIDDEN=4`, `OPERATOR=256`, `OPERATOR_WRITE=2`, …) | flags |
| **B736** | The `BStatus` 8-bit model — set/propagate fault/down/stale/null/overridden honestly | status |
| **B745** | Units — `BUnit`/`UnitDatabase`, the `units` facet, how to put °C/kPa/% on a slot | facets |
| **B759** | Lexicon/i18n + the `-doc`/help profile — `module.lexicon` key=type/slot, `toFriendly` fallback | lexicon |
| **B746** | Module palette authoring (BOG XML) + pre-wired assembly templates so commissioning is drag-one-thing | palette |
| **B750** | The organization taxonomy applied to OUR modules — actionable gaps + a deploy-safe sequence | organization |
| **B753** | The WB/UX playbook applied to OUR modules — our components sit at wb rung 0; keep the servlet-SPA + `OPERATOR_WRITE` RBAC | wb / ux |

## P2 — read when that feature/decision comes up

| Block | What it gives the builder | Layer |
|---|---|---|
| **B743** | Testing timer-arming/lifecycle — the scheduler seam + `BTestNgStation` | build / test |
| **B747** | The Wire Sheet as a flow surface: pin = `SUMMARY` exactly; live values render with facets + status color | wire-sheet |
| **B748** | Interactivity / low-cognitive-load playbook (curate pins, compose, icons, palette, tags) | organization |
| **B757** | Station integration — authoring a `BAbstractService` (register-by-placement) + the nav tree | rt / service |
| **B732** | Authoring real alarms — `BAlarmSourceExt` is a point extension; the offnormal/fault algorithm family | rt / alarms |
| **B733** | Modulating (0-10V) outputs, `kitControl.BLoopPoint` PID, the math block family | rt / control |

## Research-tooling caveats (A18)

Before concluding a topic is undocumented: a tool returning zero results is not proof of absence — run a control query or fall back to `rg`/`mem_context`/the source (S1 [ev: retro rt-authoring-campaign Δ4]). Never mark a claim `[CERT]` from a mangled decompiled body — prefer the vineflower/procyon tree or mark `[INFER]` (S2 [ev: retro rt-authoring-campaign Δ3]).
| **B758** | Tags/relations authoring + northbound data exposure (tag dictionary, oBIX agent, Fox/BOX, BQL-from-code cursor) | ux / data |
| **B731** / **B742** | Our own modules' audit + the consolidated deploy-safe refactor backlog | reference |

---

## Campaign 6 — exemplars + own-modules focus (B762–B791)

Added by the research fold (PR7). Idiom note: B778 + B782 + B785 are three instances of ONE Niagara extension idiom — *subclass a framework base + register a `<type>`/agent in `module.xml` + hand back a self-describing SPI object; there is no central registry the author edits.* Teach it once; the per-surface blocks are instances. See `types/logic-authoring.md` §Author-side SPIs.

### → `types/logic-authoring.md`

| Block | What it gives the builder | Priority | Layer |
|---|---|---|---|
| **B790** | The MINIMAL correct N4 module skeleton (copy-start): `module.xml` header + 3-part dep floor + non-empty palette + prefixed lexicon + one `BComponent` with one OPERATOR property + one HIDDEN engine action + one `Clock.Ticket` (started/atSteadyState arm, stopped cancel) + signed jar — verify-gate-green + biting-check-green by construction | P0 | rt / scaffold |
| **B772** | Authoring a point EXTENSION: extend `javax.baja.control.BPointExtension` (NO `BAbstractPointExt`, NO `onExtended`/`onRetracted`); implement the sole abstract `onExecute(BStatusValue out, Context cx)` (mutate `out`, or leave it for notification-only); `requiresPointSubscription()`→true to see every change; `final getParentPoint()`; legality via `isParentLegal`/`isSiblingLegal`; execution = slot-declaration order, proxyExt always first | P2 | rt / extensions |
| **B779** | Child-tree containers, pick by cardinality: frozen `@NiagaraProperty` (fixed) / runtime `add(name,BValue)` + `reorder(Property[])` (data-driven) / typed `BFolder` (growable). NO `BComponentList`; typed-tree legality via `isChildLegal`/`isParentLegal` `instanceof` vetoes | P1 | rt / children |
| **B778** | Author-side SPIs: a custom SERVICE (`extends BAbstractService`, `getServiceTypes()` = registration-by-placement, `serviceStarted()`); a new ORD SCHEME (`extends BOrdScheme` + `@NiagaraType(ordScheme)` + `resolve()`); a SERVER-side subscription (`extends Subscriber`, `event(BComponentEvent)`) | P2 | rt / service |
| **B781** | Grouping/relating author postures (three, distinct): categories = NO author scaffold (runtime-only); relations = never subclass `BRelation`, define a type via `RelationInfo`/`BCustomRelation`; hierarchy = compose `BHierarchy`+`BLevelDef` under `BHierarchyService` | P2 | rt / relations |
| **B782** | Build a query/search/index surface with ONE recipe: a typed `BQuery`/NEQL payload + the matching `BIAgent` provider (`BQueryEngine`/`BColumnsProvider`/`BISearchProvider`/`BSystemIndexer`) discovered by the agent registry → a `BITable` | P2 | rt / data |
| **B783** | Template author path = ARTIFACT production, not type registration: a "template type" is an `.ntpl` ZIP made by a job from a `BTemplateConfig`-marked subtree — do NOT scaffold a `BTemplate` subclass (no SPI) | P2 | rt / template |
| **B785** | Extend a framework via a Device + a self-describing SPI object — the rdb dialect exemplar: `B<X>Database extends BRdbms` + 3 abstract methods; `getRdbmsContext()` → a `RdbmsDialect`; register a manifest `<type>`; no central registry | P2 | rt / data |
| **B777** | SECURITY-MODULE skeleton: `BAbstractService` → `BAuthenticationScheme` subclass wired via `getLoginConfiguration`; permissions INLINE in `module.xml` `<permissions>` (NOT `module-permissions.xml`); jar-signed `NIAGARA4.RSA/SF` is MANDATORY; register `@AgentOn "baja:AuthenticationScheme"` | P2 | rt / security |
| **B776** | ACTION PROTECTION: gate declaratively — `@NiagaraAction(flags=Flags.OPERATOR=256)` → operator-invoke, OMIT → admin-invoke (DEFAULT), enforced by `BComponent.canInvoke` + fox/box `PermissionException`; `doPrivileged` = JVM permission ONLY (never wrap a Niagara RBAC check) | P2 | rt / security |
| **B775** | Authoring a WATCHDOG/monitor + choosing a timer: subclass `BAbstractAlarmMonitor` (override `doRunCheck`, `raiseAlarm` edge-latch); timer = `Clock.schedule` (one-shot) vs `Clock.schedulePeriodically` + keep the `Clock.Ticket` (NO `javax.baja.sys.BTimer` — cl.hvac only); cadence is a configurable `BIntervalTriggerMode` (15-min default), not a 2s poll; native `EngineWatchdog` is a separate layer | P2 | rt / watchdog+timer |
| **B774** | Authoring a background JOB: subclass `BSimpleJob` + `run(Context)` for the normal async case; submit via `BJobService.submit(job,cx)` → an ORD handle (poll, no join); multi-step = `BJobStep` under a `BBatchJob` | P2 | rt / jobs |
| **B773** | Authoring an analytics compute NODE: `@NiagaraType` subclass of `javax.bajax.analytics.algorithm.BOutputBlock`; inputs = `BBlockPin` `@NiagaraProperty` + `BLink` DAG edges; register by `module.xml <type>` (NO @AgentOn — EXCEPTION to the idiom) | P2 | rt / analytics |

### → `types/dashboard.md`

| Block | What it gives the builder | Priority | Layer |
|---|---|---|---|
| **B780** | `module.palette` + `module.lexicon` copy-ready conventions: palette `<p n= t= m=>` (bare-Type-minus-`B` names, plural folders, `m="alias=module"` once, nested pre-seed); lexicon is flat/module-global → PREFIX keys (`parent.child`) to dodge the B759 collision | P1 | palette / lexicon |
| **B763** | The `-ux` servlet WRITE-surface, five gates: OPERATOR_WRITE fail-closed → hand-rolled `X-Requested-With` guard IN the pure `route()` → `SERVICE_ORD` pinning/allowlist → per-Ord lock + HTTP 423 → audit. Plus the pure RBAC test seam (`canWrite(boolean)`) | P1 | ux / security |
| **B762** | Off-station testing seams for `-ux`: the pure `route()`→`RouteAction` seam, the purity gradient (inject Baja as a `Function`), and the SPA-JS limit (`node --check` = syntax only) | P1 | ux / test |
| **B791** | Web-tier exemplars audit: servlet routing → B29; hx → B433; `module://` rc → B5/B752; `@AgentOn` web agent → B752/B421; Tridium CSRF → B58; plus DashboardPan divergences (CSRF via hand-rolled X-Requested-With = DELIBERATE, STRONGER-than-vendor; RBAC seam COLLAPSED = punch-list) | P2 | ux / web-tier |

### → `types/wb-widgets.md`

| Block | What it gives the builder | Priority | Layer |
|---|---|---|---|
| **B762** | Off-station testing of `-wb`: the `wb/model/` lambda-injection seam — a Baja-free `model/` package with the slot check injected as a `Predicate<String>`; the `BWidget` view stays the adapter | P1 | wb / test |
| **B780** | Dual-surface `@AgentOn` registration: write `@NiagaraType(agent={@AgentOn(types={"mod:Type"},requiredPermissions="r")})` on the view/FE; Slot-o-Matic emits `<type><agent><on/></agent></type>`; multi-type = one view over several source types | P1 | wb / agent |

### → `METHODOLOGY.md`

| Block | What it gives the builder | Priority | Layer |
|---|---|---|---|
| **B784** | Real `module.xml` conventions: profile split `-rt`/`-ux`/`-wb`/`-se` (server), `-doc` is a SEPARATE `runtimeProfile="doc"` module; `<dependency>` `vendorVersion` = 3-part Tridium FLOOR (`4.14.0`) vs the module's own 4-part build stamp (`4.14.0.162`); header attribute roster | P1 | build / module.xml |
| **B787/B788/B789** | Conformance rules: lintable (statically decidable: lexicon dup-keys, Clock.Ticket without stopped-cancel, empty palette, coverage-%) vs advisory (human-review: action operator-vs-admin intent, container order-sensitivity, poll-vs-subscribe) | P2 | build / verify |

---

Maintenance: this index is a curated pointer, not a copy — when a block's content is folded into a kit guide
(METHODOLOGY, BUILD-LOOP, `types/*.md`, build-verify), that guide carries the rule and cites the block; this
index stays the "what to read" map. Source: retro `corpus-index-rt-authoring-and-organization-blocks`.
