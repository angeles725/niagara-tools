# Corpus index — the curated map of the niagara-research corpus for module builders (B5–B1462)

The niagara-research corpus carries a COMPLETE N4 module-authoring body — the RT campaign (B729–B746),
the Wire-Sheet + organization work (B747–B750), the WB/UX authoring taxonomy (B751–B753), and the
module-authoring axes (B754–B760: versioning/upgrade-safety, bits, build/signing, integration, tags,
lexicon, and a consolidated audit). Later campaigns extended it with exemplar, own-module, distribution and
platform blocks, the N4 Developer course focus and the running-station topics (B5–B1420), so the title names the full range it cites. This index points the builder at the
block that answers each need. `[ev: retro kit-meta-hygiene-2026-10-01 Δ2]`

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
| **`docs/how-to-create-an-n4-module.md`** | **START HERE** — master end-to-end guide: scaffold → deps → slots → sign → test → deploy. Written from first principles; every step traces to a B-block. Lives in `niagara-research/docs/`. | reference |
| **B817** | The N4 module STRUCTURE STANDARD — how Tridium/Honeywell lay out rt/ux/wb/doc profiles; a lintable conformance checklist for our four modules | organization |
| **B960** | The canonical N4 module build scaffold from the first-party SDK examples — the Gradle layout, module.xml header, signing wiring, and Slotomatic hook as Tridium ships them | build / scaffold |

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
| **`docs/module-dev-workflow.md`** | Module dev runbook — the edit→build→sign→deploy→test loop with exact tool steps; every step cites the relevant B-block. Lives in `niagara-research/docs/`. | reference |
| **`docs/module-best-practices.md`** | Evidence-grounded best-practices guide — rules distilled from reference modules and corpus audits (B705+). Lives in `niagara-research/docs/`. | reference |
| **B961** | SDK dev-examples SYNTHESIS — what the four SDK examples collectively teach + the deltas for our module work; read after B960 | reference |
| **B955** | envCtrlDriver SDK example (rt) — a complete minimal N4 field driver, first-party source; the rt SPI contract in a real build | rt / exemplar |
| **B956** | envCtrlDriver SDK example (wb) — the Workbench device/point manager UI for the SDK driver exemplar | wb / exemplar |
| **B877** | NDriver chassis vs DDF: the two generic driver bases, their comm-layer contracts and the transport matrix — read before choosing a driver framework (`types/driver-authoring.md` § 1.8) | driver |
| **B1421** | DR5 station-test evidence: `BTestNgStation` needs its default Fox service, the WSL user-home lock and symlink-mirror walls, the Context-form discovery dispatch, and why a test run is judged by its counts — read before running driver station tests (`types/driver-authoring.md` § 12 / § 12.1) | driver / test |
| **B957** | typeExtensionDemo SDK example (ux) — a browser type-extension with JS build + Jasmine tests; the ux SPI patterns in a real build | ux / exemplar |
| **B1015** | Building `UmbrellaDashboard` — a dashboard-type N4 module (facade + servlet + 3D SPA), rt+ux built green @4.14; the DashboardPan exemplar in a second module | reference / dashboard |

## P2 — read when that feature/decision comes up

| Block | What it gives the builder | Layer |
|---|---|---|
| **B743** | Testing timer-arming/lifecycle — the scheduler seam + `BTestNgStation` | build / test |
| **B747** | The Wire Sheet as a flow surface: pin = `SUMMARY` exactly; live values render with facets + status color | wire-sheet |
| **B748** | Interactivity / low-cognitive-load playbook (curate pins, compose, icons, palette, tags) | organization |
| **B757** | Station integration — authoring a `BAbstractService` (register-by-placement) + the nav tree | rt / service |
| **B732** | Authoring real alarms — `BAlarmSourceExt` is a point extension; the offnormal/fault algorithm family | rt / alarms |
| **B733** | Modulating (0-10V) outputs, `kitControl.BLoopPoint` PID, the math block family | rt / control |
| **B958** | componentLinks SDK example — link-lifecycle callbacks + the `BTestNg` moduleTest framework; the correct seam for station-level tests (WSL-unsafe, documents the boundary) | test |
| **B867** | Server-side TYPE-LEVEL component subscription (`BComponentSpace.subscribe(Type[], TypeSubscriber)`) — the SPI for observing every live instance of a type from a service | rt / service |
| **B869** | Directional vs bidirectional communication in N4 — links, writeback, and "who respects whom"; pick the correct direction before wiring a facade→rt slot | rt |
| **B871** | The `converters` type-bridge graph — 104 converter classes over ~17 core value types; read when two heterogeneous slot types need to wire across a link | rt / build |
| **B892** | `driverUpgrade` — the WB per-driver component-upgrade SPI (`upgradeClass`, BOG rewrite); relevant when providing a WB migration path on a schema change | wb / build |
| **B894** | `dashboard-wb` — the multi-target render agents for a `DashboardPane` (`@AgentOn` HX-bootstrap + PDF-paint via one model) | wb |
| **B895** | `uxBuilder` — rendering Workbench Px as browser bajaux widgets (embed a JS widget in Px by ORD; the agent-filter gate + the Px-serving servlet) | ux |
| **B900** | The `BComponentEvent` id catalog — 21 named component-event ids (0–20) + 4 reserved slots; read before writing a `Subscriber.event()` handler | rt |
| **B33** | The history system — history extensions, collection and the batch editor; read before adding trend/history to a module's points | history |
| **B34** | The alarm framework — alarm classes, recipients, the alarm database format; read before routing a module's alarms to the station alarm console | alarms |
| **B39** | Provisioning, station backup, supervisor replication, operational HA and fleet management; read before a deploy that needs a backup or a fleet rollout | backup / fleet |
| **B48** | RBAC visibility in the frontend — the server is the truth, client-side gating is UX only; read before hiding a control by role in a dashboard | ux / security |

## Research-tooling caveats (A18)

