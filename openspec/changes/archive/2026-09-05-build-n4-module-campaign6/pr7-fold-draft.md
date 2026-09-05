# campaign6 — PR7 fold draft (exact kit bullets to paste)

**Author**: investigador1 (Opus 4.8). **Status**: DRAFT for the PR7 writer — paste-ready kit text, grouped by
destination file + section, in the kit's terse checklist voice, each with `[ev: corpus B<n>]` and a GREP-BEFORE-FOLD
term (run it against the kit first; fold only if absent, to avoid double-folding). Skips the 6 module-ux-testing
deltas (B762/B763) already folded in PR4. Covers the exemplars focus (B772–B785) + own-modules focus (advisory
checklist + meta-delta) + the corpus-index 26-row delta.

---

## → `types/logic.md`

### NEW section "Author-side SPIs" — lead with the IDIOM, then the instances
> grep-before-fold: `grep -n "self-describing SPI object\|Author-side SPIs" types/logic.md`
- **The N4 extension idiom (one shape for most SPIs):** to extend the framework you *subclass a base + register a
  `<type>`/agent in `module.xml` + hand back a self-describing SPI object* — there is NO central registry you edit.
  Instances: a service (`getServiceTypes`), an ORD scheme (`resolve`), a query/search/index provider (a `BIAgent`),
  an rdb dialect (`RdbmsDialect`), an auth scheme (`getLoginConfiguration`). Learn it once. `[ev: corpus B778/B782/B785/B777]`
- **Custom SERVICE:** `extends BAbstractService`, override `getServiceTypes(){return new Type[]{TYPE};}` → registered
  by DROPPING it under `/Services` (auto at bootstrap); hook `serviceStarted()`; look up via `Sys.getService(Type)`. `[ev: corpus B778]`
- **New ORD SCHEME:** `extends BOrdScheme` (a `BSingleton`) + `@NiagaraType(ordScheme="<id>")` + `@NiagaraSingleton`,
  ctor `super("<id>")`, override `resolve()`; Slot-o-Matic emits `<type … ordScheme="<id>"/>`, registry resolves via
  `BOrdScheme.lookup`. `[ev: corpus B778]`
- **Server-side SUBSCRIPTION:** subclass `javax.baja.sys.Subscriber`, override `event(BComponentEvent)`, call
  `subscribe(component, depth, cx)` and `unsubscribeAll()` on stop (the SERVER complement to the BOX client). `[ev: corpus B778]`
- **EXCEPTION to the idiom — analytics nodes register by TYPE, not by an agent:** a custom analytics node is a
  `@NiagaraType` subclass of `javax.bajax.analytics.algorithm.BOutputBlock` (impl `getValue`/`getTrend`, or
  `BFunctionBlock.apply` for single-input); inputs are `BBlockPin` `@NiagaraProperty` wired by `BLink` DAG edges;
  registered by a plain `module.xml <type>` with NO `@AgentOn`; external feed = the duck-typed
  `AnalyticDataSource.Provider`. `[ev: corpus B773]`

### NEW section "Authoring a point extension"
> grep-before-fold: `grep -n "BPointExtension\|point extension" types/logic.md`
- **Extend `javax.baja.control.BPointExtension`** (there is NO `BAbstractPointExt`, no `onExtended`/`onRetracted`);
  implement the sole abstract `onExecute(BStatusValue out, Context cx)` — mutate `out` (control), or leave it
  (notification-only). Override `requiresPointSubscription()`→true only to see every change; reach the point via the
  `final getParentPoint()`; restrict hosting with `isParentLegal`/`isSiblingLegal`; execution is slot-declaration
  order, proxyExt always first. `[ev: corpus B772]`

### NEW section "Child-tree containers — pick by cardinality"
> grep-before-fold: `grep -n "BComponentList\|child-tree\|isChildLegal" types/logic.md`
- **Frozen `@NiagaraProperty`** (BComponent-typed) for fixed/known children; **runtime `add(name, BValue)` + `reorder(
  Property[])`** for data-driven children; **a typed `BFolder` subclass** for a homogeneous growable collection. There
  is NO `BComponentList`. Enforce a typed tree by overriding `isChildLegal`/`isParentLegal` (both default `true`) with
  `instanceof` vetoes. `[ev: corpus B779]`

