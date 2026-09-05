# campaign6 — corpus-index.md delta (B762–B785)

**Author**: investigador1 (Opus 4.8). **Status**: DRAFT for PR7 (uncommitted, feeds the corpus-index.md writer verbatim).
**Shape**: matches `build-n4-module-kit/corpus-index.md` — `Block | what it gives the builder | Layer`, with a Priority
(P0 read-before-building / P1 read-for-the-layer / P2 read-when-the-decision-comes-up). Here GROUPED BY KIT DESTINATION
FILE so the PR7 writer drops each row next to the section it documents. All blocks are grep-verified [CERT] (see each
block's self-verify tally).

> Coverage note: rows below are the investigador1 slice — the ux-testing focus (B762–B763) + the census slice
> (B778–B785). companero's MAE1–MAE6 (B772–B777) rows are appended by the second-reader pass as those blocks land;
> holes B764–B771 are intentional (allocation gaps, doctrine-tolerated, never refilled).

---

## → `types/logic.md`

| Block | What it gives the builder | Priority | Layer |
|---|---|---|---|
| **B779** | Child-tree containers, pick by cardinality: frozen `@NiagaraProperty` (fixed) / runtime `add(name,BValue)` + `reorder(Property[])` (data-driven) / typed `BFolder` (growable). NO `BComponentList`; typed-tree legality via `isChildLegal`/`isParentLegal` `instanceof` vetoes | P1 | rt / children |
| **B778** | Author-side SPIs: a custom SERVICE (`extends BAbstractService`, `getServiceTypes()` = registration-by-placement, `serviceStarted()`); a new ORD SCHEME (`extends BOrdScheme` + `@NiagaraType(ordScheme)` + `resolve()`); a SERVER-side subscription (`extends Subscriber`, `event(BComponentEvent)`) | P2 | rt / service |
| **B781** | Grouping/relating author postures (three, distinct): categories = NO author scaffold (runtime-only); relations = never subclass `BRelation`, define a type via `relationId`+`RelationInfo`/`BCustomRelation`; hierarchy = compose `BHierarchy`+`BLevelDef` under `BHierarchyService` | P2 | rt / relations |
| **B782** | Build a query/search/index surface with ONE recipe: a typed `BQuery`/NEQL payload + the matching `BIAgent` provider (`BQueryEngine`/`BColumnsProvider`/`BISearchProvider`/`BSystemIndexer`) discovered by the agent registry → a `BITable` | P2 | rt / data |
| **B783** | Template author path = ARTIFACT production, not type registration: a "template type" is an `.ntpl` ZIP made by a job from a `BTemplateConfig`-marked subtree — do NOT scaffold a `BTemplate` subclass (no SPI) | P2 | rt / template |
| **B785** | Extend a framework via a Device + a self-describing SPI object — the rdb dialect exemplar: `B<X>Database extends BRdbms`/`BEncryptableTransportRdbms` + 3 abstract methods; `getRdbmsContext()` → a 60-method `RdbmsDialect`; register a manifest `<type>`; no central registry | P2 | rt / data |
| **B777** | SECURITY-MODULE skeleton (saml-rt exemplar): module.xml → `BAbstractService` (`getServiceTypes`) → `BAuthenticationScheme` subclass whose real auth lives in a `NiagaraLoginModule` (wired via `getLoginConfiguration`, B510) + optional `BISecurityDashboardProviderAgent` (B563); permissions INLINE in module.xml `<permissions>` (NOT module-permissions.xml — B721 correction); jar-signed NIAGARA4.RSA/SF (B18) is MANDATORY; register `@AgentOn "baja:AuthenticationScheme"` | P2 | rt / security |
| **B776** | ACTION PROTECTION: gate declaratively — `@NiagaraAction(flags=Flags.OPERATOR=256)` → OPERATOR_INVOKE(4), OMIT → ADMIN_INVOKE(64) is the DEFAULT, enforced by `BComponent.canInvoke` + fox/box `PermissionException` (no code in the body); reserve operator for low-privilege writes, config/emergency = admin-only. Correct `doPrivileged` = a JVM permission ONLY (BPassword/setDefault/sys-prop); NEVER wrap a Niagara RBAC check (AP-27) | P2 | rt / security |
| **B775** | Authoring a WATCHDOG/monitor + choosing a timer: subclass `BAbstractAlarmMonitor` (override `doRunCheck`/domain `checkX`, `raiseAlarm` edge-latch, maintain `status`/`lastAlarmTime`); timer = `Clock.schedule` (one-shot) vs `Clock.schedulePeriodically` + keep the `Clock.Ticket` (NO `javax.baja.sys.BTimer` — cl.hvac only); `BRandom` = configurable-`BRelTime` exemplar. NOTE: cadence is a configurable `BIntervalTriggerMode` (15-min default), not a 2s poll; native `EngineWatchdog` (B124/B681) is a separate layer | P2 | rt / watchdog+timer |
| **B774** | Authoring a background JOB: subclass `BSimpleJob` + `run(Context)` for the normal async case (thread + success/fail + interrupt-cancel free), or raw `BJob` + `doRun`/`doCancel` to own threading; `progress(pct)`/`log()`; submit via `BJobService.submit(job,cx)` → an ORD handle (poll, no join); multi-step = `BJobStep`/`BDeviceJobStep.doRun` under a `BBatchJob` | P2 | rt / jobs |
| **B773** | Authoring an analytics compute NODE: `@NiagaraType` subclass of `javax.bajax.analytics.algorithm.BOutputBlock` implementing `getValue`/`getTrend` (or `BFunctionBlock.apply` single-input); inputs = `BBlockPin` `@NiagaraProperty` + `BLink` DAG edges; register by `module.xml <type>` (NO @AgentOn — contrast B782); filters/rollups/sources are all BOutputBlock subclasses; external feed = duck-typed `AnalyticDataSource.Provider` | P2 | rt / analytics |
| **B772** | Authoring a point EXTENSION: extend `javax.baja.control.BPointExtension` (there is NO `BAbstractPointExt`, no `onExtended`/`onRetracted`); implement the sole abstract `onExecute(BStatusValue out, Context cx)` (mutate `out`, or leave it for a notification-only ext); `requiresPointSubscription()`→true to see every change; reach the point via `final getParentPoint()`; legality via `isParentLegal`/`isSiblingLegal`; execution = slot-declaration order, proxyExt always first | P2 | rt / extensions |

**Idiom note (the META-delta — belongs at the TOP of `types/logic.md`'s author-side section):** B778 + B782 + B785 are
three instances of ONE Niagara extension idiom — *subclass a framework base + register a `<type>`/agent in module.xml +
hand back a self-describing SPI object; there is no central registry the author edits.* Teach it once; the per-surface
blocks (B778 service/ORD-scheme, B782 query providers, B785 rdb dialect) are instances.

## → `types/dashboard.md`

| Block | What it gives the builder | Priority | Layer |
|---|---|---|---|
| **B763** | The `-ux` servlet WRITE-surface, five gates: OPERATOR_WRITE fail-closed (deny on no-user/no-service/exception) → hand-rolled `X-Requested-With` guard IN the pure `route()` → `SERVICE_ORD` pinning/allowlist → per-Ord lock + HTTP 423 → audit (who/what/when/old→new). Plus the pure RBAC test seam (`canWrite(boolean)`) | P1 | ux / security |
| **B762** | Off-station testing seams for `-ux`: the pure `route()`→`RouteAction` servlet seam (thin adapter over the `final` WebOp; plain-JUnit testable), the purity gradient (inject the Baja touch as a `Function`), and the SPA-JS limit (`node --check` = syntax only) | P1 | ux / test |
| **B780** | `module.palette` + `module.lexicon` copy-ready conventions: palette `<p n= t= m=>` (bare-Type-minus-`B` names, plural folders, `m="alias=module"` once, nested pre-seed); lexicon is flat/module-global → PREFIX keys (`parent.child`) to dodge the B759 collision | P1 | palette / lexicon |

## → `types/wb-widgets.md`

| Block | What it gives the builder | Priority | Layer |
|---|---|---|---|
| **B762** | Off-station testing of `-wb`: the `wb/model/` lambda-injection seam — a Baja-free `model/` package with the slot check injected as a `Predicate<String>`; the `BWidget` view stays the adapter. `-wb` IS off-station testable (the discovery) | P1 | wb / test |
| **B780** | Dual-surface `@AgentOn` registration: write `@NiagaraType(agent={@AgentOn(types={"mod:Type"},requiredPermissions="r")})` on the view/FE; Slot-o-Matic emits `<type><agent><on/></agent></type>`; multi-type = one view over several source types | P1 | wb / agent |

## → `METHODOLOGY.md` / `corpus-index.md`

| Block | What it gives the builder | Priority | Layer |
|---|---|---|---|
| **B784** | Real `module.xml` conventions: profile split `-rt`/`-ux`/`-wb`/`-se` (server), `-doc` is a SEPARATE `runtimeProfile="doc"` module (never a code-module part); `<dependency>` `vendorVersion` = 3-part Tridium FLOOR (`4.14.0`) vs the module's own 4-part build stamp (`4.14.0.162`); header attribute roster | P1 | build / module.xml |

---

## Excluded

- **B761** (Honeywell Spyder → JACE-8000 connect + discovery workflow) — a FIELD-INTEGRATION / commissioning block
  (BACnet MS/TP + LON), NOT module authoring. Out of scope for the build-kit corpus-index (which maps the authoring
  corpus B729–B785). Do not index it here.

## → `types/logic.md` (addendum: minimal-module synthesis)

| Block | What it gives the builder | Priority | Layer |
|---|---|---|---|
| **B790** | The MINIMAL correct N4 module skeleton (copy-start): module.xml header + 3-part dep floor + non-empty palette + prefixed lexicon + one BComponent with one OPERATOR property + one HIDDEN engine action + one Clock.Ticket (started/atSteadyState arm, stopped cancel) + signed jar — gate-green + biting-check-green by construction; also the `scaffold-module.sh` fixture spec | P0 | rt / scaffold |

## Status: COMPLETE

All 14 MAE blocks (B772–B785) are covered and grep-verified — B772 (MAE1) by second-read (companero authored), B773–B777
(MAE2–MAE6) authored by investigador1 after companero stalled, B778–B785 (MAE7–MAE14) the original investigador1 slice.
Every block above has its corpus-index row. See the FOCUS-CLOSE retro
(`retros/2026-09-05-research-sdd-module-authoring-exemplars-FOCUS-CLOSE-retro.md`) for the consolidated delta list + the
cross-cutting extension-idiom meta-delta.