Before concluding a topic is undocumented: a tool returning zero results is not proof of absence — run a control query or fall back to `rg`/`mem_context`/the source (S1 [ev: retro research-sdd-rt-authoring-campaign-retro #4]). Never mark a claim `[CERT]` from a mangled decompiled body — prefer the vineflower/procyon tree or mark `[INFER]` (S2 [ev: retro research-sdd-rt-authoring-campaign-retro #3]).
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
| **B777** | SECURITY-MODULE skeleton: `BAbstractService` → `BAuthenticationScheme` subclass wired via `getLoginConfiguration`; permissions INLINE in `module.xml` `<permissions>` (NOT `module-permissions.xml`); jar signing is MANDATORY (any signer alias: `<ALIAS>.SF` + its signature block, `NIAGARA4` by default); register `@AgentOn "baja:AuthenticationScheme"` | P2 | rt / security |
| **B776** | ACTION PROTECTION: gate declaratively — `@NiagaraAction(flags=Flags.OPERATOR=256)` → operator-invoke, OMIT → admin-invoke (DEFAULT), enforced by `BComponent.canInvoke` + fox/box `PermissionException`; `doPrivileged` = JVM permission ONLY (never wrap a Niagara RBAC check) | P2 | rt / security |
| **B775** | Authoring a WATCHDOG/monitor + choosing a timer: subclass `BAbstractAlarmMonitor` (override `doRunCheck`, `raiseAlarm` edge-latch); timer = `Clock.schedule` (one-shot) vs `Clock.schedulePeriodically` + keep the `Clock.Ticket` (NO `javax.baja.sys.BTimer` — cl.hvac only); cadence is a configurable `BIntervalTriggerMode` (15-min default), not a 2s poll; native `EngineWatchdog` is a separate layer | P2 | rt / watchdog+timer |
| **B774** | Authoring a background JOB: subclass `BSimpleJob` + `run(Context)` for the normal async case; submit via `job.submit(cx)` (= `BJobService.getService().submit(job, cx)`) → an ORD handle (poll, no join); multi-step = `BJobStep` under a `BBatchJob` | P2 | rt / jobs |
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

## P2 — distribution (read when shipping a module or targeting a specific N4 release)

| Block | What it gives the builder | Layer |
|---|---|---|
| **B1020** | N4 Distribution ND1 — the Installer Contract (`install.properties`, `version.properties`, EULA/licensing): what the customer's installer reads when deploying your module | distribution |
| **B1021** | N4 Distribution ND2 — shipped default config (`nre.properties`, `system.properties`, `units.xml`) vs installed values; how OEM defaults layer over stock N4 (refines B31) | distribution |
| **B1022** | N4 Distribution ND3 — Honeywell OEM overlay anatomy over stock 4.13.2.18 (branding, versioning, overlay extras); understand what an OEM bundle adds vs the stock install | distribution |
| **B1023** | N4 Distribution ND4 — the dist/ payload tree: JRE bundles, framework versions, Supervisor-as-installer mode | distribution |
| **B1024** | N4 Distribution ND5 — install-time transfer & migration (`conversion/`, AX→N4 path, `cleanDist`, station templates) | distribution |
| **B1025** | N4 Distribution ND6 — the shipped modules set: 721 jars, the 4.13.2.18.5 manifest, OEM-vs-stock families (vs the installed N4.14 corpus) | distribution |
| **B1026** | N4 Distribution ND7 — Installer mechanics: `Installer_x64.exe`, overlay-merge sequence, `niagarad`/Service setup | distribution |
| **B1027** | N4 Distribution ND8 — SDK dev examples (`dev/`) and default security material (`overlay/security/`) — file layout and what is shipped | distribution |

## n4-dev-course — the N4 Developer Certification Course (B1230–B1258, B1300–B1354, recordings B1416–B1420)

The Tridium-authored course (N4.15.6.20) digested topic by topic: 29 blocks, a 73-rule checklist (B1242 §1242.3)
and a ranked "what not to copy" list (B1242 §1242.4). Course files are example code only: mirror them first,
apply the §1242.4 fixes only as named opt-in changes (`BUILD-LOOP.md` § 0, `SOURCES.md` § Exemplar source).
`[ev: retro n4-course-as-exemplar-source Δ1]` The recordings (B1416–B1419) are spoken: read a claim through B1420,
which checked the open ones against code. `[ev: retro course-coverage Δ18]`

| Block | What it gives the builder | Layer |
|---|---|---|
| **B1242** | **P0** — the course SYNTHESIS: ordered learning path, the Tridium rules-of-thumb checklist, what not to copy (ranked), 4.15 vs 4.14 deltas, proposed kit deltas; START HERE for course work | reference |
| **B1238** | **P0** — multithreaded development and Jobs: the station thread map, which thread runs a sync / ASYNC / scheduled action, the Lab 9 executors, the `BSimpleJob` lifecycle (deck 9, Labs 9–10) | rt / threading |
| **B1235** | **P0** — Gradle and permissions: the 4.12→4.14 build DSL ladder, the plugin pin, `module-permissions.xml` groups → Java permissions | build / permissions |
| **B1236** | **P0** — logging, lexicons and testing: the DebugService logging model, the `<Type>.displayName` lexicon key, `BTestNg`/`BTestNgStation` + Mockito | lexicon / test |
| **B1237** | **P0** — ORDs and BQL: the ORD query chain, `BOrd.resolve`/`get`, a custom `BOrdScheme`, the BQL grammar against `bql-rt` code | ord / bql |
| **B1239** | **P0** — drivers / NDriver: the NMessage/NComm/ILinkLayer model, the Lab 11 driver, and the lab solution's latent defects | driver |
| **B1230** | The course map (decks 0–10 + bonus, labs 1–11 + bonus) and the N4.15.6.20 developer environment | reference |
| **B1300** | The developer command set the N4.15 course uses (gradlew and signing tasks, `test.exe`, the `-watch`/`settestargs` loop, `moduleVerificationMode`), each mapped to its code or block, 4.14 vs 4.15 | build / test |
| **B1302** | Study map: each course deck joined to the official devguide/guide pages, the blocks that deepen it, and where the concept lives (or is missing) in our modules | reference |
| **B1354** | Verified course exemplars: the live-tested 4.15 certification exam (`BLimiter`, `BHistoryExportService`, a minimal TCP-ping NDriver) and the Review solution, read against the course's own code | exemplar |
| **B1416** | Recordings Day 1: lifecycle as spoken vs `EngineManager` (links propagate before `started()`), `disabled()` vs `stopped()`, `started()` vs `stationStarted()`, dev-environment recipes | rt / lifecycle |
| **B1417** | Recordings Day 2: default code signing (`niagara.signing.jceks`), the license-gated security-manager developer mode, modules / types / Slot-o-Matic rules | build / signing |
| **B1418** | Recordings Day 3: `getHtmlSafeText`, logging and lexicon rules, BQL case rules, the 60 s lease and its stale first read, build recovery | lexicon / ord / test |
| **B1419** | Recordings Day 4: `getServiceTypes()`, alarms / NEQL / SystemDB, engine watchdog defaults, threads, driver idioms (stale time, NDriver) | rt / alarm / driver |
| **B1420** | Child gaps of B1416-B1419 worked to code: stale read after `lease()`, COV sensitivity per driver, alarm UUID on return-to-normal, `orientSystemDb` + relation IDs, shutdown thread, Niagara Console environment | rt / driver / reference |
| **`docs/n4-module-development-course-guide.md`** | How to develop an N4 module as Tridium teaches it (N4.15), every rule linked to its block (B1230–B1258). Lives in `niagara-research/docs/`. | reference |

## Debugging, services, licensing, removal, web tooling and classic UI (B5–B1297)

Blocks the type cards cite for the running-station and authoring topics added on 2026-11-15
(`types/diagnostics.md` § 4, `types/service.md`, `types/security.md` § 5.2-5.3, `types/distribution.md` § 10a,
`types/wb-widgets.md`, `types/theme.md` § 8-10). Each was opened and its subject confirmed before it was listed.

| Block | What it gives the builder | Layer |
|---|---|---|
| **B1296** | The station console: the eight documented verbs and five hidden ones (`threads`, `save`, `quit`, `kill`, `syncmodules`, …), how Application Director and `plat tellstation` deliver them through the daemon, station exit codes; code of 4.14.0.162 | diagnostics |
| **B1297** | Launcher command lines from code: `nre`, `station`, `wb`, `niagarad`, `test` flags (documented and not), the shared `-name:value` grammar, the command-line property blacklist | diagnostics / build |
| **B1249** | Course child gaps closed from code; §1249.1 is the launcher's `-@<jvm option>` pass-through and the `developer`-licence gate on Java agents (what a remote debugger needs) | diagnostics |
| **B1257** | Expected results of the course labs as screenshots show them: §1257.7 the debugger attach (Appendix A), §1257.10 the rules they teach (signing, permissions, logging) | reference / test |
| **B1191** | Station save: the rotated `config_backup_<time>.bog` rollback file and backup count, the save job; the spy page table with the state-changing, CSRF-gated pages | persistence / diagnostics |
| **B1247** | Course runtime gaps from code; §1247.11: the `spy:` scheme is superuser-only, no per-component spy permission | security / diagnostics |
| **B1240** | Course bonus topics: authoring a spy page (`spy(SpyWriter)`, always call `super`), a `-doc` help part (`toc.xml`, no custom styles), BajaDocs; the two Review Assignments decoded | observability / doc |
| **B20** | Overview of `BAbstractService` internals (callbacks, bootstrap phases, fault states), `BApp`, system monitors, `BJobService`; an early overview: prefer B757/B778/B1145 for exact code | rt / service |
| **B1145** | Service start order in `ServiceManager.startService`: licence check, then `serviceStarted()` with no guard between them; a throw is logged and the boot continues; the symmetric stop path | rt / service |
| **B1143** | `BAbstractService` licence enforcement: the start-time presence gate sets a FATAL fault but stops nothing, limits fail open when the check failed, the fault cannot be edited away | service / licensing |
| **B1144** | Licence validation chain: host id compare, vendor certificate signed by a built-in key, licence file signed by the vendor, load-time failures that only log, the exception taxonomy, node-locked vs subscription | licensing |
| **B1139** | Recovering a station stuck at boot with the platform Software Manager (station down): statuses, Downgrade / Uninstall / Re-Install / Rebuild Module Signatures, Commit; never File Transfer Client | platform / recovery |
| **B633** | Daemon-side module install mechanics: signature gate by the target's verification mode, every station stopped, jar overwritten in place with no backup or rollback | platform / distribution |
| **B5** | ORD syntax and resolution, the BOG file format (type references `t=` and module bindings `m=`), queries; an early broad block | ord / persistence |
| **B29** | The web tier end to end: Jetty embedding, servlet registry, filter chain (auth, CSRF), endpoint matrix, sessions, caching of JS bundles | ux / web |
| **B38** | File system and `BOrd` resolution end to end; §38.22 notes the module-dev switch that reads module resources from the source tree instead of the jar | ord / web |
| **B1132** | The `-ux` JavaScript build toolchain: grunt-niagara, the Gradle `niagara-grunt` plugin (`babel:dist`, `copy:dist`, `requirejs`), RequireJS `nmodule/<module>/rc/…` IDs | ux / build |
| **B1119** | ES6 vs ES5 for `-ux` JavaScript: transpiled by the grunt build, mandatory ES5 for raw `rc/` files | ux / build |
| **B204** | The bajaux framework: `Widget` lifecycle (`doInitialize`, `doLoad`, `doRead`, `doSave`), properties and validators, spandrel, and the rt→web bridge (`BIJavaScript`, `JsInfo`, `BJsBuild`, the widget servlet) | ux |
| **B433** | The Hx framework from code: `BHxView` is a servlet view, buffered HTML wrapped by the profile with its CSRF token, header-keyed events, stateless requests, 5 s polling, legacy status | wb / hx |
| **B427** | The Workbench widget model: `BWidget` is a `BComponent`, layout by `doLayout`, painting through the `gx` `Graphics` interface, event dispatch, the theme families | wb |
| **B1086** | Field-editor extras: `BIPopupEditor`, `BWbFieldEditor.dialog()`, the `fieldEditor` facet override, `IDialogContentProvider`, the profile filter | wb |


## Cited by the guides — every other block a kit guide names in an `[ev: corpus …]` tag

The sections above rank blocks by when to read them. This one lists the remaining blocks the guides cite, so a
citation always resolves to a line here: the second column is the block's title in the corpus catalog
(`cnav list`, 2026-11-23, shortened), the third the guide(s) that cite it, the place to read what the block
proves. `tests/corpus-index.bats` (CI-ALL) fails when a guide cites a block this index does not hold.
`[ev: retro doc-coverage-audit Δ20]`

| Block | What the corpus catalog says it is | Cited in |
|---|---|---|
| **B4** | Baja Object Model | `types/logic-authoring.md`, `types/logic.md` +1 |
| **B12** | Build System + Module Development Lifecycle | `METHODOLOGY.md` |
| **B14** | Point counting + License limits + Templates + Batch Editor + EasyTemplates | `types/driver-authoring.md` |
| **B15** | Workbench editing deep (wiresheet + property sheet + nav + point/device manager) | `types/logic.md` |
| **B18** | Module signing standalone + module-permissions + CSRF + Header Auth | `build-verify.md`, `types/distribution.md` +2 |
| **B26** | NRE launcher C++ + DLLs nativas + standalone module signing ops | `types/ml-libraries.md`, `types/third-party-libraries.md` |
| **B28** | Discovery cross-protocol + Virtual Components | `types/driver-authoring.md` |
| **B35** | Nav + Workbench shell + views registry | `types/logic-authoring.md`, `types/module-wiring.md` +1 |
| **B36** | PX widgets deep + EasyTemplates + BajaScript BOX wire + browser lifecycle | `types/dashboard.md` |
| **B46** | Writes con priority array desde SPA externa | `types/dashboard.md` |
| **B67** | Analytics deep dive para construir módulo MX60: librerías + 56 algorithm blocks + 17 combiners + 22 trend wrappers + Psychrometric HVAC + Degree-Day + veredictos PID/ML/Forecasting/Anomaly … | `types/math-control-libraries.md`, `types/ml-libraries.md` +1 |
| **B92** | `lonhoneywellAXWizards`: suite de wizards Workbench para dispositivos LON Honeywell Excel 10 / XL15C / T7350, sobre el driver `lonworks` estándar de Tridium, deofuscado | `types/observability.md` |
| **B106** | `honeywellSpyderTool` / `XL10NextGen`: la herramienta de programación del Spyder (3er motor de Function Blocks Honeywell — modelo **compilar→binario→descargar**), deofuscada | `types/observability.md` |
| **B143** | nmodsreflow.77 (`-rt`): subsistema sync (colaboración config multiusuario JSON-Patch, favoritos por-usuario, sin locking) | `types/issues-and-gotchas.md` |
| **B145** | nmodsreflow.77 (`-rt`): superficie REST de config (read con `?file=` traversal, overwrite total, delta = 2ª puerta a applyConfig) | `types/issues-and-gotchas.md` |
| **B289** | The live station's file space: `^` resolves to `shared/`, and why a `.px` cannot live in `Drivers/` | `types/logic.md` |
| **B301** | Licensing: there is no `modbus` feature — there are four, one per palette; the TCP gateway spends the `modbusTcp` licence; and the guide's limits topic quotes an MS/TP example | `types/driver-authoring.md` |
| **B347** | The UI layers: bajaux field editors + one unrestricted-but-benign preview RPC (ux), and a Swing output widget that pretty-prints via bundled Gson with NO typed deserialization (wb) | `types/third-party-libraries.md` |
| **B402** | Station Save Trigger and Dirty-Flag Propagation: BStationSaveJob, StationManager, and BBogSpace | `types/logic.md` |
| **B405** | BOG Version Migration: BIBogElementConverter, ConverterRegistry, BBogMigrator, and MigratorRegistry | `types/distribution.md` |
| **B408** | BComponentSpace Internal Lifecycle: LoadCallbacks, SubscribeCallbacks, TrapCallbacks, AuditableSpace Annotation, and BHandleScheme Resolution | `types/logic-authoring.md` |
| **B426** | How the `program` module compiles a Program: a spawned `bin/javac` subprocess, with tools.jar one level removed | `types/math-control-libraries.md` |
| **B459** | JACE-8000 architecture: a QNX/ARM embedded controller, not Linux or Windows (focus jace8000 bootstrap, J1) | `types/structure.md` |
| **B473** | J8 DEMONSTRATED END-TO-END: a hand-rolled Fox client pulled the live station's config.bog over foxs:4911 with no Workbench (read-only save=false) — plus live corrections: the JACE runs QNX … | `types/structure.md` |
| **B507** | `apis` API1: `@NiagaraRpc` — N4's annotation-driven server-side RPC API (`POST /rpc/{method}/{ord}`, JSON in/out, three transports web/box/fox, four-layer auth: session + CSRF + HTTPS + … | `types/actions.md` |
| **B512** | `apis` API4: the BOX protocol wire — N4's live-subscription substrate (`BBoxServlet` at `/box`, **JSON** frames not binary, a `ProxyBroker`-backed sub/event model, HTTP-poll v1 + … | `types/dashboard.md`, `types/logic-authoring.md` |
| **B517** | `framework-drivers` FD11: `basicDriver` — the shared serial-driver base that 12 drivers ride, layering a message-dispatch/transaction engine (`BBasicNetwork`) + a serial binding … | `types/driver-authoring.md` |
| **B536** | The `control` module writable-point model: `WritableSupport`, the 16-level priority-array arbitration, override actions, and the point-extension execution chain | `types/logic.md` |
| **B537** | The kitControl function-block catalog: 151 classes → ~130 deployable blocks across 10 packages, with the PID loop, the latch/switch/select mux family, and the multi-input null contract … | `types/logic.md` |
| **B539** | BLoopPoint deep: ramp-aware anti-windup, direct/reverse sign, the four disable actions, the deviation loop alarm, propagateFlags masking, and the official P/PI/PID tuning methodology | `types/logic.md`, `types/math-control-libraries.md` |
| **B541** | The `program` module runtime: BProgram execution, freeform vs Robot, source+bytecode stored in the `.bog`, and the code-signing/SecurityManager sandbox | `types/logic-authoring.md` |
| **B543** | HVAC control-logic SAFETY / fail-safe analysis: the five defensive layers, and the six default-UNSAFE gaps an engineer must close | `types/logic.md` |
| **B552** | The control-point alarm + history extension chain: BAlarmSourceExt + the offnormal-algorithm family, interval-vs-COV history, and the confirmation that alarms are notification-only | `types/logic.md` |
| **B553** | N4 real-time / push transports consolidated (API10): Fox, BOX (HTTP-poll v1 + WebSocket-push v2), WebSocket — and Server-Sent Events PROVEN-ABSENT | `types/dashboard.md` |
| **B587** | The `hierarchy:` ORD scheme: a `BSpaceScheme` whose body is a slash-path (`/Hierarchy/seg/seg…`) resolved level-by-level — a grouping segment resolves to a `BLevelElem`, an escaped … | `types/logic-authoring.md` |
| **B617** | graphql-admin (GQL-G5): module classloader isolation — a standalone module can safely bundle graphql-java | `types/ml-libraries.md`, `types/module-wiring.md` +1 |
| **B630** | Niagara N4 — module-anatomy (MA2): the module BOOT scan — `modules/` is enumerated by manifest `runtimeProfile` (not filename), dependency-resolved recursively, and loaded from a PREBUILT … | `types/structure.md` |
| **B635** | Niagara N4 — module-anatomy (MA7): module `<permissions>` → Java security policy — two tracks (`<java-permissions>` enforced by the SecurityManager per-CodeSource … | `types/logic.md` |
| **B793** | B793 · Building B790's minimal module: the skeleton is gate-green [CERT], and the 10 corrections the build forced | `METHODOLOGY.md`, `types/logic-authoring.md` |
| **B795** | B795 · MM3 decision table — the slot-change survival matrix as a machine-readable classifier for schema-risk.sh (mechanizes B754) | `BUILD-LOOP.md`, `METHODOLOGY.md` |
| **B796** | B796 · The `-ux` write-surface seam, captured as THE kit exemplar — the dashboard module -ux walked with file:line (document-mode §20) | `types/dashboard.md` |
| **B797** | B797 · `verify-module.sh --plano`: pinning the four-value agreement check (issue #47 spec) `[CERT]` | `types/dashboard.md` |
| **B800** | B800 · Console-log census: exceptions through the operator's OWN modules (triage-console.sh contract input) `[CERT]` | `BUILD-LOOP.md`, `METHODOLOGY.md` +2 |
| **B801** | B801 · The `Clock` timer delay/period floor — `schedule`/`schedulePeriodically` reject a non-positive value with `IllegalArgumentException` (extends B775) `[CERT]` | `METHODOLOGY.md`, `ORCHESTRATION.md` +3 |
| **B802** | B802 · Inter-module communication is module-AGNOSTIC within a station — links, service discovery, and subscription never check the module; the only real boundaries are the compile-time Type … | `types/logic-authoring.md` |
| **B803** | B803 · Step-up / re-authentication for a critical dashboard write — Niagara ships NO core step-up primitive; the clean `-ux` pattern re-verifies the session user through their auth scheme … | `METHODOLOGY.md`, `types/dashboard.md` |
| **B804** | B804 · History-extension authoring — a `BHistoryExt` IS a point extension; pick Interval vs COV; `BHistoryConfig` sets capacity + rollover; ONE ext per logged slot `[CERT]` | `types/logic-authoring.md` |
| **B805** | B805 · RT control-logic exemplars — how Tridium implements PID / deadband / latch / protection, the control-physics WHY behind each line, a one-bit fault trace from point to console, and … | `skill/SKILL.md`, `types/logic.md` |
| **B806** | B806 · Resource budget for a JACE/station — what oversaturates it, and the viability of our module logic `[CERT]` | `METHODOLOGY.md`, `ORCHESTRATION.md` +2 |
| **B807** | B807 · The N4 build pipeline + module versioning — plugin-source-cited task matrix, the station-lock copy, and the reload code path `[CERT]` | `BUILD-LOOP.md`, `METHODOLOGY.md` +2 |
| **B808** | B808 · Who watches the logic — feedback surfaces, traced from the cold-room module BEvaporatorUnit fault to the operator `[CERT]` | `types/logic.md` |
| **B809** | B809 · Tridium `-wb` authoring conventions — the 5 checks (off-UI-thread traversal, @AgentOn breadth, wb lexicon, load/save lifecycle, deps) + a "good -wb artifact" doctrine, Tridium … | `types/wb-widgets.md` |
| **B810** | B810 · Driver-module authoring — the BNetwork→BDevice→BPointDeviceExt→BProxyExt hierarchy, BPingMonitor as the device-health "quién vigila", and the critical finding that a write to a DOWN … | `BUILD-LOOP.md`, `types/driver-authoring.md` +1 |
| **B811** | B811 · Station snapshot automation — how to copy a running station (bog, logs, logic, links) to inspect it, without mounting the filesystem `[CERT]` | `BUILD-LOOP.md` |
| **B812** | B812 · The heartbeat/liveness watchdog pattern — an author-built independent monitor that detects a STALLED producer (not a bad value) `[INFER, primitive-grounded]` | `METHODOLOGY.md`, `types/logic.md` |
| **B813** | B813 · UX servlet authoring conventions — registration/routing/lifecycle, the JSON+facet+CSRF contract, from Tridium's own BWebServlet vs our dashboards `[CERT]` | `types/dashboard.md` |
| **B814** | B814 · Authoring a MODULE tag dictionary — ship a smart dictionary so your components are nav/search/hierarchy addressable `[CERT]` | `types/logic.md` |
| **B815** | B815 · Authoring a STATION-LEVEL / component-lifecycle test — mount the component, drive the interval, catch what lint and pure-JUnit cannot `[CERT]` | `BUILD-LOOP.md`, `METHODOLOGY.md` +2 |
| **B816** | B816 · Write-path & overlap testing — the threading/link-override mechanism (a dashboard write to a LINK-TARGET slot lands then is SILENTLY overwritten; servlet `set()` and the engine … | `METHODOLOGY.md`, `types/logic-authoring.md` +1 |
| **B818** | B818 · Forensics: `Missing class for "<module>:HoaMode"` was a DANGLING module-include.xml registration (a dead `<type>` for a class never built), not a bog retype — the runtime triage … | `types/logic-authoring.md` |
| **B819** | B819 · Zero-demand / idle-state doctrine for staged processes — demand decides WHETHER, a process variable only modulates HOW MUCH; NaN never counts as demand; every staged process has an … | `types/logic.md`, `types/math-control-libraries.md` +1 |
| **B821** | B821 · Protection anatomy of our RT modules — every protection trip as (WHAT fires · WHO fires · WHO watches), one-bit-traced at `fbe9009`; the RT control logic raises ZERO alarm-console … | `types/logic.md`, `types/ml-libraries.md` |
| **B822** | B822 · Making the room setpoint remotely settable — the ADDITIVE-code options (a `setpointCmd` writable-double slot, or an `applySetpoint(BDouble)` OPERATOR action) vs the OUTAGE of … | `types/actions.md`, `types/logic-authoring.md` |
| **B823** | B823 · Can the EXISTING `BRoomPanel.setpoint` be written from the mini-PC with NO module change? — every no-code write channel enumerated, one-bit-traced `[CERT]` | `types/dashboard.md`, `types/logic-authoring.md` |
| **B825** | B825 · Why an oBIX wrapped-`<obj>` PUT of a `BStatusNumeric` façade slot PROPAGATES to the linked control point — and why the read can lag: the write is a TOP-SLOT REPLACEMENT (decode into … | `types/logic-authoring.md` |
| **B826** | B826 · Is a `/setpoint/value` CHILD ORD routable over oBIX? — closes [B825]-G2 by read-only reasoning: the `BStatusValue` agent COLLAPSES the struct to a leaf `<real>` so the child is NEVER … | `types/logic-authoring.md` |
| **B827** | B827 · Raising an OPERATOR ALARM for a protection trip in an -rt module — the concrete fix for [B821]'s zero-alarm finding (seed S18): `BAlarmSourceExt` needs a `BControlPoint` parent, so a … | `types/logic.md` |
| **B828** | B828 · HOA modes as a real `BFrozenEnum` over oBIX (for future modules) — a `@NiagaraEnum(@Range …)` `BHoaMode` renders `<enum val="hand" display="…" range="…"/>` and decodes `<enum … | `types/logic-authoring.md` |
| **B829** | B829 · One audit trail for a client site — what Niagara's config-audit records for each write path, settled by CODE: a slot `set()` fires an `AuditEvent` ONLY inside `if (context != null && … | `types/dashboard.md`, `types/security.md` |
| **B853** | A vendor Px in the BROWSER: the Hx web profile (EC-Net 4.3.58.18 ≈ N4.4) — ActionBinding/set limits, Paste Special, and the working stock-Px HOA/selector recipe | `types/issues-and-gotchas.md` |
| **B872** | Driver poll-scheduler, tuning policy, and history/schedule/file extensions (fills B810) | `types/driver-authoring.md`, `types/logic-authoring.md` |
| **B873** | The `event` module: the wired event-routing substrate (sources → filters → recipients) | `types/logic-authoring.md` |
| **B874** | The Niagara `schedule` framework: time-based control primitive (hierarchy, recalc, priority) | `types/logic-authoring.md` |
| **B909** | DDF comm concurrency: the transaction manager (single-outstanding queue vs tag-correlated multiple) + the unsolicited-frame manager (async request correlation) | `types/logic-authoring.md` |
| **B928** | The converter catalog as a type-conversion MATRIX: 104 classes / 102 `@Adapter` pairs over ~17 core baja value types (String/StatusString are the hubs) | `types/logic.md` |
| **B1004** | JACE-8000 QNX `moduleVerificationMode` default is `medium` — STRICTER than the Win Supervisor's explicit `low` | `build-verify.md` |
| **B1010** | `Nre.java` compiled-in fallback for `niagara.moduleVerificationMode` is `medium` for N4 ≥ 4.10 — Confirms B1004 at [CERT] level | `build-verify.md` |
| **B1016** | The client's canonical N4 module build configuration: `gradle.properties` + `settings.gradle.kts` (SDK pin, JDK toolchain, gradle-plugin pin) across the four module groups | `BUILD-LOOP.md`, `build-verify.md` +2 |
| **B1017** | Modeling equipment status on a dashboard module: operating STATE (rt slot) vs point HEALTH (BStatus), and how the -ux reader + SPA reflect both | `types/dashboard.md` |
| **B1028** | SDE7-G1: Conformance of our three modules vs the SDK Exemplars — moduleTest, ux Specs, module.palette (confirmed against live source) | `types/issues-and-gotchas.md`, `types/moduleTest.md` |
| **B1054** | vendor pure-resource -wb palette modules: WB profile with only images and a declarative `module.palette` | `METHODOLOGY.md` |
| **B1055** | andoverAC256-wb: Tridium driver WB with DeviceManager, PointManager, Console view, and BackupManager | `types/wb-widgets.md` |
| **B1057** | ccn-wb: Carrier CCN driver WB — nine agents across BDeviceManager, BFolderManager, BPointManager, and custom BWbComponentView with agent-slot override pattern | `types/driver-authoring.md`, `types/wb-widgets.md` |
| **B1058** | andoverInfinity-wb: VT100 terminal emulator as BWbComponentView — RT topic-fire pattern, screen-scraping intelligence, and backup/restore menu agent | `types/logic-authoring.md`, `types/wb-widgets.md` |
| **B1060** | clCBus-wb: Honeywell C-Bus driver WB — nDriver-based device + point managers, parameter file tree-table, rich 5-tab schedule editor, and PX-editor alarm-hyperlink listener | `types/logic-authoring.md` |
| **B1061** | WV8 synthesis: cross-vendor WB intuitive-UI pattern catalog and PROPOSED-DELTA list for build-n4-module kit | `METHODOLOGY.md`, `types/dashboard.md` +2 |
| **B1064** | ODA3: the dashboard module data path and frontend quality — REST-POLL, {v,st} JSON contract, no external libraries, preview server, portability | `types/dashboard.md` |
| **B1066** | ndriver-wb deep: MVC internals, agent-command extension point, column reflection, state persistence, and DnD absence | `types/driver-authoring.md` |
| **B1067** | bacnet-wb: BBacnetDeviceManager, BBacnetHistoryImportManager, field editor family, and protocol-specific WB anatomy | `types/wb-widgets.md` |
| **B1069** | Gateway web-serving: servlet live-view, auto-refresh options, oBIX surface, outbound HTTP, and auth for API consumers | `types/cloud-connector.md`, `types/issues-and-gotchas.md` |
| **B1070** | Multi-tab BWbComponentView: BTabbedPane pattern for components that own multiple child collections | `types/wb-widgets.md` |
| **B1072** | ndriver child gaps: subscribeDepth contract + getDatabaseInstanceDefault factory + NMgrModelUtil column assembly | `types/driver-authoring.md` |
| **B1073** | bacnet-wb child gaps: BacnetDeviceLearn 14-column learn table + BBacnetPointManager 18-column point set | `types/wb-widgets.md` |
| **B1075** | Gateway follow-up: BHttpClient send-only API, BWebServlet sync-only (no native SSE), BAuthenticationScheme SPI | `types/security.md` |
| **B1076** | BTabbedPane edge cases + MQTT/cloud SPI hierarchy | `types/cloud-connector.md`, `types/wb-widgets.md` |
| **B1077** | honeywellDeviceManager-wb: shared Honeywell WB device manager base | `types/driver-authoring.md`, `types/wb-widgets.md` |
| **B1078** | honeywellBacnetDeviceManager-wb + honeywellModbusDeviceManager-wb: Honeywell OEM WB specialization | `types/wb-widgets.md` |
| **B1079** | galileoKitPx-wb: Galileo PX binding system (pin-level authorization + EasyPicture) | `types/security.md`, `types/wb-widgets.md` |
| **B1080** | honPlantControllerHMI-wb: BACnet export table manager + writable-slots editor + firmware OTA menu | `types/distribution.md`, `types/logic-authoring.md` |
| **B1082** | cloudLinkHonSbp: Honeywell SBP cloud connector RT architecture | `types/logic-authoring.md`, `types/observability.md` +1 |
| **B1087** | Custom BWbFieldEditor recipe: @AgentOn, station-component chooser, and BComponent.add() for creating points from WB | `types/wb-widgets.md` |
| **B1088** | BAbstractManager: class anatomy, init protocol, lifecycle, and support-object wiring | `types/wb-widgets.md` |
| **B1089** | MgrModel and MgrColumn: database-table model, column taxonomy, and addInstances | `types/wb-widgets.md` |
| **B1091** | MgrController commands, the discovery/learn split-pane, and the minimal custom-manager recipe | `types/wb-widgets.md` |
| **B1094** | obixDriver-wb: the oBIX CLIENT driver Workbench profile (station-as-oBIX-client managers, local-vs-remote discover, and the client-over-REST WB pattern) | `types/driver-authoring.md` |
| **B1095** | lonworks-wb: the Tridium core LonWorks driver Workbench profile (NV-binding managers, changeable-NV type editing, LNML/XIF import, and the LON commissioning UX) | `types/driver-authoring.md` |
| **B1096** | honBACnetUtilities-wb: Honeywell's BACnet extension Workbench profile (device type-float over stock managers, notification-class alarm manager, batch export view, dual bitstring FE, and the … | `types/driver-authoring.md`, `types/wb-widgets.md` |
| **B1097** | niagaraDriver-wb: the Supervisor→JACE Workbench surface (station manager, bidirectional history/schedule/user sync, file transfer over the point channel, and the N-space path-bar agent) | `types/driver-authoring.md`, `types/wb-widgets.md` |
| **B1098** | ace-wb: the Tridium ACE (Distech "Titan") driver Workbench profile — the richest vendor WB pattern (own wire sheet, app-creation wizard, catalog palette, opcode expression editor, and … | `types/wb-widgets.md` |
| **B1099** | platPower-wb: the Workbench PlatformServicePlugin SPI (a platform-daemon-connection extension point, distinct from station component views) — JACE battery/power monitoring UI | `types/wb-widgets.md` |
| **B1100** | mbus-wb: the M-Bus (EN 13757) metering driver Workbench profile (dual primary/secondary addressing, the 5-mode discovery wizard, multi-baud scan, and the manufacturer-specific data-record … | `types/driver-authoring.md`, `types/issues-and-gotchas.md` |
| **B1101** | nSnmp-wb: the SNMP v3 driver Workbench profile (USM/VACM security-manager cluster, MIB-browser discovery, and the station-as-SNMP-agent export surface) | `types/wb-widgets.md` |
| **B1102** | zwave-wb: the Z-Wave mesh-wireless driver Workbench profile (two-level scene management, node inclusion/exclusion + failed-node recovery, firmware-gated options, and battery wake-up … | `types/driver-authoring.md`, `types/wb-widgets.md` |
| **B1103** | opc-wb: the OPC DA classic client driver Workbench profile (the DCOM/NT security-config view, lazy hierarchical COM browse, and the parallel device/point MVC quartet) | `types/driver-authoring.md` |
| **B1104** | honAdvWirelessCfg-wb: Honeywell BEATS wireless platform-config Workbench profile (WiFi + BLE radio provisioning over the platform-daemon file channel, from the Workbench Platform tab) | `types/security.md`, `types/wb-widgets.md` |
| **B1105** | wb-vendor-ux wave-3 synthesis: the cross-vendor WB pattern catalog (11 driver/platform profiles), the WB "target" taxonomy, and the consolidated kit-delta list | `types/wb-widgets.md` |
| **B1106** | module-hardening REF3: the complete BFacets key catalogue (reference card) — every facet key a module author can set, with string literal, purpose, and factory method | `types/logic-authoring.md` |
| **B1108** | module-hardening REF1: the complete BQL function catalogue (5 aggregates + 7 expression scalars + the 24-function BBqlTime library) and why the names are un-greppable (reflective resolution) | `types/logic.md` |
| **B1109** | module-hardening REF4: the BEnumRange authoring & encoding grammar (`facets="range=…"`, the `frozen+{dynamic}?options` BNF, and the `E:` BOG facet form) | `types/logic-authoring.md` |
| **B1110** | module-hardening REF7: the SlotPath slot-name grammar (what `isValidName` accepts/rejects, the `$xx`/`$uXXXX` escaping, and why programmatic names from field data fail silently) | `types/logic-authoring.md` |
| **B1112** | module-hardening REF5: the BFormat pattern language (`%script%` reflective calls, the `?` fallback, special functions, and the reflection-resolution order) — used in alarm/history/display … | `types/logic.md` |
| **B1124** | module-hardening BLD2: a newer-bytecode (major ≥ 53) class throws a RAW UnsupportedClassVersionError — N4's ModuleClassLoader does not wrap it, and catch(Exception) won't catch it (it's a … | `types/issues-and-gotchas.md` |
| **B1125** | module-hardening BLD3: split-package resolution is first-dependency-wins (declaration order, with a lastModuleScanned cache) — a same-FQCN class in a second module is silently shadowed; N4 … | `types/issues-and-gotchas.md` |
| **B1128** | module-hardening BLD6: a `<type>` in module-include.xml whose class has no @NiagaraType (thus no generated `TYPE = Sys.loadType()`) is caught at registry-build with SEVERE "Missing … | `types/module-wiring.md` |
| **B1134** | module-hardening UXS4: CORS on a Niagara servlet — `BGenericHttpHeaderProvider` CAN inject `Access-Control-Allow-Origin` (static value), but `BProfileFilterFactory` is closed to third-party … | `types/security.md` |
| **B1135** | module-hardening BLD7: signing alias/profile absent at build — the Gradle plugin's behavior is PROFILE-TYPE-DEPENDENT (Local auto-generates a dev cert → silently signed-but-untrusted … | `types/issues-and-gotchas.md` |
| **B1137** | module-hardening PER1: dynamic-slot ghost accumulation — the prune API exists (`getDynamicPropertiesArray` + `remove`), but there is NO framework "prune-orphans" utility; non-transient … | `types/logic-authoring.md` |
| **B1138** | module-hardening PER4: a worked `started()` self-migration recipe — detect an orphan dynamic slot, copy its value to the new frozen slot, remove the orphan (safely, via a snapshot) — closes … | `types/logic-authoring.md` |
| **B1140** | BWbComponentView subscription gap vs BAbstractManager: why child slot values show defaults on first paint and the minimal fix | `types/wb-widgets.md` |
| **B1141** | Reading a control point's `out` value in rt: typed extraction, BStatus semantics, BOrd resolution, and facets | `types/logic-authoring.md`, `types/wb-widgets.md` |
| **B1142** | Creating control points from wb code: direct `BComponent.add()` on a BFolder, MgrController/Mark.moveTo path, BEnumRange.make, and unique slot naming | `types/wb-widgets.md` |
| **B1146** | SES1-G2: the advisory license fault IS honored in practice — well-behaved services gate their work on `isOperational()` — but it is a CONVENTION the framework never enforces | `types/security.md` |
| **B1158** | module-hardening RUN8: the HOT `changed()` + non-transient WRITE anti-pattern — a control component that re-runs its full cycle on every NRIO-linked input update (~18/s) and writes … | `types/issues-and-gotchas.md` |
| **B1159** | module-hardening PER8: the PERSISTED-ACCUMULATOR-written-every-cycle anti-pattern — a monotonic counter slot declared non-transient (so it survives restart) and re-written on every control … | `types/issues-and-gotchas.md` |
| **B1160** | module-hardening UXS7: the STATIC `-ux` session/token store with lazy-only eviction — a `static ConcurrentHashMap<jsessionId,Entry>` that inserts on login, removes on explicit logout, and … | `types/issues-and-gotchas.md` |
| **B1172** | n4-agent-mcp AM5: placing a component on the wire sheet — `BWsAnnotation` ("p,q,w[,h]" wixels), the add-or-set recipe (flags READONLY, Transaction batch), default auto-stagger, drop/paste … | `METHODOLOGY.md`, `types/dashboard.md` +1 |
| **B1176** | n4-agent-mcp AM9: saving, persistence, transactional safety, undo and audit — the station `save` action is an ASYNCHRONOUS job (`config.bog.working` → backup → rename), `Transaction` … | `types/logic.md` |
| **B1177** | n4-agent-mcp AM10: the BOX write path as a wire recipe an external client can implement — session bootstrap (`ssession` make/makessc/callssc), handle discovery … | `types/dashboard.md` |
| **B1222** | OPC DA (classic) end-to-end connection recipe: Windows-only host, OPCEnum + DCOM server/client hardening, firewall, and the OpcNetwork → OpcDaClient → points sequence, with every … | `types/driver-authoring.md` |
| **B1231** | n4-dev-course DC2: Slots as Tridium teaches them — properties/actions/topics, Slot-o-Matic output, frozen vs dynamic CRUD, slot names, flags, facets (and the rules the deck leaves implicit) | `types/actions.md`, `types/logic.md` |
| **B1232** | n4-dev-course DC3: Components (deck 3 + Labs 1–4) — callbacks, validation, lifecycle, services, palettes, and the BFirstComponent → BAdd → BTstat lab ladder | `types/logic.md` |
| **B1233** | n4-dev-course DC4: "Niagara Fundamentals" as taught — modules, the Type system, the BObject hierarchy decision rules, common classes and BOG; plus the code that ENFORCES Tridium's rules … | `types/module-wiring.md`, `types/structure.md` |
| **B1234** | DC5 Code Signing as Tridium teaches it: the internal-CA lab recipe (createProfile → generateCertificate → CSR → importCertificate → niagaraSigning), why the lab flips … | `build-verify.md` |
| **B1244** | OPC UA client child-gap bundle OCC2: the first-connect certificate path decompiled (`DefaultExemptionApprover` always says no, `CoreClientTrustManager` writes an UNAPPROVED `host:port` row … | `types/driver-authoring.md` |
| **B1252** | n4-dev-course DC15: the 41 course files no block cited — the 25-test slide suite read as Tridium's testing style guide (BTestNg vs BTestNgStation, fixed `Thread.sleep` waits instead of … | `types/value-types.md` |
| **B1253** | n4-dev-course DC16: the written answer key for every lab exercise (Labs 1–11, Bonus driver labs, Appendix A) — 273 items answered from code without execution, 17 shipped-solution defects … | `types/logic.md` |
| **B1254** | n4-dev-course DC17: written solutions to Review Assignments 1 and 2 (`BComponentWithId`, `BPointSummaryService`) — design decisions, traceability, traps, and the code findings the decoding … | `types/logic-authoring.md` |
| **B1270** | data-egress DE1: audit-first inventory of every egress and ingress channel of a JACE or Supervisor (63 rows) and the five-class taxonomy DE2–DE9 use, incl. `/box/query` history paging and the export RPC … | `types/data-access.md` |
| **B1271** | data-egress DE2: read-only egress per channel from code — request model, per-request station cost, threads, rate ceiling, latency, status/unit fidelity; best read path per scenario … | `types/data-access.md` |
| **B1272** | data-egress DE3: every way to write into a station — user it runs as, permission/range/audit gate, failure signal, traps; the min/max slot-facet check on user-Context `set` … | `types/data-access.md` |
| **B1273** | data-egress DE4: READ+WRITE bidirectional applications on N4 — seven architectures (station servlet, split oBIX, BOX, OPC UA/BACnet, Supervisor proxies, MQTT/cloudLink, Haystack) compared … | `types/security.md` |
| **B1274** | data-egress DE5: bulk, historical, alarm and audit export compared (oBIX history query/rollup, `/ord` CSV exporter views over `history:`/`alarm:`/`bql:`, rdb export, report CSV, alarm feed …) | `types/data-access.md` |
| **B1275** | data-egress DE6: real-time and event channels from code — oBIX watches as a lazily swept short poll, BOX/WebSocket push cadence, Fox 500 ms pull, MQTT drop while disconnected, alarm replay … | `types/data-access.md` |
| **B1280** | data-egress B1278-G2 + B1262-G1: the N4.14 → N4.15.3.28 → N4.15.6.20 parity pass over every class behind the data-egress decision guide (22 jars, 2 615 CRC-changed Tridium class files … | `types/driver-authoring.md` |
| **B1284** | OptimizerSupervisor post-install parameter map (OS3): every file you can tune after install, which copy wins, and the live N4.14.0.162 values — nre.properties heap, system.properties … | `types/platform.md` |
| **B1304** | Threads, workers and Jobs: what the 4.15.6.20 Developer Guide and `javax.baja` source let a module do, mapped against the build-n4-module kit — the kit teaches a … | `types/logic-authoring.md`, `types/logic.md` |
| **B1305** | Lexicons: the 4.15.6.20 lookup chain, key shapes, overrides, locale, diagnostics and the JS `lex!` plugin mapped against the build-n4-module kit — the kit's `Type.slot` prefixing advice is … | `skill/SKILL.md`, `types/dashboard.md` +1 |
| **B1306** | Subscriptions, leases and watch: what the 4.15.6.20 Developer Guide and source let a module do (Subscriber, event masks, the three causes of "subscribed", leases, virtual components … | `types/driver-authoring.md`, `types/logic-authoring.md` |
| **B1307** | Trees and nav (BINavNode, BNavContainer, spaces, NavEvent, nav files, subtree traversal, bajaScript nav) per the 4.15.6.20 docs and source, mapped against the build-n4-module kit — the … | `types/structure.md` |
| **B1308** | BQL from code and ORDs/resolve per the 4.15.6.20 Developer Guide and `javax.baja.naming` source, mapped against the build-n4-module kit — `BOrd.make` neither parses nor normalizes (the kit … | `types/logic-authoring.md`, `types/security.md` |
| **B1309** | Links (BLink, direct vs indirect, LinkCheck, makeLink/BConversionLink, knobs, activation, link capacity) per the 4.15.6.20 Developer Guide and source, mapped against the build-n4-module kit … | `types/logic.md` |
| **B1310** | Facets, formats, units and collections (BFacets keys and builders, BFormat scripts, BUnit/BUnitConversion, BITable/TableCursor/Tables, BEnumRange/BDynamicEnum, BOrdList) per the 4.15.6.20 … | `types/logic-authoring.md` |
| **B1311** | Tags, relations, tag dictionaries, NEQL, search and hierarchies from code per the 4.15.6.20 docs and source, mapped against the build-n4-module kit — the kit's "ship a tag dictionary" … | `types/logic-authoring.md`, `types/logic.md` |
| **B1312** | Module permissions, the Security Manager, user permissions and code signing per the 4.15.6.20 Developer Guide and source, mapped against the build-n4-module kit — SIX permission groups … | `build-verify.md`, `types/distribution.md` +2 |
| **B1313** | Third-party libraries, API stability and module authoring files (uberjar, apiInfo, `moduleManifest`, `module-include.xml`, parts and profiles, the `-doc` part, registry) per the 4.15.6.20 … | `types/module-wiring.md`, `types/structure.md` +1 |
| **B1314** | Control points, priority arrays and status handling (BStatus bits, `propagateFlags`, kitControl math/switch/loop status rules, alarm-extension bits) per the 4.15.6.20 Developer Guide and … | `types/logic.md` |
| **B1315** | Horizons: 4.15.6.20 Developer Guide capabilities the build-n4-module kit never teaches (module web apps via `WEB-INF/web.xml`, the auto-logoff activity monitor, js-ux helpers, history … | `types/horizons.md`, `types/wb-widgets.md` |
| **B1316** | Engine contract child gaps closed from source: the engine action queue is FIRST-wins while `CoalesceQueue` is LAST-wins (B1304-G1), a null Context or null user makes … | `types/actions.md`, `types/logic.md` +1 |
| **B1317** | UI and format child gaps closed from source: `Command`/`ToggleCommand` read exactly `<keyBase>.label/.icon/.accelerator/.description` (B1305-G2), nav files load only through explicit ORD … | `types/logic.md`, `types/structure.md` +1 |
| **B1318** | Shipped-artifact child gaps closed: in the 26 readable 4.15.6.20 `dist.xml` manifests `noStation` is the live alias of the devguide's `noRunningApp` and `true` means the station MUST be … | `types/dashboard.md`, `types/distribution.md` |
| **B1319** | kitControl status behaviour of Select/MuxSwitch/Switch and the new 4.15 Hysteresis/Limiter (B1314-G2): with the default `propagateFlags` (`nullStatus` for switches, `ok` for logic/math) a … | `types/logic.md` |
| **B1320** | Shipped libraries I (SL1a): the third-party jars in the 4.15.6.20 `bin/ext` (httpclient5 5.3.1, okhttp 4.12.0 + okio 3.9.1 + kotlin-stdlib 2.0.20, jose4j 0.9.6, commons-lang3 3.19.0 … | `types/cloud-connector.md`, `types/third-party-libraries.md` |
| **B1321** | Shipped libraries II (SL1b): the Tridium modules that wrap third-party libraries in 4.15.6.20 — jsonSmart-rt (json-smart 2.6.0), jsonToolkit-rt/wb (jayway jsonpath, Gson 2.13.2) … | `types/third-party-libraries.md` |
| **B1322** | UG JsonToolkit (4.15.6.20 guide, byte-identical to 4.13.2.18) as a module author reads it: five Java extension points (`BJsonSchemaProperty<T>.getJsonValue`, `BQueryResultWriter` registered … | `types/json-toolkit.md` |
| **B1323** | UG MQTT (`abstractMqttDriver`, 4.15.6.20 guide): the stock driver is licensed (`tridium:mqtt`) and caps devices at 25 on a JACE and 50 (max 500) on a Supervisor, publishes ONE scalar per … | `types/cloud-connector.md` |
| **B1324** | UG Alarms + Histories (4.15.6.20) for cold-room/compressor modules: `timeDelay` never debounces FAULT, `alarmInhibit` is a linkable `BStatusBoolean` (defrost/compressor-off can suppress … | `types/logic.md` |
| **B1325** | UG Templates + Provisioning (4.15.6.20) for module authors: a template module is a `.jar` of read-only `.ntpl` files; the manifest `<dependencies>` must list every module the logic and Px … | `types/distribution.md`, `types/logic-authoring.md` +2 |
| **B1327** | UG BACnet + Modbus (4.15.6.20) for modules that read/write field points or expose values: `staleTime=0` makes points stale immediately when unsubscribed (the kit says "0 (off)") … | `types/driver-authoring.md`, `types/issues-and-gotchas.md` +1 |
| **B1328** | UG web UI and theming (Graphics, JavaWebClients, the web-profile/theme pages of User) for a dashboard-style module: a theme module is found by the registry def `themeName` whose VALUE is … | `types/security.md`, `types/theme.md` |
| **B1331** | UG KitControl (4.15.6.20) for cold-room/compressor control — reuse vs hand-roll: LeadLagRuntime/Cycles rotate ONE output every `maxRuntime` (default 6 min) with TRANSIENT outputs and no … | `types/logic.md` |
| **B1332** | UG ModuleSign + SigningService (4.15.6.20) reconciled with [Block 1312]: the Signing Service signs X.509 TLS/MQTT certificates by CSR and NEVER module jars; verification modes Low (warn) / … | `build-verify.md` |
| **B1333** | UG HttpClient (4.15.6.20): the shipped HTTP client driver does REST without Java (standalone `HttpClient` or `HttpClientNetwork` string points), is licensed (`http` feature + SMA, 1 … | `types/cloud-connector.md` |
| **B1334** | UG Touch (4.15.6.20) for our wall HMI: the guide documents the Android Niagara Touch kiosk display, not generic touch rules — its panel-user recipe is one user per display with Auto Logoff … | `types/dashboard.md` |
| **B1335** | UG Scheduling (4.15.6.20) as the replacement for hand-made Clock timers in cold-room modules: a `TriggerSchedule` fires `trigger` at fixed wall-clock times (one entry per defrost), keeps … | `types/logic.md` |
| **B1336** | UG EngNotes (4.15.6.20: 151 files, mostly oBIX R2 migration, BACnet MS/TP diagnostics, Expr components and display-name operator notes) mined for module-author rules only: the vendor … | `types/cloud-connector.md`, `types/driver-authoring.md` |
| **B1337** | UG Drivers (the generic driver framework behind NDriver, 4.15.6.20): `BPingMonitor` never pings a disabled or faulted device (a `configFail`ed device cannot self-clear by ping), gates … | `types/driver-authoring.md` |
| **B1339** | UG WebCharts + SeriesTransform (4.15.6.20) for the dashboard module: the stock webChart `ChartWidget` embeds in Px/Hx/Dashboard panes and its rt servlet (`/query/*`, `/file/*` with CSRF on POST) is … | `types/history-transforms.md` |
| **B1340** | UG Rdbms (4.15.6.20): the RDBMS driver supports exactly MySQL, Oracle, SQL Server and HSQLDB — no Postgres/Supabase — runs on a Supervisor only (controllers get HSQLDB), needs a … | `types/cloud-connector.md` |
| **B1341** | UG Platform (4.15.6.20) for module authors: `se` jars "can run on Windows-based hosts only", so a JACE never loads `-se` (the kit lists JACE as an `-se` host); a JACE's `-ux`/`-wb` jars … | `types/platform.md`, `types/structure.md` |
| **B1343** | UG Obix (4.15.6.20) for the client viewer/write-server: the guide is mostly about the oBIX CLIENT driver; its server content is the licence (`obixDriver` with `export=true` + the `web` … | `types/security.md` |
| **B1344** | Web session lifetime from source (closes B1334-G1, static half of B1315-G1): Niagara runs its own idle timer on `NiagaraSuperSession.lastActivity` (Jetty's max-inactive is never set) … | `types/dashboard.md` |
| **B1345** | dist.xml `noRunningApp` is ENFORCED, not reported (closes B1318-G1): `BDistribution.getProcessingFlags` turns it into bit `1L`, and both the headless `PlatformInstallManager` and … | `types/distribution.md` |
| **B1346** | httpClient-rt has no OAuth2 auth type and an unmounted `HttpClientBuilder` client is NOT silently allowed (closes B1333-G1): the 125-class jar is identical in 4.14 and 4.15 and offers … | `types/cloud-connector.md` |
| **B1347** | oBIX server and HIDDEN slots (closes B1343-G1): `ObixEncoder.getKids` lists a component's children only through `canSee`, which drops topics and `Flags.HIDDEN` slots, then keeps a child … | `types/security.md` |
| **B1348** | Driver health from source (closes B1337-G1 and B1327-G1): generic driver-rt never turns a failed poll into a ping fail — `BProxyExt.readFail` only marks the point and `Tuning.readFail()` is … | `types/driver-authoring.md`, `types/logic-authoring.md` |
| **B1349** | Security headers and themes on a module `BWebServlet` (closes B1328-G1): Jetty gives every `BWebServlet` (and every `web.xml` war module) its own context and `configureNiagaraWebApp` adds … | `types/frontend-standard.md`, `types/security.md` +1 |
| **B1350** | jsonToolkit extension contract, exact (closes B1322-G1): a custom property extends `BJsonSchemaProperty<T>` and implements ONE method, `T getJsonValue()` (from `BIJsonProperty`; `process` … | `types/json-toolkit.md` |
| **B1351** | kitControl lead/lag and interstart, from 4.15 source (closes B1331-G1): `BInterstartDelayControl` declares only `delay`, `master`, `startPending` — its min-on/min-off are … | `types/logic.md` |
| **B1353** | Two verification closes (B1317-G1, B1336-G1): none of the 728 module jars of the 4.15.6.20 install contains a `*.nav` resource and none of `baja`/`workbench-wb`/`web-rt`/`fox-rt` contains … | `types/logic.md` |
| **B1365** | Driver-framework docs vs code: the guide's `Stale Time = 0` ("stale immediately when unsubscribed") and `Num Retries Until Ping Fail` ("a ping does not fail until…") are not what driver-rt … | `types/driver-authoring.md` |
| **B1384** | Z-Wave doc-vs-code: the missed-wake-up "device down" rule never fires (doPingTimeout only logs), the documented `Configure As Sis` slot does not exist, the FLiRS policy is named … | `types/driver-authoring.md` |
| **B1453** | The `bin\` launchers and the `nre <class>` tool set from code: which class each exe boots, the full `hdbt` and `n4mig` usage, and the headless tools with no exe of their own (DependencyChecker, PaletteGenerator, RecompileTool) | BUILD-LOOP |
| **B1410** | DataRecoveryService defaults vs the guide: save-limit fault (3 saves in 15 min), persistent capacity triggers a save at 75 %, Service Enabled true | `types/logic.md` |
| **B1462** | Persistence map: TRANSIENT as the only gate, JACE auto-save 24 h with no backup, DataRecoveryService and Non-Critical, writable levels and override expiry at start, history/alarm/audit stores, files and backup excludes | `types/logic.md` |

---

Maintenance: this index is a curated pointer, not a copy — when a block's content is folded into a kit guide
(METHODOLOGY, BUILD-LOOP, `types/*.md`, build-verify), that guide carries the rule and cites the block; this
index stays the "what to read" map. Source: retro `corpus-index-rt-authoring-and-organization-blocks`.