### NEW section "Grouping/relating declaration surfaces" (three distinct postures)
> grep-before-fold: `grep -n "BCategoryService\|BRelation\|BLevelDef\|grouping/relating" types/logic.md`
- **Categories:** author NOTHING — every `BComponent` is `BICategorizable`; categories are operator-runtime via
  `BCategoryService`. Emit NO category scaffold. `[ev: corpus B781]`
- **Relations:** never subclass `BRelation` (a concrete carrier); define a relation TYPE by registering a `RelationInfo`
  / `BCustomRelation` in a tag dictionary. `[ev: corpus B781]`
- **Hierarchy:** compose a `BHierarchy` root + ordered `BLevelDef` children (`BQueryLevelDef`/`BRelationLevelDef` entity
  levels, `BGroupLevelDef`/`BListLevelDef` grouping) under `BHierarchyService`; subclass `BLevelDef`+`getElements` only
  for a bespoke level. `[ev: corpus B781]`

### NEW section "Query/search/index surface — one recipe"
> grep-before-fold: `grep -n "BQuery\|BITable\|query/search/index" types/logic.md`
- Declare a typed `BQuery`/NEQL payload + plug the matching `BIAgent` provider (`BQueryEngine` execute /
  `BColumnsProvider` table columns / `BISearchProvider` station search [`@AgentOn` scope×scheme] / `BSystemIndexer`+
  `BIIndexQueryProvider` station index [scope = a `BOrdList` of NEQL queries]) → read the resulting `BITable` cursor. `[ev: corpus B782]`

### NEW section "Templates are artifact production, not a type SPI"
> grep-before-fold: `grep -n "BTemplateConfig\|\.ntpl\|BTemplateService" types/logic.md`
- A "template type" is an `.ntpl` ZIP (a `.bog` + `template-manifest.xml`) produced by a make-job (`BMakeTemplateJob`/
  `NiagaraTemplate.createFrom().save()`) from a component subtree marked with a `BTemplateConfig` (+ `BConfigBinding`
  children + tagged parameter slots). Do NOT scaffold a `BTemplate` subclass — there is no such SPI. `[ev: corpus B783]`

### NEW section "Background jobs"
> grep-before-fold: `grep -n "BSimpleJob\|BJobService\|background job" types/logic.md`
- Subclass **`BSimpleJob`** + implement **`run(Context)`** for the normal async case (dedicated thread + auto
  success/fail + interrupt-cancel are free); report `progress(pct)` + `log().*`; submit via `BJobService.submit(job,cx)`
  and track the returned `BOrd` (poll `getJobState()`/`getProgress()` — no join). Use raw `BJob` (`doRun`+`doCancel`)
  only to own threading. Multi-step = `BJobStep`/`BDeviceJobStep` under a `BBatchJob`. `[ev: corpus B774]`

### NEW section "Watchdogs and timers"
> grep-before-fold: `grep -n "BAbstractMonitor\|schedulePeriodically\|Clock.Ticket\|watchdog" types/logic.md`
- **Watchdog/monitor:** subclass `BAbstractAlarmMonitor` (override `doRunCheck()`/domain `checkX()` +
  `getToNormal/OffnormalText`, maintain `status`/`lastAlarmTime`, edge-latch via `raiseAlarm(...)`). `[ev: corpus B775]`
- **Timer:** `Clock.schedule` (one-shot) vs `Clock.schedulePeriodically` (repeating) — KEEP the returned `Clock.Ticket`
  in a field, cancel it in `stopped()`, re-arm in `changed()` when the interval is configurable (`BRandom` exemplar).
  `BTimer` is a clHVAC wiresheet block, NOT a scheduler. `[ev: corpus B775]`

