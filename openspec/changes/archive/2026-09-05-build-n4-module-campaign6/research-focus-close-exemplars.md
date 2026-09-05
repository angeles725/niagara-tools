<!-- review-status: pending -->
<!-- Marker lifecycle: maintainer flips 'pending' → 'applied <date> · kit <sha>' (or 'dismissed') once folded; sweep-retros.sh reads this (METHODOLOGY §18). -->
# Retro — niagara-research · research-sdd · 2026-09-05 · FOCUS CLOSE: `module-authoring-exemplars` (MAE1–MAE14, B772–B785) — consolidated kit deltas for /build-n4-module

> **Focus-wide §18 close** covering BOTH lanes: companero (MAE1 → B772; second-read PASS by investigador1) +
> investigador1 (MAE2–MAE14 → B773–B785, after absorbing MAE2–MAE6 when companero stalled on a permission prompt).
> 14 blocks, one per authoring dimension. Every [CERT] cite was grep-verified at the real `organized/` paths — the
> verification pass CORRECTED the seed audit and delegated sweeps in EIGHT places (listed below). This SUPERSEDES the
> interim MAE7–MAE14 slice-retro (2026-09-05-…-mae7-14-slice-retro.md). READ-ONLY on the build kit — PROPOSES only.

## Consolidated kit deltas by DESTINATION FILE

### → `types/logic.md` (the bulk — an "author-side SPIs" chapter)
| Block | Delta (one line) |
|---|---|
| B772 (MAE1) | Point-extension authoring: extend `BPointExtension` (NO `BAbstractPointExt`), impl sole abstract `onExecute(out,cx)`; `requiresPointSubscription`, `getParentPoint()` final, legality overrides; exec = slot order, proxyExt first. |
| B773 (MAE2) | Analytics node: `@NiagaraType` subclass of `BOutputBlock` (`getValue`/`getTrend`) or `BFunctionBlock.apply`; inputs = `BBlockPin` + `BLink` DAG; register by `module.xml <type>` (NO @AgentOn); external feed = `AnalyticDataSource.Provider`. |
| B774 (MAE3) | Background job: `BSimpleJob`+`run(Context)` (thread/success/fail/cancel free) vs raw `BJob`+`doRun`/`doCancel`; `progress`/`log`; `BJobService.submit`→ORD (poll, no join); multi-step `BJobStep`/`BDeviceJobStep` under `BBatchJob`. |
| B775 (MAE4) | Watchdog/monitor: `BAbstractAlarmMonitor` (`doRunCheck`/`checkX`, `raiseAlarm` edge-latch). Timer: `Clock.schedule` vs `schedulePeriodically`+`Ticket`; `BRandom` = configurable-`BRelTime` exemplar. |
| B776 (MAE5) | Action protection: `@NiagaraAction(flags=Flags.OPERATOR=256)`→OPERATOR_INVOKE / omit→ADMIN_INVOKE (default), enforced by `canInvoke` (no body code); `doPrivileged` only for JVM perms, never RBAC (AP-27). |
| B777 (MAE6) | Security-module skeleton (saml-rt): module.xml → `BAbstractService` → `BAuthenticationScheme` (auth in a `NiagaraLoginModule` via `getLoginConfiguration`) + dashboard agent; register `@AgentOn "baja:AuthenticationScheme"`. |
| B778 (MAE7) | Author-side SPIs: custom service (`getServiceTypes` reg-by-placement), new ORD scheme (`BOrdScheme`+`resolve`), server-side subscription (`Subscriber.event`). |
| B779 (MAE8) | Child-tree containers by cardinality (frozen `@NiagaraProperty` / dynamic `add()`+`reorder` / typed `BFolder`); NO `BComponentList`; legality via `isChildLegal`/`isParentLegal` `instanceof` vetoes. |
| B781 (MAE10) | Categories/relations/hierarchy postures: categories = NO scaffold (runtime-only); relations = `relationId`+`RelationInfo` (not a BRelation subclass); hierarchy = compose `BLevelDef` under `BHierarchyService`. |
| B782 (MAE11) | Query/search/index = ONE recipe: typed `BQuery`/NEQL payload + a `BIAgent` provider → a `BITable`. |
| B783 (MAE12) | Template = ARTIFACT production (`.ntpl` via a make-job from a `BTemplateConfig`-marked subtree); NO `BTemplate` subclass SPI. |
| B785 (MAE14) | rdb dialect SPI: `B<X>Database extends BRdbms` + 3 abstract methods; `getRdbmsContext()`→60-method `RdbmsDialect`; register `<type>`; no central registry. |