### NEW section "Action protection"
> grep-before-fold: `grep -n "Flags.OPERATOR\|canInvoke\|doPrivileged\|action protection" types/logic.md`
- Gate an action DECLARATIVELY: `@NiagaraAction(name="…", flags=Flags.OPERATOR)` = operator-invoke (256); OMIT the flag
  = admin-invoke (the DEFAULT); enforced by `BComponent.canInvoke` + the fox/box `PermissionException` — no permission
  code in the body. Reserve operator for low-privilege writes; config/emergency stay admin-only. Use
  `AccessController.doPrivileged` ONLY for a JVM permission (read a `BPassword`, `setDefault` an authenticator, set a
  system property) — NEVER wrap a Niagara RBAC check (AP-27). `[ev: corpus B776]`

### NEW section "Minimal module (copy-start)"
> grep-before-fold: `grep -n "minimal module\|scaffold" types/logic.md`
- The SMALLEST correct module = module.xml (header roster + 3-part dep FLOOR) + a non-empty `module.palette` (one `<p>`
  per component) + a prefixed non-empty `module.lexicon` + one `B<Comp> extends BComponent` with one
  `Flags.SUMMARY|Flags.OPERATOR` property + one `Flags.HIDDEN` engine action + one `Clock.Ticket` armed in
  `started()`+`atSteadyState()` and cancelled in `stopped()` + a SIGNED jar (major-52). This skeleton is verify-gate-green
  AND biting-check-green by construction. See B790 for the annotated file tree. `[ev: corpus B790]`

## → `toolbelt/` (new `scaffold-module.sh` — spec, for a future PR)
> grep-before-fold: `ls toolbelt/scaffold-module.sh 2>/dev/null`
- **`scaffold-module.sh <MOD> <vendor> <symbol>`** emits exactly the B790 skeleton. RED→GREEN fixture (resolves the D1
  campaign6 cut — buildable + test-anchored): the emitted skeleton must PASS `verify-module.sh` + the B787/B788 biting
  checks (ticket-without-stopped-cancel, lexicon-dup-keys, empty-palette), and a template mutation (drop the
  stopped()-cancel / empty the palette / dup a lexicon key) must FAIL them. `[ev: corpus B790]`

---

## → `types/dashboard.md`
> grep-before-fold: `grep -n "module.palette\|prefix\|bare-Type" types/dashboard.md`
- **`module.palette` convention:** bare-Type-minus-`B` instance names, plural category folders, `m="alias=module"`
  declared once, nested `<p>` to pre-seed ext/config child slots. `[ev: corpus B780]`
- **`module.lexicon` prefixing:** the lexicon is flat + MODULE-GLOBAL — PREFIX keys (`parent.child`, `Type.slot`) to
  dodge the B759 collision; a missing key renders raw camelCase via `toFriendly`. `[ev: corpus B780]`

### NEW subsection "Web-tier exemplars — where the Tridium pattern lives" (DUX-WEB1)
> grep-before-fold: `grep -n "pointer table\|web-tier exemplar\|BWebServlet" types/dashboard.md`
- **This section documents the web tier from OUR modules; the Tridium exemplar for each aspect is in the corpus —
  reach for it, don't re-derive:** servlet routing (`BWebServlet`/`BServletView`) → B29; hx views
  (`BHxView`/`BHxProfile`/`HxOp`) → B433; module `rc/` web resources + `module://<mod>/rc/…` ORDs → B5/B752;
  `@AgentOn` web agents (`BIJavaScript`+`JsInfo`) → B752/B421; Tridium-servlet CSRF (`CsrfGuard`/`CsrfProtectedFilter`)
  → B58 (vs our hand-rolled `X-Requested-With` guard, B763); JSON/REST response shaping → B16/B66 (analytics `/na`
  text/plain quirk), B361-B364 (report grid), B604 (nss `/nss/station/data` JSON), B509 (obix XML). `[ev: corpus B791]`

### NEW subsection "DashboardPan divergences from the Tridium web pattern" (DUX-WEB2)
> grep-before-fold: `grep -n "divergence\|X-Requested-With\|RBAC seam" types/dashboard.md`
- **CSRF via a hand-rolled `X-Requested-With` check inside the pure `route()`** (B763) rather than the framework
  `CsrfProtectedFilter`/`/rpc/*` `CsrfGuard` (B58) — a DELIBERATE, STRONGER-than-vendor divergence (vendor bajaux treat
  `requiredPermissions` as visibility-only and skip server RBAC, B752); keep it, note it as intentional. `[ev: corpus B791/B763]`
- **PUNCH-LIST: the RBAC decision is COLLAPSED into a Baja-bound helper** (DashboardPan) where chihuahua-ux keeps a
  pure-vs-Baja seam — re-split so the write-auth decision is Niagara-free/unit-testable (B762/B763 §763.4). `[ev: corpus B763]`

## → `types/wb-widgets.md`
> grep-before-fold: `grep -n "@AgentOn\|dual-surface" types/wb-widgets.md`
- **Dual-surface `@AgentOn` registration:** write `@NiagaraType(agent={@AgentOn(types={"mod:Type"},
  requiredPermissions="r")})` on the view/FE; Slot-o-Matic emits `<type><agent><on type=…/></agent></type>`;
  multi-type `types={…}` = one view over several source types. `[ev: corpus B780]`

---

## → `METHODOLOGY.md`
> grep-before-fold: `grep -n "runtimeProfile\|3-part\|vendorVersion\|EngineWatchdog\|lintable" METHODOLOGY.md`
- **module.xml profile + dependency conventions:** split by part `-rt`/`-ux`/`-wb`/`-se` (server); `-doc` is a SEPARATE
  `runtimeProfile="doc"` module, NEVER a part of a code module. A `<dependency>` `vendorVersion` is a 3-part Tridium
  FLOOR (`4.14.0`), distinct from the module's own 4-part build stamp (`4.14.0.162`). Header roster: author fills
  `vendor/vendorVersion/description/preferredSymbol/moduleName/runtimeProfile`; `bajaVersion` is const `"0"`. `[ev: corpus B784]`
- **Watchdog note (correct the "2s poll" assumption):** the systemMonitor cadence is an operator-editable
  `BTimeTrigger`/`BIntervalTriggerMode` (default 15 min), NOT a 2s poll; `BSysMonWorker` is a work-queue thread.
  Distinguish the framework-native `EngineWatchdog` (engine/process heartbeat) from author-level `BAbstractMonitor`
  threshold watchdogs. `[ev: corpus B775]`
- **Security-module permissions are INLINE (correction):** a security module grants permissions in a `module.xml`
  `<permissions><java-permissions type="station">` block, NOT a separate `module-permissions.xml`; jar-signing
  (`NIAGARA4.RSA/SF`) is mandatory for the privileged grants. `[ev: corpus B777]`
- **Conformance rules: lintable vs human-review (meta-delta):** a `verify-module.sh`/lint may HARD-FAIL only on a
  statically-decidable rule (lexicon dup-bare-keys, a `Clock.Ticket` field with no `stopped()`-cancel, an empty palette
  on a component module, type/slot lexicon coverage-%). Rules needing semantic judgment (an action's operator-vs-admin
  intent, a container's order-sensitivity, poll-vs-subscribe) stay a HUMAN-REVIEW checklist item — never a hard fail. `[ev: corpus B787/B788/B789]`
- **Human-review checklist (the advisory rules):** (a) does each non-HIDDEN command `@NiagaraAction` intend its
  operator/admin gating? (b) does an order-sensitive container guard its children with `isChildLegal`? (c) does any
  fixed-interval poll of a sibling slot want a `Subscriber` instead? `[ev: corpus B776/B789]`

---

## → `corpus-index.md`
> grep-before-fold: `grep -n "B762\|B785" corpus-index.md`
- Paste the 26-row delta from `campaign6-corpus-index-delta.md` verbatim (B762–B785 grouped by kit destination,
  P1/P2, one-line "read when"), including the "Idiom note" pointing types/logic.md's Author-side-SPIs section at
  B778/B782/B785. `[ev: corpus B762–B785]`

---

## Fold discipline (reminder for the PR7 writer)
For EACH bullet: run its grep-before-fold term against the kit; fold ONLY if absent (several framework MECHANISMS are
already in the kit — these are the AUTHOR-side residue). After folding, confirm the `[ev: corpus B<n>]` citation
appears in the core file (the content fold-audit gate). The idiom section goes FIRST in types/logic.md's author-side
material; the per-surface sections are its instances.