### → `types/dashboard.md`
- B780 (MAE9): `module.palette` conventions (bare-Type-minus-B names, plural folders, `m=` alias once, nested pre-seed) + lexicon PREFIXING (flat/module-global → `parent.child`/`Type.slot`, the B759-avoidance discipline).

### → `types/wb-widgets.md`
- B780 (MAE9): dual-surface `@AgentOn` registration (Java annotation + Slot-o-Matic `<type><agent><on/></agent>`).

### → `METHODOLOGY.md` / `corpus-index.md`
- B784 (MAE13): real module.xml profile matrix (`-rt`/`-ux`/`-wb`/`-se`, `-doc` = SEPARATE module) + `<dependency>` 3-part Tridium FLOOR (`4.14.0`) vs the module's own 4-part build stamp (`4.14.0.162`).
- B775 (MAE4) note: correct the "MonitorWorker 2s poll" assumption (configurable `BIntervalTriggerMode`, 15-min default); distinguish native `EngineWatchdog` (B124/B681) from author-level `BAbstractMonitor`.
- B777 (MAE6) note: security modules embed permissions INLINE in module.xml, not a `module-permissions.xml` (B721 correction).

## Cross-cutting META-DELTAS (the highest-value synthesis)
1. **ONE recurring Niagara extension idiom** (B778 + B782 + B785 + B777): *subclass a framework base + register a
   `<type>`/agent in module.xml + hand back a SELF-DESCRIBING SPI object; no central registry the author edits.*
   Instances: services (`getServiceTypes`), ORD schemes (`resolve`), query/search/index providers (a `BIAgent`), rdb
   dialects (`RdbmsDialect`), auth schemes (`getLoginConfiguration`→a LoginModule). → teach ONCE at the top of
   `types/logic.md`; the per-surface blocks are instances. **The ONE documented EXCEPTION**: analytics nodes (B773)
   register by `module.xml <type>` with NO `@AgentOn`/BIAgent — note it explicitly.
2. **Two high-value NEGATIVES — tell the kit what NOT to scaffold**: categories have no author declaration (B781);
   templates have no `BTemplate` subclass SPI (B783). Emit no scaffold for either.

## Evidence-corrected assumptions (grep-verify earned its keep — 8 corrections)
- B772: `BAbstractPointExt` does not exist → base is `BPointExtension` (seed premise refuted).
- B779: `BComponentList` does not exist (confirmed absent); `reorder(Property[])` IS used (audit's "0 hits" refuted).
- B773: node base is `BOutputBlock`, not `BAlgorithmBlock` (seed candidate list corrected).
- B775: NO 2s MonitorWorker poll (configurable 15-min trigger); NO `javax.baja.sys.BTimer` (only cl.hvac).
- B777: security-module permissions are INLINE in module.xml, not a `module-permissions.xml` (B721 correction).
- Line-number drift in delegated sweeps corrected on verify in B779 (BComponent) and B780 (@AgentOn).

## Research-tools lane proposal (NOT a build-kit delta — routed here by the lead)
- `module_nav palette-lexicon-agents <module>` — dump palette `<p>` + lexicon (grouped by prefix, with a dup-bare-key
  collision report operationalizing B759) + `<type>/<agent>/<on>` cross-checked against `@AgentOn`; teach `module_nav
  resources` the `organized/<mod>/<sub>/extracted/` + `/vineflower/` layout. Evidence: B780. → niagara-research `tools/`.

## Already covered (dedupe)
Every MAE block CITES the framework MECHANISM (B4/B5/B6/B11/B12/B18/B20/B48/B66/B67/B408/B429/B510/B511/B558-566/B567/
B573/B584-586/B721/B729/B730/B734/B754/B755/B757/B758/B402-413/B124/B681) and adds only the AUTHOR-side residue.

## What went well (keep)
- INLINE grep-verify at real paths caught 8 sweep/seed errors before any shipped as [CERT] — the single most valuable
  discipline; a delegated sweep's file:line is a HYPOTHESIS until grep-confirmed.
- The negative + correction findings (what the kit must NOT do / what the seed got wrong) are as valuable as the recipes.
- Two-lane shared-corpus discipline (own rows only, pull --rebase before push, hand-recompute the envelope) held across
  14 concurrent-lane blocks + a stall handoff + a housekeeping push, with zero clobbers.
