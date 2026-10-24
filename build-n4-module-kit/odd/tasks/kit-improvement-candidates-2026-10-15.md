# build-n4-module kit — Improvement register (2026-10-15 retros)

**What this is.** One consolidated, prioritized register of every proposed kit delta in the 23 pending retros of 2026-10-15
(N4 developer course mining, the 4.15.6.20 devguide rounds 1–3, and the BTestNgStation testing-parity pilot). Nothing here is
folded. Every C-item is a proposal; folding follows BUILD-LOOP.md §7 exit (c) "promotion", one work unit at a time.
The 23 retros stay untouched (`review-status: pending`); this file only maps them.

**Precedent.** Same location and shape as `odd/tasks/kit-improvement-candidates-2026-09-19.md` (commit 6daf444).

**Count.** 23 retros, **168** source deltas (parsed rows = declared `**Delta count**` in every file; no file uses another
row shape). The HEAD commit message (ffcddad) says 178 because it sums the 14 devguide retros as 129; they hold 119
(10+5+10+10+10+10+9+10+9+10+6+4+6+10). 43 course + 119 devguide + 6 pilot = 168.
**Addendum (15ad06f).** A 24th retro, `review2-signing-build-route` (alias `rsb`, 6 deltas), was added after the
first consolidation and mapped into existing C-items (C39, C48, C51, C58). Current total: **24 retros, 174 deltas**.
**Addendum (2026-10-16 hands-on, T9a).** Five hands-on retros from executing the N4 Developer course
(`build-n4-module-kit/retros/2026-10-16-n4-course-handson-*.md`, 30 deltas) are mapped in **Appendix C**: 15 genuinely new
deltas open C81–C93; the other 15 confirm, refine or extend an existing C-item and are mapped to it. Appendix A and its
check script stay scoped to the 24 retros of 2026-10-15. Register total: **29 retros, 204 deltas, 93 C-items**.

**No-loss proof.** `bash build-n4-module-kit/odd/tasks/kit-improvement-candidates-2026-10-15.check.sh` (next to this file).
It extracts every delta ID from the 23 retros, checks each appears exactly once in Appendix A, that Appendix A holds no
unknown ID, and that every C-id it maps to exists as a heading below. Result recorded in § Proof.

**How to fold.** Each fold is a work-unit commit on a kit branch with `Retro: promotion (folds <stem> Δk, …)` and a
structural anchor (an `retros/INDEX.md` flip when a retro is fully folded, or a `BUILD-STATE.md` `open_issue` stamp for a
partial fold), and every folded Δ is cited in the kit text as `[ev: retro <stem> Δk]` or parked as
`[deferred: retro <stem> Δk]` so `sweep-fold-audit.sh --strict` stays green (BUILD-LOOP.md §7). A retro whose deltas are
split across several C-items is folded partially until its last C-item lands.

---

## Source aliases

| Alias | Retro (`build-n4-module-kit/retros/2026-10-15-<slug>.md`) | Δ |
|---|---|---|
| has | devguide415-horizons-authoring-security | 10 |
| lux | devguide415-lexicons-ux-i18n | 5 |
| nsto | devguide415-nav-subscriptions-tags-ords | 10 |
| cas | devguide415-r2-control-alarms-schedules | 10 |
| drp | devguide415-r2-drivers-protocols | 10 |
| pps | devguide415-r2-packaging-platform-signing | 10 |
| rcf | devguide415-r2-runtime-contracts-formats | 9 |
| sta | devguide415-r2-security-tags-analytics | 10 |
| shl | devguide415-r2-shipped-libraries | 9 |
| wuh | devguide415-r2-web-ui-hmi | 10 |
| r3di | devguide415-r3-drivers-integration | 6 |
| r3pec | devguide415-r3-packaging-extension-control | 4 |
| r3wsh | devguide415-r3-web-session-headers | 6 |
| rac | devguide415-runtime-api-corrections | 10 |
| cex | n4-course-as-exemplar-source | 4 |
| afl | n4-course-authoring-files-and-libraries | 5 |
| bdps | n4-course-build-deps-permissions-signing | 6 |
| ord | n4-course-drivers-bql-ords-files | 5 |
| lex | n4-course-lexicon-logging-testing | 5 |
| poh | n4-course-parity-observability-horizons | 6 |
| scc | n4-course-slots-components-contracts | 6 |
| tjd | n4-course-threading-jobs-doctrine | 6 |
| tpb | testing-parity-btestng | 6 |
| rsb | review2-signing-build-route | 6 |

Hands-on retros of 2026-10-16 (`build-n4-module-kit/retros/2026-10-16-n4-course-handson-<slug>.md`, mapped in Appendix C):

| Alias | Retro slug | Δ |
|---|---|---|
| bsw | build-signing-windows | 6 |
| ndh | ndriver | 8 |
| shs | services-history-threading | 6 |
| slo | slotomatic-authoring | 6 |
| hwf | workflow | 4 |

**r3 corrections not written into any retro** (reported by the round-3 run; recorded here as `r3-report#Rn`, mapped in
Appendix B, outside the 168 count):

| ID | Correction | Evidence | Also in a retro? |
|---|---|---|---|
| r3-report#R1 | `types/security.md:366-377` "DashboardPan omits X-Frame/nosniff/CSP" is wrong — `TridiumSecurityFilter` applies the station header providers to every `BWebServlet`; the snippet's CSP would REPLACE the station policy; `lint-servlet.sh` `api-response-headers` WARN is a false positive | vineflower `TridiumSecurityFilter.java:22-27`, vineflower `BJettyWebServer.java:1178-1180,1462-1468`, corpus B1349 C1 | yes, r3wsh Δ3 |
| r3-report#R2 | `types/logic-authoring.md:646` — `pingEnabled` is already ON by default (`<network>/monitor`) | `BPingMonitor.java:128`, corpus B1348 C1 | yes, r3di Δ2 |
| r3-report#R3 | shl Δ7(b): the non-driver-client gate is CLOSED by default (`HttpClientService.enableNonDriverClients=false`) | `BHttpClientService.java:231-252,270`, corpus B1346 §1346.2 | yes, r3di Δ4 |
| r3-report#R4 | shl Δ9: the jsonToolkit type-override hook is `overrideValueEncoding(BValue)`; `doOverride` is `final` in the 4.15 jar | `javap jsonToolkit-rt.jar!…/BTypeOverride`, corpus B1350 | yes, r3pec Δ2 |
| r3-report#R5 | sta Δ10: the `forceUpdate` flag reset applies only to R2 oBIX proxy points ("Niagara R2 to AX via oBIX") | `guides-clean/EngNotes/aForceUpdateAction.txt:2,13-15,27`, corpus B1353 §1353.2 row 8 | **no — only here** |
| r3-report#R6 | pps Δ1: `noRunningApp`/`noStation`=true stops ALL apps incl. Sedona; Workbench aborts on decline; a missing attribute means false | `DistributionManifest.java:87`, `PlatformInstallManager.java:353-356`, `CommitCommand.java:74-86`, corpus B1345 | yes, r3pec Δ1 |

---

## Summary

| Tier | Meaning | C-items | Source deltas |
|---|---|---|---|
| P0 | The kit currently teaches something wrong | 34 (C01–C30, C81–C84) | 56 + 13 |
| P1 | High-value missing rules that affect client modules | 28 (C31–C52, C85–C90) | 69 + 10 |
| P2 | New reference sections and templates | 26 (C53–C75, C91–C93) | 43 + 7 |
| P3 | Horizons and nice-to-have | 5 (C76–C80) | 6 |
| **Total** | | **93** | **174 + 30 = 204** |

The `+ n` figures are the 2026-10-16 hands-on deltas of Appendix C, counted by the C-item tier they map to.

**Rejected:** 0 whole deltas. 10 deltas carry a corrected, superseded or rejected PART (status `accepted-corrected` /
`superseded-in-part` / `partial-reject` in Appendix A): has Δ1 (poll-servlet route), has Δ2 (CSRF lint scope), shl Δ7
(gate default), shl Δ9 (`doOverride`), pps Δ1 (attribute default), cas Δ5 (Interstart min on/off), drp Δ4 (COV-only
scope), sta Δ10 (forceUpdate scope), rac Δ9 (BFormat step order), lex Δ1 (slot row, per lux Δ1). Each names the winning source in its C-item.

**Relation vocabulary.** primary = the source whose text the fold starts from; duplicate = says the same thing;
reinforces = agrees and adds evidence; extends = adds rules on the same topic; corrects = replaces part of another
source; supersedes = replaces another source's rule entirely.

---

## P0 fold plan — suggested work units (~400 changed lines each)

Order: highest copy-risk first (examples and API names that fail to compile or misbehave), then doctrine, then facts.

| WU | C-items | Files | Est. lines |
|---|---|---|---|
| WU-P0-1 Threads & jobs | C01, C02, C03 | `types/logic.md`, `types/math-control-libraries.md`, `types/logic-authoring.md`, `toolbelt/lint-timers.sh`, `tests/lint-timers*.bats` + `tests/fixtures/lint-timers/` | ~380 |
| WU-P0-2 Status & links | C04, C05, C08 | `types/logic.md` (BStatus, propagateFlags, LC3/LC4, link lifecycle), new bats row pinning the BStatus table | ~260 |
| WU-P0-3 Facets & formats | C06, C07 | `types/logic-authoring.md` (BFacets card), `types/logic.md` (BFormat card), new key-diff script or bats | ~350 |
| WU-P0-4 Actions, lexicon keys, palette | C09, C10, C11 | `types/actions.md`, `types/dashboard.md`, `types/value-types.md`, `types/structure.md`, `toolbelt/slot-coverage.sh`, `toolbelt/generate-wiring-map.sh`, `toolbelt/verify-module.sh`, `fixtures/MinimalPan/…/module.palette`, bats | ~420 |
| WU-P0-5 Drivers & subscriptions | C14, C15, C16, C20 | `types/driver-authoring.md`, `types/logic-authoring.md` (:9-24, :646) | ~250 |
| WU-P0-6 Nav, ORD, BQL, tags | C17, C18, C19, C21 | `types/structure.md` (nav), `types/logic-authoring.md` (ORD/BQL/hierarchy), `types/logic.md` (tag dictionary), `types/security.md:220`, `types/cloud-connector.md:206-217`, `toolbelt/lint-null-context-write.sh` + bats | ~420 |
| WU-P0-7 Packaging & authoring files | C23, C24, C25, C26, C29 | `types/distribution.md`, `types/structure.md`, `types/module-wiring.md`, `types/third-party-libraries.md`, `types/logic.md:185`, `toolbelt/lint-structure.sh` + bats | ~400 |
| WU-P0-8 Web headers & theme | C13, C30 | `types/security.md:360-380`, `types/theme.md`, `types/frontend-standard.md`, `toolbelt/lint-servlet.sh` (WARN→info) + bats | ~220 |
| WU-P0-9 Libraries, signing roster, alarm default, test route | C12, C22, C27, C28 | `types/moduleTest.md`, `BUILD-LOOP.md:162-165`, `build-verify.md`, `types/logic.md:324`, `types/cloud-connector.md`, `types/third-party-libraries.md`, `types/distribution.md:110-115`, `toolbelt/verify-module.sh` + bats | ~380 |

Each WU folds partial retros, so each commit stamps the owed halves in `BUILD-STATE.md` (§7 exit (c), partial promotion).

**Fold log.** WU-P0-1 docs folded on `docs/r8-t9b-p0-1` (2026-10-22): C01-C03 doc text in `types/logic.md`,
`types/math-control-libraries.md`, `types/logic-authoring.md`; the lint halves (tjd Δ2, shs Δ3 `jdk-thread` re-scope, the
`postAsync(`/`BJobs.` doc greps, the `isAlive()` loop WARN) are owed in `BUILD-STATE.md` kit `open_issues`.
WU-P0-3 docs folded on `docs/r8-t9b-p0-3` (2026-10-22): C06 BFacets card regenerated from the 4.15.6.20 `BFacets.java`
(34 constants) in `types/logic-authoring.md`, C07 BFormat card in `types/logic.md`; the key-diff check, the BFormat WARN
and the rac Δ9 cursor recipe (with C19, WU-P0-6) are owed.
WU-P0 lints, `docs/r8-t9b-p0-lints` (2026-10-22): `lint-timers.sh` `jdk-thread` re-scoped (FAIL only a
thread-modifying call without `MANAGE_EXECUTION`, multi-line class header) + new `job-cancel` WARN.
New `lint-bformat-call.sh` (BFormat `%time(<arg>)%` / `%user%` WARN).
New `check-facets-card.sh` + pinned 4.15.6.20 key list.
New `tests/kit-doc-api-names.bats` doc greps. Every owed lint half of C01-C03, C06, C07 is landed; only the rac Δ9
cursor recipe stays owed (WU-P0-6).

---

## P0 — the kit teaches something wrong

### C01 — JDK threads are allowed; only thread modification needs MANAGE_EXECUTION
- **Sources:** tjd#Δ1 (primary) · tjd#Δ2 extends (lint re-scope)
- **Winning text:** tjd Δ1 — Tridium's ladder (Clock / BWorker / BSimpleJob first; a named `Thread` or `ExecutorService` is production-grade); `shutdownNow()`/`interrupt()` need `MANAGE_EXECUTION` (type="station"). The kit's "denied by the SecurityManager, throws at runtime" is wrong.
- **Evidence:** course `Labs/Lab Instructions.pdf` p78-80; `Labs/Lab9 Multithreaded/module-permissions.xml`; user's live `training-rt` ran all three executors.
- **Target:** `types/logic.md:344`, `types/math-control-libraries.md:91,132`, `toolbelt/lint-timers.sh` jdk-thread check (+ bats, mutation; Go port: `go/internal/lint/rules/timers/threads.go:15-45`).
- **Lint-able:** yes — FAIL only when `shutdownNow(`/`interrupt(`/`setDaemon(`/`setPriority(` appear without `MANAGE_EXECUTION`; else WARN. **Fold type:** doc fix + lint.
- **Fold status:** FOLDED (WU-P0-1 doc + WU-P0 lints, 2026-10-22; tjd Δ1, Δ2, shs Δ3, Δ4). Open decision 2 answered by the
  Integrador: keep a blocking FAIL only for a thread-modifying call without `MANAGE_EXECUTION`.

### C02 — `BWorker.post(Runnable)` / `postAsync()` do not exist
- **Sources:** rac#Δ1 (primary). Thread-naming half reinforced by tjd Δ5 (C37).
- **Winning text:** rac Δ1 — enqueue on `getWorker()`'s queue, or `ExecutorUtil.newSingleThreadBackgroundExecutor(name, keepAlive, unit)` + `ExecutorUtil.shutdown(svc)` in `stopped()`; `ConcurrencyUtil.defaultThreadFactory(prefix)`.
- **Evidence:** `source/baja/javax/baja/util/BWorker.java:49-139`, `ExecutorUtil.java:43,89,122`, corpus B1304 §1304.3.
- **Target:** `types/logic.md:346`, `types/math-control-libraries.md:86`.
- **Lint-able:** yes — FAIL `postAsync(` in kit docs, WARN in rt sources. **Fold type:** doc fix + lint.
- **Fold status:** FOLDED (WU-P0-1 doc + `tests/kit-doc-api-names.bats`, 2026-10-22; rac Δ1). The rt-source WARN half is
  REJECTED. Note: `postAsync(Runnable)` is a legitimate name for a
  method a `BWorker` subclass declares itself (Tridium's course `BTrainingWorker`), so the rt-source WARN must target a call on a
  type that does not declare it, not the bare token.

### C03 — Background jobs: no `BJobs`; `BSimpleJob` owns its thread; the job contract
- **Sources:** rac#Δ2 (primary) · tjd#Δ6 duplicate on the pool claim + extends (job contract, test idiom)
- **Winning text:** `new BRunnableJob(() -> …).submit(cx)`; `BSimpleJob` runs its own `JobThread` (no ForkJoinPool, no `RejectedExecutionException`); loop checks `isAlive()` → `JobCancelException`; `progress()`/`heartbeat()`; `log().message(module,key,args)`; snapshot service properties in the constructor.
- **Evidence:** `BSimpleJob.java:51-56`, `BRunnableJob.java:41`, corpus B1238 §1238.8, B1304 §1304.2.
- **Target:** `types/logic-authoring.md:216-233` § Background jobs.
- **Lint-able:** yes — doc grep FAIL `BJobs.`; WARN a `BSimpleJob.run(` loop with no `isAlive()`. **Fold type:** doc fix + lint.
- **Fold status:** FOLDED (WU-P0-1 doc + WU-P0 lints, 2026-10-22; rac Δ2, tjd Δ6, shs Δ6): `BJobs.` doc grep,
  `lint-timers.sh` `job-cancel` WARN.

### C04 — BStatus bit table is shifted from STALE on and misses ALARM
- **Sources:** rac#Δ4 (primary)
- **Winning text:** DISABLED 0x01, FAULT 0x02, DOWN 0x04, ALARM 0x08, STALE 0x10, OVERRIDDEN 0x20, NULL 0x40, UNACKED_ALARM 0x80; `isValid()` passes ALARM, OVERRIDDEN, UNACKED.
- **Evidence:** `source/baja/javax/baja/status/BStatus.java:758-786`; corpus B736 already right (transcription error).
- **Target:** `types/logic.md:233-243`.
- **Lint-able:** yes — bats row pins the table to the constants. **Fold type:** doc fix + test pin.
- **Fold status:** FOLDED (WU-P0-2, `docs/r8-t9e-status-links`, 2026-10-22; rac Δ4): table = `BStatus` constants, pinned by
  `tests/kit-api-pins.bats` KP1.

### C05 — `propagateFlags` recipe uses a non-existent `BStatus.and()`; kitControl status families
- **Sources:** rac#Δ5 (primary) · rcf#Δ7 extends (Select/Switch hold + `nullStatus` default; math `ok` default masks FAULT/DOWN/STALE)
- **Winning text:** `BStatus.make(agg.getBits() & getPropagateFlags().getBits())`; set `fault,down,stale,disabled` downstream of probes; an invalid selector HOLDS `out` + invalid facet.
- **Evidence:** `kitControl-rt/…/logic/BLogic.java:113,169`, `util/BSwitch.java:127,317,336`, corpus B1314 §1314.2, B1319 §1319.1.
- **Target:** `types/logic.md:252-256`, `:335` (LC3/LC4).
- **Lint-able:** yes — doc grep FAIL `getPropagateFlags().and(`; partial bog check for default flags behind proxy points. **Fold type:** doc fix + lint.
- **Fold status:** FOLDED doc + doc grep (WU-P0-2, 2026-10-22; rac Δ5, rcf Δ7): recipe and status families in `types/logic.md`,
  `tests/kit-api-pins.bats` KP2 is the doc grep. OWED: the bog check for default flags behind proxy points.

### C06 — BFacets key card names `makeDouble` and ~17 non-constant keys
- **Sources:** rac#Δ8 (primary) · bdps#Δ6 extends (absorbed: `multiLine`, `targetType "baja:IDirectory"`, `@Facet` expression syntax)
- **Winning text:** regenerate from `BFacets.java` (32 constants; 4.15u2 `showHours/showMinutes`); `unitConversion` is a BUnitConversion enum; `make(...)` takes `BIDataValue`. `multiLine` is kept only if it is one of the constants (course code uses `BFacets.MULTI_LINE`, so expected yes — the generator decides).
- **Evidence:** `docSource/…/javax/baja/sys/BFacets.java:155-159`, corpus B1310 §1310.3 C1; course `Labs/Lab10 Jobs/BHistoriesToCsvService.java`.
- **Target:** `types/logic-authoring.md:489-534`.
- **Lint-able:** yes — script diffs card keys against the constant list. **Fold type:** doc fix + script.
- **Fold status:** doc FOLDED (WU-P0-3, 2026-10-22; rac Δ8, bdps Δ6). 4.15.6.20 has 34 constants, so `multiLine` is one
  (`BFacets.MULTI_LINE`). FOLDED with `toolbelt/check-facets-card.sh` + `tests/check-facets-card.bats`.

### C07 — BFormat card gets `?`, `time()`, `substring` and the lookup order wrong
- **Sources:** rac#Δ9 (primary; cursor half cross-ref C19) · rcf#Δ6 extends (line-level facts)
- **Conflict:** rac Δ9 "reflection order has 7 steps" vs rcf Δ6 "slot → six reflect steps → `BIFormatPropertyHandler`". **Winner rcf Δ6** (line-cited `BFormat.java:62-72,372,443,666,744-750,755,1095-1106`, corpus B1317 §1317.3, later round).
- **Winning text:** `?` = fallback script; `%%` literal; `time()`/`user()` take no argument (chain after `time()`); `substring(i)`, `(-n)`, `(a,b)` on the previous value.
- **Target:** `types/logic.md:399-418`; cursor recipe → `types/logic-authoring.md:190-200` (with C19).
- **Lint-able:** yes — WARN `%time([^)]`, `%user%`. **Fold type:** doc fix + lint.
- **Fold status:** BFormat card FOLDED (WU-P0-3, 2026-10-22; rac Δ9 card half, rcf Δ6); `lint-bformat-call.sh` LANDED (WU-P0
  lints); the cursor recipe (rac Δ9, with C19) FOLDED in WU-P0-6 (`docs/r8-t9g-nav`, 2026-10-22).

### C08 — Link rules: programmatic links are never `checkLink`ed; `BLink` 4th arg is `enabled`
- **Sources:** rac#Δ10 (primary) · rcf#Δ3 extends (call `target.checkLink(...)` and require `isValid()` before adding)
- **Winning text:** `linkTo()` = TRANSIENT direct link, no check; `add(name, BLink)`/`makeLink()` never run `checkLink`; `new BLink(BOrd,…, true)` → `true` is `enabled`; use `getHandleOrd()`; past licence link capacity `fatalFault` → read back `isActive() && !isFatalFault()`.
- **Evidence:** `BComponent.java:1604-1612,1778-1785`, `BLink.java:270`, `ComponentSlotMap.java:1683-1686`, corpus B1309 §1309.2, B1316 §1316.3.
- **Target:** `types/logic.md:111-143`.
- **Lint-able:** yes — WARN `BLink`/`linkTo(` without `checkLink(`; `getOrdInSession` beside `new BLink`. **Fold type:** doc fix + lint.
- **Fold status:** FOLDED doc (WU-P0-2, 2026-10-22; rac Δ10, rcf Δ3): `types/logic.md` § Link lifecycle. OWED: the rt-source
  WARN lint (outside the unit's allowed surface), stamped in `BUILD-STATE.md`.

### C09 — A mis-named frozen action handler is a type-load error, not a silent no-op
- **Sources:** scc#Δ1 (primary)
- **Winning text:** missing/mis-signed `do` method on `@NiagaraAction` → "Missing action do method" at load; if both `doX()` and `doX(Context)` exist the no-Context one wins — declare one, take `Context`.
- **Evidence:** corpus B1231 §1231.4 (`ComplexIntrospector`); course `Slides/2 Slots.pdf`.
- **Target:** `types/actions.md:24-26`.
- **Lint-able:** yes — WARN both `do<X>()` and `do<X>(…Context)`. **Fold type:** doc fix + lint.
- **Fold status:** FOLDED (WU-P0-4, `docs/r8-t9g-actions`, 2026-10-22; scc Δ1): `types/actions.md` § 1 (type-load errors,
  public handler taking `Context`, one handler); the pair WARN is owed as Go lint G1 (new checks go in Go). Premise checked against
  `ComplexIntrospector.java:316-343`; the kit's own example handler was package-private (also a type-load error), fixed.

### C10 — Lexicon keys: slot = bare slot name, type = `<Type>.displayName`, enum = bare tag
- **Sources:** lux#Δ1 (primary, corrects the kit's `Type.slot` advice and its checkers) · lex#Δ1 extends (`<Type>.displayName`, `%lexicon()%` in facets; lux Δ1 refines its slot row)
- **Winning text:** `Type.slot` keys are never read; colliding slot names → `displayNames` `BNameMap` or `getDisplayName(Slot,Context)`; a bare `<Type>=` does nothing for the type.
- **Evidence:** `NSlot.java:42`, `EnumType.java:50`, devguide `localization.txt:106-108,135-136`, corpus B1305 §1305.2 C1, B759 §759.1, B1236 §1236.3.
- **Target:** `types/dashboard.md:203`, `types/value-types.md:51-52`, `types/structure.md` L4, `toolbelt/slot-coverage.sh:23,39-41`, `toolbelt/generate-wiring-map.sh:20`.
- **Lint-able:** yes — `slot-coverage.sh` WARN dotted-only match and bare `<Type>=`; require `<Type>.displayName`. **Fold type:** doc fix + script fix.
- **Fold status:** FOLDED (WU-P0-4, 2026-10-22; lux Δ1, lex Δ1): `types/structure.md` § Lexicon key catalogue,
  `types/dashboard.md` (wiring-map bullet + key bullet, now :239 not :203), `types/value-types.md`; `slot-coverage.sh`
  parse = `<Type>.displayName` only + WARN bare `<Type>=`, per-slot = bare keys + `DOTTED` row; `generate-wiring-map.sh`
  bare key only (:20-22, :145). `%lexicon()%` is a BFormat call (`BFormat.java:452`), not `Lexicon.java:466-495` as
  B1305 row 11 grouped it. The `generate-wiring-map.sh` display-names comment (:132) reworded after T8c.

### C11 — Palette example uses class names and attribute overrides; fixture root violates the kit's rule
- **Sources:** afl#Δ3 (primary)
- **Winning text:** `<p m="b=baja" t="b:UnrestrictedFolder">` root; module prefix declared once, `t="<x>:<Type>"`; overrides as child `<p n v/>`; only draggable component types; fixture root → `b:UnrestrictedFolder`.
- **Evidence:** `Niagara-4.15.6.20/modules/kitControl-rt.jar!module.palette:133-141`; course `Slide Code/…/module.palette`; `fixtures/MinimalPan/MinimalPan/MinimalPan-rt/module.palette:3`.
- **Target:** `types/structure.md:133-180`, fixture palette, `toolbelt/verify-module.sh`.
- **Lint-able:** yes — FAIL `t="` with a `.`, undeclared prefix; WARN any `b:Folder`. **Fold type:** doc fix + template + lint.
- **Fold status:** FOLDED (WU-P0-4, 2026-10-22; afl Δ3): `types/structure.md` § Palette format card + example + key
  rules; `verify-module.sh` `palette_scan` (`palette-type` FAIL, `palette-root` WARN on every baja Folder);
  `fixtures/MinimalPan` root `b:UnrestrictedFolder`. Premise confirmed in `ValueDocDecoder.java:1071-1073,1091`; the
  undeclared-symbol FAIL also catches a root without `m="b=baja"`. OWED: `fixtures/MinimalDash` palettes (two `b:Folder`
  roots; outside the unit's surface).

### C12 — `niagaraTest` is not "WSL finds 0 tests → Workbench only"; native Windows route works
- **Sources:** tpb#Δ1 (primary) · tpb#Δ3 extends (signing prerequisite: CA-chained test-jar signer, else BUILD SUCCESSFUL with 0 tests)
- **Winning text:** `cmd.exe /c "cd /d <win copy> && gradlew.bat -PniagaraPluginVersion=<from etc/m2> :<part>:clean :<part>:niagaraTest"`; the 0-test result is specific to WSL + plugin 7.6.17; Workbench Run Tests = fallback; back up `<niagara_home>/modules/<part>.jar`.
- **Evidence:** commit 337ee6c (6/6, five mutations bite); course Lab 7 9/9; `Could not validate cert chain` → `invalid test target(s)`.
- **Target:** `types/moduleTest.md:10-13,156,385-386`, `BUILD-LOOP.md:162-165`, `build-verify.md` § Signing.
- **Lint-able:** no (doctrine; C40 adds the runner). **Fold type:** doc fix.
- **Fold status:** FOLDED (WU-P0-9, `docs/r8-t9i-libs`, 2026-10-23; tpb Δ1, Δ3): `types/moduleTest.md` NRE note,
  checklist step 4, new § Running from WSL (cmd.exe on a `%TEMP%` copy, `:clean` first, count the tests, the
  self-signed test-jar cause, back up `modules/<part>.jar`), Known gotchas; `BUILD-LOOP.md` § moduleTest;
  `build-verify.md` § Unit tests in WSL. The runner (tpb Δ2) stays with C40. Pin KL4.

### C13 — Security headers: station headers already apply to module servlets
- **Sources:** r3wsh#Δ3 (primary, corrects the kit) · wuh#Δ5 extends (Hx needs X-Frame Sameorigin + default CSP) · r3wsh#Δ5 extends (embed via CSP `frame-ancestors` on the station provider, never X-Frame `Any`) · r3-report#R1
- **Winning text:** remove "DashboardPan omits X-Frame/nosniff/CSP — gaps ODA2-G1/G2" and the `setHeader` snippet (its CSP replaces the station policy); set a header only to deviate deliberately; commissioning `curl -I` check.
- **Evidence:** vineflower `BJettyWebServer.java:1178-1180,1462-1468`, vineflower `TridiumSecurityFilter.java:22-27`, corpus B1349 C1, B1328 row 9.
- **Target:** `types/security.md:360-380`; `toolbelt/lint-servlet.sh` `api-response-headers` WARN → info.
- **Lint-able:** yes (downgrade). **Fold type:** doc fix + lint.
- **Fold status:** FOLDED (WU-P0-8, `docs/r8-t9i-headers`, 2026-10-23; r3wsh Δ3, Δ5, wuh Δ5, r3-report#R1):
  `types/security.md` § 7 rewritten (station providers reach every servlet, set a header only to deviate, embed with
  CSP `frame-ancestors`, Hx needs the defaults); `lint-servlet.sh` `api-response-headers` WARN → INFO (no `--strict`
  escalation), `report-module.sh` counts an INFO-only servlet as PASS (RM16b), `commissioning-verify.sh` probe text.
  Premise refined: the X-Frame provider class default is `deny` (vineflower `BXFrameOptionsHeaderProvider.java:34`); only the
  guide and vineflower `BWebService.java:255` say `Sameorigin`, so the commissioning probe reads the live value. Pins KH1-KH4.

### C14 — `pingEnabled` is ON by default
- **Sources:** r3di#Δ2 (primary) · r3-report#R2
- **Winning text:** "never disable `<network>/monitor/pingEnabled`; tune `pingFrequency`/`numRetriesUntilPingFail`".
- **Evidence:** `driver-rt/…/ping/BPingMonitor.java:128`, corpus B1348 C1.
- **Target:** `types/logic-authoring.md:646`.
- **Lint-able:** yes — doc grep for "enable `pingEnabled`". **Fold type:** doc fix.
- **Fold status:** FOLDED (WU-P0-5, `docs/r8-t9e-drivers`, 2026-10-22; r3di Δ2): `types/logic-authoring.md` ping line;
  doc grep = `tests/kit-driver-pins.bats` KD2.

### C15 — Proxy-point example inverts `readUnsubscribed`
- **Sources:** nsto#Δ1 (primary)
- **Winning text:** `readSubscribed` = start polling/register COV; `readUnsubscribed` = stop/unregister; neither blocks; report via `readOk/readFail`.
- **Evidence:** `driver-rt/javax/baja/driver/point/BProxyExt.java:886-897`, devguide `driverPoint.txt:64-67`, corpus B1306 §1306.2 C1.
- **Target:** `types/driver-authoring.md:54-70` (and `:209`).
- **Lint-able:** yes — WARN `send`/`poll`/`post` in `readUnsubscribed`. **Fold type:** doc fix + lint.
- **Fold status:** FOLDED doc (WU-P0-5, 2026-10-22; nsto Δ1): `types/driver-authoring.md` § 2, doc pin KD4. OWED: the
  rt-source WARN lint.

### C16 — Tuning facts: `staleTime=0` is not "off"; `pollEnabled` lives on the poll service
- **Sources:** drp#Δ3 (primary, staleTime) · drp#Δ8 (primary, pollEnabled; extends with ≤75 % busy, Slow/Normal/Fast policies, no exts on proxy points)
- **Evidence:** `guides-clean/Bacnet/bacnet-BacnetTuningPolicy.txt:66`; `driver-rt/…/util/BAbstractPollService.java:64`; corpus B1327 §1327.2 C1, B1337 §1337.2 C1-C2.
- **Target:** `types/driver-authoring.md:121-132,205-213`.
- **Lint-able:** yes for `:213`; partial otherwise. **Fold type:** doc fix.
- **Fold status:** FOLDED (WU-P0-5, 2026-10-22; drp Δ3, Δ8; KD3 pins `:213`). The staleTime half is CORRECTED, not
  applied: B1365 §1365.4 and B1419 §1419.4 show 0 = timer off in code; the guide's stale-on-unsubscribe is marked unimplemented.

### C17 — Nav files: `version="1.0"` mandatory, cache re-parses, nothing auto-discovers `.nav`
- **Sources:** nsto#Δ5 (primary) · rcf#Δ5 extends (delete "Workbench and the station discover `.nav` files"; only `BUser.navFile` / mobile nav file)
- **Evidence:** `NavFileDecoder.java:56,71,130-133`, `BUser.java:1030`, corpus B1307 §1307.2, B1317 §1317.2; **B1353 §1353.1** upgrades "no rc/nav convention" to `[CERT]` (0 `.nav` in 728 jars).
- **Target:** `types/structure.md:253-287`.
- **Lint-able:** yes — `<nav>` without `version=`; FAIL phrase "discover `.nav` files". **Fold type:** doc fix + lint.
- **Fold status:** FOLDED (WU-P0-6, `docs/r8-t9g-nav`, 2026-10-22; nsto Δ5, rcf Δ5): `types/structure.md` § Nav files;
  doc pins `tests/kit-nav-tag-pins.bats` KN1/KN2. OWED: Go lint G5 `nav-version` (spec in `BUILD-STATE.md`).

### C18 — ORD/nav basics: `BOrd.make` only wraps; `nav:` needs a base; `getNavChildren` is permission-blind
- **Sources:** nsto#Δ7 (primary, ORD basics + unmounted BQL test) · nsto#Δ6 (primary, nav safety) · ord#Δ4 extends (narrowest base, `resolve()`+`canRead()`, custom scheme `UnresolvedException`)
- **Evidence:** `BOrd.java:97-101,375-380`, `BINavNode.java:96-98`, `BNavContainer.java:364`, devguide `bql.txt:453-460`, corpus B1307 §1307.1, B1308 §1308.2, B1237 §1237.3.
- **Target:** `types/logic-authoring.md:37-39,76,120,142`, `types/moduleTest.md`, `types/security.md`.
- **Lint-able:** yes — WARN `getNavChildren(` without `filter(` in servlet/ux; `nav:` without `root:|`. **Fold type:** doc fix + lint.
- **Fold status:** FOLDED doc (WU-P0-6, 2026-10-22; nsto Δ6, Δ7, ord Δ4): `types/logic-authoring.md` (nav safety,
  call-site recipe, `h:` scope, `nav:` base, custom scheme), `types/moduleTest.md` (unmounted BQL); doc pins KN3/KN5.
  OWED: Go lint G6 `nav-children-filter` (spec in `BUILD-STATE.md`). The `types/security.md`
  target needed no nav text: the permission rule lives with the walk.

### C19 — BQL consumer example omits the Context and closes the cursor by hand
- **Sources:** ord#Δ1 (primary). Cursor recipe of rac Δ9 (C07) folds here.
- **Winning text:** `.get(base, cx)` with the caller's Context; `try (TableCursor<?> c = t.cursor())`; `|cell:0,0`; `newCopy()` on kept records; BQL never subscribes (`lease` / `Subscriber`); remote ORDs need a `BFoxProxySession`.
- **Evidence:** corpus B1237 §1237.7 (BQL Reference.pdf, deck 8), course `Slide Code/…/ords/BPointsQuery.java`.
- **Target:** `types/logic-authoring.md:191-204`, `toolbelt/lint-null-context-write.sh` (or sibling).
- **Lint-able:** yes — flag one-arg `get(` or `null` Context on a `bql:` ORD in `-rt`. **Fold type:** doc fix + lint.
- **Fold status:** FOLDED (WU-P0-6, 2026-10-22; ord Δ1, rac Δ9 cursor half): `types/logic-authoring.md` § BQL from code
  (+ digest D3-31 `BFoxProxySession`, spoken claim); `lint-null-context-write.sh` bql rule (also no-arg `get()` and
  `resolve(base)`; -wb skipped) with a literal-aware comment stripper; to be re-done as Go lint G4 `bql-context`.

### C20 — Subscription semantics and virtual ORDs are described wrongly
- **Sources:** nsto#Δ2 (primary: subscribed = permanent ‖ subscribers ‖ knobs-while-running; leases; SELF_EVENTS = 11–16,19,20) · nsto#Δ3 (primary: virtual ORD = gateway ORD + `|virtual:/a/b`; `SlotPath.escape`; cache properties)
- **Evidence:** `BComponent.java:408-414`, `BComponentSpace.java:172-174`, `BComponentEventMask.java:319-327`, devguide `virtualComponents.txt:252-254`, corpus B1306.
- **Target:** `types/logic-authoring.md:9-24`, `types/driver-authoring.md:234-240`.
- **Lint-able:** partial (`addVirtualSlot` without `SlotPath.escape`). **Fold type:** doc fix.
- **Fold status:** FOLDED doc (WU-P0-5, 2026-10-22; nsto Δ2, Δ3). OWED: the `addVirtualSlot` grep.

### C21 — Tag-dictionary recipe names APIs that do not exist; `hierarchy:` is client-only; dictionaries are licensed
- **Sources:** nsto#Δ9 (primary, recipe rewrite) · nsto#Δ10 extends (tags-in-code card, `hierarchy:` client-only) · sta#Δ5 extends (licence: 2 dictionaries, `n` takes one) · sta#Δ7 reinforces (ScopedTagRule, implied index off, heap-only caches)
- **Evidence:** `BSmartTagDictionary.java:71`, `BTagDictionary.java:144,200`, `BNamespace.java:19` (`@Deprecated`), devguide `neql.txt:18,28-46`, `hierarchy.txt:63-66`, `Tagging/LicenseRequirements-474FC1B4.txt:20`, corpus B1311 §1311.2, B1329.
- **Target:** `types/logic.md:359-376`, `types/logic-authoring.md:148-154,181`, `types/security.md:220`, `types/cloud-connector.md:206-217`.
- **Lint-able:** yes — doc grep FAIL `getRules()`, `SmartTagRule`, `tag::`, `tagInfoList.add`; grep `BOrd.make("hierarchy:` in `-rt`. **Fold type:** doc rewrite + lint.
- **Fold status:** FOLDED doc + doc grep (WU-P0-6, 2026-10-22; nsto Δ9, Δ10, sta Δ5, Δ7): `types/logic.md` § Ship a tag
  dictionary + § Tags in code, `types/logic-authoring.md` § `hierarchy:`, `types/security.md` § 4, `types/cloud-connector.md`
  § 6; doc pin KN4. OWED: Go lint G7 `hierarchy-ord` (spec in `BUILD-STATE.md`).

### C22 — `ackRequired` default is version-dependent; `timeDelay` never debounces FAULT; stale points never alarm
- **Sources:** cas#Δ1 (primary)
- **Evidence:** `alarm-rt/javax/baja/alarm/BAlarmClass.java:178` (4.15.6.20 = toOffnormal|toFault|toNormal), `BAlarmSourceExt.java:1462,1634`, corpus B1324 §1324.1.
- **Target:** `types/logic.md:324-325`.
- **Lint-able:** partial. **Fold type:** doc fix.
- **Fold status:** FOLDED (WU-P0-9, 2026-10-23; cas Δ1): `types/logic.md` alarm policy items 3-4. **Premise
  corrected:** the default is NOT version-dependent: vineflower 4.14.0.162 `BAlarmClass.java:115` (`make(7)`) and the
  devguide 4.15.6.20 source `:178` both say toOffnormal|toFault|toNormal; the kit's "without toNormal" was wrong in
  both. `timeDelay` offnormal-only and the stale/disabled skip folded as written. Pin KL3; owed Go lint G16.

### C23 — Dist attribute table inverts `noStation`
- **Sources:** pps#Δ1 (primary) · r3pec#Δ1 corrects part (default) · r3-report#R6
- **Conflict:** pps Δ1 "devguide defaults `noRunningApp` true, code reads false" vs r3pec Δ1 "omitting = false, enforced: stops ALL apps incl. Sedona; Workbench aborts on decline". **Winner r3pec Δ1** (parser + installer + CommitCommand). Keep pps Δ1's "always write both attributes" as practice.
- **Evidence:** `DistributionManifest.java:87,130`, `PlatformInstallManager.java:353-356`, `CommitCommand.java:74-86`, 22/26 shipped dists, corpus B1318 §1318.3, B1345.
- **Target:** `types/distribution.md:35-45`.
- **Lint-able:** yes — FAIL "installs without stopping"; WARN dist.xml without explicit `reboot`/`noStation`. **Fold type:** doc fix + lint.
- **Fold status:** FOLDED (WU-P0-7, `docs/r8-t9i-packaging`, 2026-10-23; pps Δ1, r3pec Δ1): `types/distribution.md` § 2
  table (`noStation` = stop every app, missing = false, `noRunningStation`, `reboot` default false,
  `updateDaemonBinaries` no reboot) + "always write both"; § 3 gotcha. Pin KP1. The dist.xml check is owed as Go lint G8.

### C24 — `-se` means "depends on Java SE APIs" and runs on the Windows Supervisor only
- **Sources:** has#Δ8 (primary, meaning + part dependency matrix) · pps#Δ8 extends (JACE never loads `-se`)
- **Evidence:** devguide `modules.txt:72-76`, `Platform/aPlat4PlatformAdminConfigureRuntimeProfiles.txt:49-51`, corpus B1313 §1313.3 C3, B1341 §1341.2 C1.
- **Target:** `types/structure.md:300` + § Layout.
- **Lint-able:** yes — parse cross-part `api(...)`; WARN `runtimeProfile=se` on a JACE module. **Fold type:** doc fix + lint.
- **Fold status:** FOLDED (WU-P0-7, 2026-10-23; has Δ8, pps Δ8): `types/structure.md` § Layout (profile meanings, part
  dependency matrix from `modules.txt:148-155`) and § The `-se` runtime profile; the JACE premise was in five places
  (load table, AWT trap, `-wb` dependency trap, JACE limits, `METHODOLOGY.md`), all corrected. The JACE row is
  doc-proven (`ConfigureRuntimeProfiles.txt:30`), not run on a JACE. Pin KP2; owed Go lint G9 `part-deps`.

### C25 — `module-include.xml` holds only `<type>` lines; stale lines are never removed
- **Sources:** has#Δ6 (primary) · afl#Δ5 extends (processor appends, never removes → SEVERE "Missing Sys.loadType()"; test types only in `moduleTest-include.xml`)
- **Evidence:** devguide `upgradingBuild.txt:121`, corpus B1313 §1313.3 C2, B1233 §1233.4.
- **Target:** `types/structure.md:8,338`, `types/module-wiring.md` §2, `toolbelt/verify-module.sh`.
- **Lint-able:** yes — WARN `<defs>`/`<lexicons>`/`<moduleParts>` in it; FAIL dangling `<type class>`. **Fold type:** doc fix + lint.
- **Fold status:** FOLDED (WU-P0-7, 2026-10-23; has Δ6, afl Δ5): `types/module-wiring.md` § 2 (only `<types>`,
  `moduleManifest {}`, append-only processor, SEVERE `Missing Sys.loadType()` at `Builder.java:776`, test types in
  `moduleTest-include.xml`); `types/structure.md` § Layout + `-se` sibling step 3 no longer put `<moduleParts>` in it.
  Pin KP5; owed Go lint G10 `module-include` (the `verify-module.sh` target is left to it).

### C26 — Authoring recipe: doc part allowed, `help.guide.base` optional, `uberjar` first
- **Sources:** has#Δ7 (primary) · afl#Δ1 reinforces/duplicate on `uberjar` + extends (Path 0, `testUberjar`, library→permission table, POJO DTO, Java-8 pins)
- **Winning text:** drop "NEVER a doc part"; omit `help.guide.base` (never `module://docUser/doc` in a vendor module); `uberjar`/`testUberjar` primary, Shadow only for relocation.
- **Evidence:** devguide `deployingHelp.txt:10,29-31`, `modules.txt:122`, `build.txt:250`, `upgradingBuild.txt:101`, course `Lab Instructions.pdf` p59-63.
- **Target:** `types/structure.md:9,176-250`, `types/third-party-libraries.md:38-80`.
- **Lint-able:** yes — FAIL `help.guide.base=module://docUser`; WARN `shadowJar`. **Fold type:** doc fix + lint.
- **Fold status:** FOLDED (WU-P0-7, 2026-10-23; has Δ7, afl Δ1 uberjar half): doc part allowed, `help.guide.base`
  optional (`deployingHelp.txt:8,28`) in `types/structure.md`; `types/third-party-libraries.md` Path 0 `uberjar` /
  `testUberjar` first (`build.txt:187-193`), Shadow demoted to the relocation fallback. Pins KP3, KP4; owed Go lint
  G11. afl Δ1's library→permission table and Java-8 pins stay with C42.

### C27 — Wrong library facts: okhttp location, Gson version, MQTT module name
- **Sources:** shl#Δ2 (primary, okhttp ships in `bin/ext`, `nre("com.squareup.okhttp3:okhttp")`) · shl#Δ5 (Gson 2.13.2, Jackson 2.22.1, `abstractMqttDriver`) · drp#Δ1 duplicate on the module name + extends (MQTT native first, `tridium:mqtt`, 25 devices on a JACE)
- **Evidence:** `unzip -l modules/net-rt.jar` → 0 okhttp; wrapper `pom.properties`; `BAbstractMqttDriverNetwork.java:64,69,125`; corpus B1320 §1320.3, B1321 §1321.2, B1323.
- **Target:** `types/cloud-connector.md:61-63`, `types/third-party-libraries.md:30,143,159-160,167,171,174`.
- **Lint-able:** yes — grep `okhttp3.*net-rt`, `2\.9\.0`, `mqtt-driver`. **Fold type:** doc fix + lint.
- **Fold status:** FOLDED (WU-P0-9, 2026-10-23; shl Δ2, Δ5, drp Δ1): `types/cloud-connector.md` § 2.1 (okhttp in
  `bin/ext`, `nre(...)`; MQTT native first) and `types/third-party-libraries.md` (Gson 2.13.2, `abstractMqttDriver`
  with Paho 1.2.5 / Jackson 2.22.1 / JJWT 0.12.7, OkHttp row). Verified on the 4.15.6.20 install (`bin/ext`,
  `pom.properties`); the MQTT guide lines moved to `MQTTRequirements-74179811.txt:11,17`. Pins KL1, KL2. The Commons
  Lang `nre(...)` half of shl Δ2 is not folded (no kit text names it).

### C28 — Signing-required permission roster names three groups; there are six
- **Sources:** has#Δ4 (primary) · bdps#Δ2 reinforces + extends (roster with triggers: NETWORK_COMMUNICATION, REFLECTION, MANAGE_EXECUTION)
- **Winning text:** six groups require a signed module (REFLECTION, ACCESS_CLASS, MBEAN_PERMISSION, THIRD_PARTY_PERMISSION, PROTECTION_DOMAIN, HSM_SIGNING); trust = any CA/internal CA/self-signed cert in the user trust store (self-signed rejected in `high`).
- **Open verification:** the 4.14 factory SKIPS an unsigned group with a WARNING; the devguide says the station halts — verify on a 4.15 target before writing the outcome as fact.
- **Evidence:** `com/tridium/security/*PermissionGroup.java` `requiresSignature()`, `NiagaraPermissionGroupFactory.java:196-204`, devguide `security/codeSigning.txt:123-129,163`, corpus B1312 §1312.3, B1235 §1235.5.
- **Target:** `build-verify.md:132`, `types/distribution.md:110-115`, `types/third-party-libraries.md:148-151`, `types/structure.md` § module-permissions.xml, `toolbelt/verify-module.sh`.
- **Lint-able:** yes — FAIL unsigned jar naming one of the six; source cross-check (Jackson→REFLECTION, `shutdownNow`→MANAGE_EXECUTION, sockets→NETWORK_COMMUNICATION). **Fold type:** doc fix + lint.
- **Fold status:** FOLDED (WU-P0-9, 2026-10-23; has Δ4, bdps Δ2): `build-verify.md` § Signing per deploy target
  (six groups, triggers, three trust routes), `types/distribution.md` § Hard signing gate, `types/third-party-libraries.md`
  signing note, `types/structure.md` module-permissions bullet. **Open verification closed at the factory layer:** code wins over
  the devguide's "halt" (`codeSigning.txt:185`): organized/ vineflower 4.14.0.162 `NiagaraPermissionGroupFactory.java:196-204`
  (decompiled `:243-248`, procyon `:215-220`) and javap of `baja.jar` 4.15.6.20 and iC-Niagara 4.13.2.18 (throw at 537
  inside the 134-547 try, WARNING "Skipping element") all skip; other builds not verified; the cert-chain layer stays open (niagara-research B1312-G1). The
  removed kit claim "fails to load unsigned (`ValidationException`) even in LOW" had no code behind it. Pin KL5; owed Go
  lint G15 `permission-signature`.

### C29 — A palette entry is not a `.ntpl`; the `§Templates` link dangles
- **Sources:** pps#Δ4 (primary; provisioning half rides along)
- **Evidence:** `guides-clean/Templates/DeployingTemplates-054F175F.txt:33`, corpus B1325 C1-C2; reinforces afl Δ3.
- **Target:** `types/distribution.md`, `types/logic.md:185`, `types/structure.md:133`.
- **Lint-able:** yes — dangling `§` link check. **Fold type:** doc fix + lint.
- **Fold status:** FOLDED (WU-P0-7, 2026-10-23; pps Δ4): `types/logic.md` UX row 5 link fixed, palette ≠ `.ntpl` note
  in `types/structure.md` § Palette assembly templates, fleet install in `types/distribution.md` § 12. Corrected
  cite: the evidence line `DeployingTemplates-054F175F.txt:33` is past the end of the 21-line 4.15.6.20 file (claim at
  `:7-11`). Pin KP6; owed Go lint G12 `kit-section-links`.

### C30 — Theme module rules: `themeName` value is the module name; picker location wrong; SPA not auto-themed
- **Sources:** wuh#Δ2 (primary) · r3wsh#Δ4 extends (servlet SPA must link `theme.css` explicitly)
- **Evidence:** vineflower `CustomThemeModuleManager.java:24,47`, vineflower `BHxProfile.java:291`, vineflower `WbWebWidgetServlet.java:143-149`, corpus B1328 §1328.2 C1, B1349 §1349.2.
- **Target:** `types/theme.md` §2-§3, §7; `types/frontend-standard.md`.
- **Lint-able:** yes — def value == moduleName, `theme` prefix, `nss/*.nss` present; grep SPA HTML for `theme.css`. **Fold type:** doc fix + lint.
- **Fold status:** FOLDED (WU-P0-8, 2026-10-23; wuh Δ2, r3wsh Δ4): `types/theme.md` § 2 (value = module name with
  the `theme` prefix, `nss/*.nss` required, silent fallback, per-web-profile choice, Tools › Options for Workbench,
  def via `moduleManifest { defines }`), § 3 layout, § 7 SPA theming; `types/frontend-standard.md` § 1 theme link
  rule. Owed Go lints G13 `theme-module`, G14 `spa-theme-link`.

### C81 — `verify-module.sh` false-FAILs live-loaded signed jars (issue #260)
- **Sources:** bsw#Δ1 (primary: `signed` PASSes on any `META-INF/<ALIAS>.SF` + matching `.RSA`/`.DSA`/`.EC`; report the alias) · bsw#Δ2 extends (pipefail/SIGPIPE 141 false negatives in `check_signed`, `check_types_have_classes`, `check_palette`; here-string membership tests) · bsw#Δ3 extends (skip `META-INF/versions/**` in `check_bytecode_major`, as `lint-bundled-jar-class-version.sh` already does)
- **Winning text:** bsw Δ1 — the signature entry is named after the signing alias (`NIAGARA4`, `CODESIGN`, `SERVER1`); the station decides by verification mode + chain (reinforces pps Δ6, C48). Drop "OEM jars are expected to FAIL".
- **Evidence:** `[ev: live 2026-10-16 verify-module.sh on course jars]`, `[ev: live 2026-10-16 verify-module.sh training-rt.jar]`; GitHub issue #260.
- **Target:** `toolbelt/verify-module.sh` (`check_signed` :213-215, :214, :223, :411, `check_bytecode_major` :202-212), `build-verify.md` § Verify (:96, :103) + § Signing per deploy target (:131), `METHODOLOGY.md:67`, `types/third-party-libraries.md` § packaging paths, `tests/` new bats rows (alias-named `.SF`, >64 KiB listing, multi-release jar).
- **Lint-able:** yes — the check itself is the fix; mutation "restore the pipe" must turn its bats red. **Fold type:** kit-check fix + bats + doc fix.

### C82 — Palette entry count misses entries whose `n=` is not the first attribute
- **Sources:** slo#Δ6 (primary). Confirms afl Δ3 (C11) on the root folder type.
- **Winning text:** count every `<p` element carrying `n="…"` in any position with a non-folder `t="x:Type"`; keep the folder exclusion.
- **Evidence:** `[ev: live 2026-10-16 verify-module.sh training-rt.jar]` (`training-rt.jar` → 10 entries, `Cristian_Angeles_Cert-rt.jar` → 2).
- **Target:** `toolbelt/verify-module.sh` `check_palette` (:420) + bats.
- **Lint-able:** yes. **Fold type:** kit-check fix + bats.
- **Fold status:** FOLDED (WU-P0-4, 2026-10-22; slo Δ6): `palette_scan` counts `<p>` with `n=` in any position and
  a non-folder `t=`, and also skips `v=` slot overrides (kitControl `module.palette:131` children carry `n`+`t`+`v`).

### C83 — The ping API is `void doPing()` with void `pingOk()`/`pingFail(String)`
- **Sources:** ndh#Δ1 (primary) · ndh#Δ3 extends (reachability-only ping recipe: `Socket.connect(addr, timeoutMs)` → `pingOk()`, `IOException` → `pingFail(...)`; confirms bdps Δ2's `NETWORK_COMMUNICATION` grant live, `hosts="*"`/`ports="*"` stays WARN)
- **Winning text:** ndh Δ1 — class-ladder row → `doPing()`; § 5 snippet `public void doPing() throws Exception { … if (ok) pingOk(); else pingFail("no response"); }`; keep the DOWN-propagation text.
- **Evidence:** `[ev: code javap driver-rt.jar BDevice 4.15.6.20]`, `[ev: live 2026-10-16 exam P3 ping {ok} www.google.com:443]`.
- **Target:** `types/driver-authoring.md:15` + § 5 Health (:142-153).
- **Lint-able:** yes — kit self-check FAILs on `BObject doPing` or `return pingOk` in `types/*.md`. **Fold type:** doc fix + self-check.
- **Fold status:** FOLDED (WU-P0-5, 2026-10-22; ndh Δ1, Δ3): § 1.1 + § 5; self-check = `tests/kit-driver-pins.bats` KD1.

### C84 — K15 describes the slot-o-matic marker failure wrongly
- **Sources:** slo#Δ1 (primary). Confirms cex Δ4 (C58: region-less sources need slot-o-matic before `jar`).
- **Winning text:** write a new class with its annotations and NO `BEGIN/END BAJA AUTO GENERATED` markers; pre-written empty markers make slot-o-matic log `Could not process <file>: Found multiple metadata blocks` without failing its task, and the build then fails in `compileJava` with `Slot with name <x> not found … have you run slot-o-matic?`.
- **Evidence:** `[ev: transcript 2026-10-16 exam P1 slotomatic --info]`.
- **Target:** `METHODOLOGY.md` K15 (:122), `toolbelt/build.sh` (pre-gradle marker scan + failure hint).
- **Lint-able:** yes — FAIL a `.java` with a `BEGIN BAJA AUTO GENERATED` marker and no `/*@ $` stamp before `END`. **Fold type:** doc fix + build.sh check + bats.
- **Fold status:** FOLDED (`docs/r8-t9i-k15`, 2026-10-23; slo Δ1): `METHODOLOGY.md` K15 rewritten to the observed
  mechanism, verified in code: organized/ devkit-wb 4.14.0.162 vineflower `JavaUnit.java:270-272` throws "Found multiple
  metadata blocks" for any stamp-line count other than one (an empty region has zero), `Compiler.java:241` logs it as a
  WARNING and goes on; javap of `niagara-slotomatic-library-2.0.4.jar` (4.15.6.20) holds both messages. The compileJava
  message is transcript-only (emitting class not located). `toolbelt/build.sh` prints the skipped file + K15 when the
  compile error appears (BS-slotomatic-hint); the pre-gradle marker scan stays owed as Go lint G3 `slotomatic-markers`
  (no new bash check). Earlier "NOT FOLDED (WU-P0-4)" superseded.

---

## P1 — high-value missing rules that affect client modules

### C31 — HMI session lifetime and the kiosk user (DashboardPan wall panel)
- **Sources:** r3wsh#Δ1 (primary, supersedes the static part of has Δ1) · has#Δ1 superseded-in-part · wuh#Δ1 corrects has Δ1/Δ2 · wuh#Δ6 extends (kiosk user slots) · r3wsh#Δ2 extends (4.15 absolute logoff, both flags) · sta#Δ3 duplicate (absolute logoff) · wuh#Δ7 extends (client-cert SSO)
- **Winning text:** "a poll is not activity"; ladder: (1) panel user Auto Logoff off AND Absolute Logoff off AND `allowConcurrentSessions=false`; (2) SPA keep-alive `POST /timeout` (CSRF) or `activityMonitor.keepAlive()`; (3) `UserActivityFilter` only for a war-style module. has Δ1's "map it on the poll servlet" is rejected as a primary route (no shipped precedent: only web-rt's `file` servlet).
- **Evidence:** `NiagaraSuperSession.java:184-198,217-244`, `SessionTimeoutServlet.java:25-38`, web-rt `WEB-INF/web.xml:59-68,84-91`, corpus B1344, B1318 §1318.2, B1334, B1326 row 8.
- **Target:** `types/dashboard.md` (new § Session lifetime / HMI kiosk), `types/security.md` §3, commissioning checklist.
- **Lint-able:** partial — poll loop must reference `/timeout`/`keepAlive` or document route (1); commissioning check of user slots. **Fold type:** new doc section + lint + commissioning item. Live proof B1315-G1 still open.
- **Fold status:** FOLDED (`docs/r8-p1-web`, 2026-10-23; r3wsh Δ1, Δ2, has Δ1, wuh Δ1, Δ6, Δ7, sta Δ3): the
  static recipe was already in `types/dashboard.md` § Session lifetime (t9c-reference-gaps Δ1); this unit adds "a
  poll is not activity", route 3 scoped to war-style modules, the 401/re-login rule, the 4.14 scope of absolute
  logoff, the kiosk-user commissioning checklist (4.15.6.20 Touch guide lines) and `types/security.md` § 3.4
  (client-certificate SSO). **Premise corrected:** r3wsh Δ1 route (2) "keep-alive = raw `POST /timeout`" is not
  adopted: contract question, doc wins — devguide `autoLogoffWeb.txt:35,112-125` publishes only
  `keepAlive()`/`release(token)`; the raw POST (B1344 §1344.4, `[INFER]`) is the monitor's internal and defeats the
  logoff policy. has Δ1 "map the filter on the poll servlet" stays rejected (only shipped mapping: web-rt `file`,
  vineflower 4.14.0.162 `WEB-INF/web.xml:84-91`). Not adopted: wuh Δ7 "assigning the cert restarts Fox/Web" (not
  found in the 4.15.6.20 Touch guide). Live proof B1315-G1 still open. Owed Go lint G18 `session-keepalive`.

### C32 — Alarms for control modules: inhibit during defrost, ext chooser, ext owns FAULT
- **Sources:** cas#Δ2 (primary) · rac#Δ6 extends (alarm ext clears/owns ALARM|FAULT|UNACKED)
- **Evidence:** `BAlarmSourceExt.java:326,1332,1954`, corpus B1324, B1314 §1314.1 row 13.
- **Target:** `types/logic.md` § alarm decisions + `:336` LC5.
- **Lint-able:** yes — WARN `%parent.` in alarm text defaults. **Fold type:** new doc section + lint.
- **Fold status:** FOLDED (`docs/r8-p1-control`, 2026-10-23; cas Δ2, rac Δ6): `types/logic.md` LC8 (ext chooser from the
  4.15.6.20 guide, `alarmInhibit` + `inhibitTime`, texts against the record) and LC5 corrected. **Premise corrected (kit
  text):** LC5 "alarm-ext … is notification-only" — code wins: the ext owns ALARM|FAULT|UNACKED_ALARM of `out` (devguide
  source 4.15.6.20 `BAlarmSourceExt.java:1330-1331,1954`; vineflower alarm-rt 4.14.0.162 `NON_ALARM_BITS = -139`, `:222`).
  LC4 reworded to "no interlock". Pin KPC1; owed Go lint G25 `alarm-text-parent`.

### C33 — Reuse kitControl before hand-rolling: LeadLag, Sequence, Interstart, timers, LoopPoint, writables
- **Sources:** cas#Δ5 (primary) · r3pec#Δ3 extends (LeadLag feedback inputs must be linked) · r3pec#Δ4 corrects cas Δ5 (Interstart min on/off = inherited `minActiveTime`/`minInactiveTime`, default 0) · rac#Δ7 extends (`activeLevel`, overrides, min on/off on Boolean writables) · cas#Δ6 (timers TRANSIENT) · cas#Δ7 (LoopPoint zero gains, 0.5 s ticket) · cas#Δ8 (Tstat no lockout; ext ordering)
- **Evidence:** `BLeadLagRuntime.java:307,385,411`, `BLeadLagCycles.java:1116-1124,1187-1200`, `BInterstartDelayControl.java:63-160`, `BSequence.java:575`, `BLoopPoint.java:352,499`, `BBooleanDelay.java:535`, `BStatus.java:824`, corpus B1331, B1351, B1314 §1314.1.
- **Target:** `types/logic.md` after `:333`, LC1, `:332-333`.
- **Lint-able:** partial — bog checks (feedback links, non-zero min on/off, kP=0). **Fold type:** new doc section + bog lint.
- **Fold status:** FOLDED (`docs/r8-p1-control`, 2026-10-23; cas Δ5, Δ6, Δ7, Δ8, r3pec Δ3, Δ4, rac Δ7): `types/logic.md`
  LC7 (LeadLag feedback links, Sequence, Interstart = inherited `minActiveTime`/`minInactiveTime`, TRANSIENT timers,
  inert LoopPoint, Tstat + ext ordering) and LC1 (`activeLevel`, override actions/`overrideExpiration`/
  `maxOverrideDuration` facet, Boolean writable min on/off). Verified in kitControl-rt and control-rt 4.14.0.162
  vineflower; the 4.15.6.20 devguide source agrees where the corpus cites it; 4.13 not verified. r3pec Δ4's correction
  of cas Δ5 adopted (min on/off is inherited, default 0). Owed Go lint G26 `kitcontrol-bog`.

### C34 — Timers and schedules: TriggerSchedule for defrost, `clockChanged` for absolute tickets
- **Sources:** cas#Δ9 (primary) · rac#Δ3 extends (absolute-start `schedulePeriodically` needs `clockChanged`) · cas#Δ10 extends (weekly schedule `defaultOutput`, share via ScheduleSelector)
- **Evidence:** `BTriggerSchedule.java:274,524,527,540`, devguide `execution.txt:131-137`, ColdRoomPan `BDefrostMode.java:16`, corpus B1335, B1304 §1304.2.
- **Target:** `types/logic.md` § Watchdogs and timers, `types/logic-authoring.md:298`, `toolbelt/lint-timers.sh`.
- **Lint-able:** yes — WARN absolute `Clock.schedule`/`schedulePeriodically` without `clockChanged`. **Fold type:** new doc section + lint.
- **Fold status:** FOLDED (`docs/r8-p1-control`, 2026-10-23; cas Δ9, rac Δ3, cas Δ10): `types/logic.md` § Safety
  fail-modes & timers (relative vs absolute tickets from the 4.15.6.20 `Clock.java` javadoc + `execution.txt:64-66`;
  TriggerSchedule with the persisted `nextTrigger`, flags 265 in vineflower schedule-rt 4.14.0.162 `:102`, `:274` in
  the 4.15.6.20 source), `types/logic-authoring.md` § Schedule integration (`defaultOutput`, specials, selectors). The
  `lint-timers.sh` WARN is owed as Go lint G24 `absolute-ticket-clock` (a new check, not a small edit).

### C35 — History extensions and commissioning: `_cfg0` split, ship disabled, capacity, alarm DB size
- **Sources:** has#Δ3 (primary: split rule + exporter URL) · cas#Δ3 reinforces (disabled on palette, literal `historyName`, rollover) · poh#Δ5 extends (capacity 500, pasted ext disabled + empty name, implied tags from facets, `out.value`) · cas#Δ4 extends (file alarm DB 500)
- **Evidence:** `LocalDbConnection.java:226-229`, `BHistoryExt.java:281`, devguide `history.txt:75`, `servlets/exporters.txt:16-20`, corpus B1315, B1324, B1256 §1256.6, B1253 §1253.3 K4.
- **Target:** `types/logic-authoring.md:536-545`, `types/dashboard.md`, `BUILD-LOOP.md` commissioning.
- **Lint-able:** partial — flag literal `historyName`; diff interval defaults across versions. **Fold type:** doc section + commissioning items.
- **Fold status:** FOLDED (`docs/r8-p1-control`, 2026-10-23; has Δ3, cas Δ3, poh Δ5, cas Δ4): `types/logic-authoring.md`
  § Logging a point to history (split, disabled by default, script name, capacity 500, rollover, `out.value`, implied
  tags, `ITableToCsv`), `BUILD-LOOP.md` § 6.b item 7 (alarm DB 500, history exts). **Premise corrected:** has Δ3 "a
  config change starts `<name>_cfg0`" — code wins: on a `recordType`/`interval` change of a non-empty history the OLD
  records are renamed to `<name>_cfg0` and a new empty history takes the original name (vineflower history-rt
  4.14.0.162 `HistoryDatabaseConnection.java:194-221`, `LocalDbConnection.java:213-238`; 4.15 not verified). Same
  operational effect (readers see an empty history), different place for the old data. Owed Go lint G27 `history-name-literal`.

### C36 — Slot and callback contracts
- **Sources:** scc#Δ3 (primary: `isRunning()` first line) · scc#Δ4 (validators skip link writes) · scc#Δ5 (slot-API traps) · scc#Δ6 (one-shot status write, TRANSIENT run-state, single-flight, ids in `started()`) · scc#Δ2 (flags are UI, not security; `checkSetFlags`)
- **Evidence:** corpus B1231 §1231.6/§1231.9/§1231.11, B1232 §1232.2/§1232.4, B1253 §1253.3 K1/K6, B1254 §1254.5.
- **Target:** `types/logic.md` § Tridium rt idioms / Slots / Restart & persistence; `types/logic-authoring.md`; `types/security.md` §2.
- **Lint-able:** yes — WARN callback without `isRunning()` guard (B1242 K4); `get("…") != null`; `removeAll()`; `setValue` then `setStatus`. **Fold type:** doc section + lint.
- **Fold status:** FOLDED (`docs/r8-p1-slots`, 2026-10-23; scc Δ2, Δ3, Δ4, Δ5, Δ6): `types/logic.md` § Tridium rt idioms
  (callback contract, validators skip link-fed writes, one-shot status write, TRANSIENT run-state, single-flight,
  persisted ids, slot-API traps), `types/security.md` § 2.2a. **Premise refined:** scc Δ2 "READONLY is honoured only by
  the Workbench property sheet" — code: `set(prop, v, cx)` checks only operator/admin write (vineflower baja 4.14.0.162
  `BUser.java:902-912`), but `BComponent.canWrite(OrdTarget)` does refuse READONLY (`BComponent.java:925-939`); the kit
  states both and leaves "which remote surfaces ask canWrite" not verified. scc Δ5's `NoSuchSlotException`: javadoc
  (docSource 4.14.0.162 `BComplex.java:686-697`) vs a `Property` cast in the 4.14.0.162 body (`ComplexSlotMap.java:371-373`);
  the rule avoids the call (K30). Owed Go lints G29 `callback-running-guard`, G32 `slot-api-traps`.
  **Extensions FOLDED** (U11, 2026-10-23; slo Δ2, Δ3): `types/logic-authoring.md` § Generated accessors and slot declaration
  order (primitive accessors from devkit-wb 4.14.0.162 `PropertyProcessor.java:40-91`; low-before-high order). Premise
  corrected: slo Δ3's "a later reorder is a schema change" — a `.bog` binds by name (`schema-risk.sh` `reorder_slot,SAFE`);
  the reorder changes the operator's view only. Owed Go lint G96 `slot-pair-order`.

### C37 — Concurrency doctrine: no intrinsic locks, named threads, callback thread map
- **Sources:** tjd#Δ4 (primary: `lint-intrinsic-lock.sh`) · tjd#Δ5 extends (naming/priority; reinforces rac Δ1's `defaultThreadFactory`) · poh#Δ1 extends (callback → thread map)
- **Evidence:** course `Slides/9 Multithreaded Development.pdf` p4-11, p21, p26-28; corpus B1238 §1238.2, §1238.10; DashboardPan `BDashboardService` has 5 `synchronized` methods.
- **Target:** `types/logic.md` § Tridium rt idioms; new `toolbelt/lint-intrinsic-lock.sh`.
- **Lint-able:** yes — FAIL `synchronized` method / `synchronized(this)` in a `B*` class; WARN unnamed `new Thread(`/`Executors.new*()`. **Fold type:** doc section + new lint.
- **Fold status:** FOLDED (`docs/r8-p1-slots`, 2026-10-23; tjd Δ4, Δ5, poh Δ1): `types/logic.md` § Tridium rt idioms (no
  intrinsic locks with the framework's own `synchronized (this.instance)` at vineflower baja 4.14.0.162
  `ComplexSlotMap.java:740`; named threads; the callback thread map with `Console.java:41`, `EngineManager.java:560`,
  fox-rt `SessionDispatcher.java:16`). The new `lint-intrinsic-lock.sh` is NOT written (round-8 rule: no new bash
  lints): owed Go lints G30 `intrinsic-lock`, G31 `thread-naming`. The live thread-map run stays open (B1238-G2).

### C38 — Action threading and the two coalescing winners
- **Sources:** tjd#Δ3 (primary: Fox dispatcher, ASYNC → engine, `post()` override with `Invocation`) · rcf#Δ1 reinforces (confirms tjd Δ3; `post()` FIRST-wins vs `CoalesceQueue` LAST-wins; queued ASYNC runs with null Context)
- **Evidence:** `ActionQueue.java:28,85`, `EngineManager.java:310`, `Invocation.java:85-91`, course `Slides/9` p13-17, corpus B1238 §1238.3-5, B1316 §1316.1.
- **Target:** `types/actions.md` § 2 Threading model.
- **Lint-able:** yes — WARN blocking ASYNC action without `post(` override; bare `this::do` in `post(`. **Fold type:** doc section + lint.
- **Fold status:** FOLDED (`docs/r8-p1-slots`, 2026-10-23; tjd Δ3, rcf Δ1): `types/actions.md` § 2 (Fox:Dispatcher, 60 s
  request timeout, `ActionQueue` FIRST-wins vs `CoalesceQueue` LAST-wins, null Context in queued ASYNC, the `Invocation`
  offload), all read in baja/fox-rt 4.14.0.162 vineflower; 4.15 not verified. Owed Go lint G28 `async-blocking-action`.

### C39 — A null Context means `BPermissions.all`; `OrdTarget.can*` pass without a security target
- **Sources:** rcf#Δ2 (primary) · nsto#Δ8 extends (custom `BOrdScheme` contract; gate every hop with `canRead()`) · rsb#Δ5 extends (worker threads must carry the action's Context: null Context = all permissions + default locale)
- **Evidence:** `BComponent.java:1979-1982`, `OrdTarget.java:465-480`, corpus B1316 §1316.2, B1308 §1308.1.
- **Target:** `types/security.md` §4, `types/logic-authoring.md:73`, `types/module-wiring.md` § ordScheme.
- **Lint-able:** partial — WARN `.resolve(x, null)` in a file that calls `.canRead(`. **Fold type:** doc section + lint.
- **Fold status:** FOLDED (`docs/r8-p1-web`, 2026-10-23; rcf Δ2, nsto Δ8, rsb Δ5): `types/security.md` § 4.4
  (`getPermissions` → `BPermissions.all` with no user; `OrdTarget.can*` true with no security target; custom
  `BOrdScheme` contract) and § 2.4 worker-thread Context, `types/logic-authoring.md` § ORD resolution pointer,
  `types/module-wiring.md` ordScheme pointer. Verified in vineflower baja 4.14.0.162 (`OrdTarget.java:281-294`,
  `BComponent.java:896-903`) and devguide source 4.15.6.20 (`OrdTarget.java:476-503`, `BComponent.java:1971-1982`);
  4.13 not verified. The lint half rides with #293 as owed Go lint G19 `null-context-permission-check`.

### C40 — Test runner that refuses silent passes
- **Sources:** tpb#Δ2 (primary)
- **Winning text:** `toolbelt/run-niagara-test.sh` — scratch copy, always `:clean`, PASS only when every `Total tests run: N` has N>0 and 0 failures; FAIL on `invalid test target(s)` / `failed signature validation`; back up and restore the reference jar.
- **Evidence:** commit 337ee6c (two BUILD SUCCESSFUL runs with 0 tests).
- **Target:** new `toolbelt/run-niagara-test.sh`, `toolbelt/INDEX.md`, bats with three log fixtures.
- **Lint-able:** yes. **Fold type:** script.
- **Fold status:** PARTIALLY FOLDED (`docs/r8-p1-tests`, 2026-10-23; tpb Δ2): the runner is a new script, which round 8
  does not write in bash; its contract is in `types/moduleTest.md` § Known gotchas (read the `Total tests run` and
  rejection lines by hand meanwhile) and its full spec is owed as Go tool G33 `run-niagara-test` (ledger fragment
  r8-p1-tests). LEFT: the runner, its `toolbelt/INDEX.md` row and the three log fixtures.

### C41 — moduleTest doctrine, layout and template
- **Sources:** tpb#Δ4 (primary: station tests for component behaviour, structural tests last resort) · tpb#Δ5 extends (structural-test advisory lint) · tpb#Δ6 extends (`moduleTest-include.xml` template + scaffold) · lex#Δ3 duplicate on the 4.14 layout + extends (BTestNg vs BTestNgStation, mockStatic, FINE assertions, no fixed sleeps)
- **Evidence:** commit 337ee6c; corpus B1252 §1252.3, B1236 §1236.5-6; `toolbelt/lint-structure.sh` L11 mixed-srcTest check (Go port: `go/internal/lint/rules/structure/profile.go:144-166`).
- **Target:** `types/moduleTest.md`, `METHODOLOGY.md` testing tiers, `toolbelt/lint-structure.sh` (L12), `toolbelt/scaffold-module.sh`.
- **Lint-able:** yes — WARN `srcTest` reading `.java` as text with no BTestNg class. **Fold type:** doc section + template + lint.
- **Fold status:** PARTIALLY FOLDED (`docs/r8-p1-tests`, 2026-10-23; tpb Δ4, Δ5, Δ6, lex Δ3): `types/moduleTest.md`
  (client parity doctrine, base-class choice, idempotent station fixtures, DB seeding, mockStatic, no fixed sleeps,
  `setup()` trap from docSource 4.14.0.162 `BTest.java:72`, `@BeforeTest` at `BTestNgStation.java:93-94`; the 4.14+
  `test.` package layout from devkit-wb 4.14.0.162 `BNfooVideoDriverTest.java.vm:4` and the course suite), new
  `templates/moduleTest-include.xml`, `METHODOLOGY.md` tier note. LEFT: `scaffold-module.sh` emission (outside the
  surface); the structural-test advisory is owed as Go lint G34 `structural-tests` (a new check, not a small edit of
  L12). Whether 4.14+ tooling still discovers the older layout is `[INFER]`. Pin KPT1.
  **Doc half of the emission FOLDED** (U11, 2026-10-23; tpb Δ6): `types/structure.md` § What the scaffold emits today (copy the
  template by hand; the § PASS state stub is not well-formed XML). The fixture + scaffold emission is owed as G97
  `fixture-moduletest-include` (`ledger.d/2026-10-23-r8-u11-leftovers.md`); the stub-block fix is an owed pointer (P4).

### C42 — Shipped libraries: `bin/ext` first, wrapper modules, version drift
- **Sources:** shl#Δ1 (primary: `nre("g:a")` no version, catalog table) · shl#Δ3 extends (drift: httpclient 4.5 → 5) · shl#Δ4 extends (wrapper table; commonsLang-rt empty in 4.15) · shl#Δ6 extends (wrapper does not grant its permissions)
- **Evidence:** devguide `build.txt:238,250`, `upgradingBuild.txt:289`, corpus B1320, B1321.
- **Target:** `types/third-party-libraries.md` after `:15` and near `:160`, `types/module-wiring.md` §1.
- **Lint-able:** yes — WARN bundling okhttp/okio/kotlin-stdlib/commons-lang3/codec/slf4j/jose4j/httpclient5/bouncycastle; `import org.apache.http.`. **Fold type:** doc section + lint.
- **Fold status:** FOLDED (`docs/r8-p1-tests`, 2026-10-23; shl Δ1, Δ3, Δ4, Δ6): `types/third-party-libraries.md` § Before
  bundling (nre without version, devguide 4.15.6.20 `build.txt:186`; a three-install drift table from `ls bin/ext` of
  iC-Niagara 4.13.2.18, Honeywell OptimizerSupervisor 4.14.0.162 and Niagara 4.15.6.20; the wrapper table from corpus
  B1321; `commonsLang-rt` empty in 4.15.6.20 by `unzip -l`, 424 `lang3` entries in the 4.14.0.162 copy; wrappers lend
  no permissions), `types/module-wiring.md` § 1 `nre("g:a")` row. The okhttp half was WU-P0-9. Pin KPT2; owed Go lint
  G35 `shipped-library-bundle`.

### C43 — Outbound HTTP ladder; the non-driver-client gate is closed by default; OAuth2
- **Sources:** shl#Δ7 (primary ladder) · r3di#Δ4 corrects shl Δ7(b) (`enableNonDriverClients=false` → every send skipped with only a health fault; mount with `mountAtOrd`) · r3-report#R3 · r3di#Δ5 extends (OAuth2 via `oauth2-rt` + `BAuthorizationHeaderScheme`, the `email-rt` pattern)
- **Evidence:** `BHttpClientService.java:88-89,231-252,270`, `BHttpClient.java:277,304-306`, `IHttpClient.java:23`, corpus B1333, B1346.
- **Target:** `types/cloud-connector.md` §7, `types/issues-and-gotchas.md` H2, `types/security.md`.
- **Lint-able:** partial — grep `HttpClientBuilder` without `mountAtOrd`. **Fold type:** doc section + commissioning item.
- **Fold status:** FOLDED (`docs/r8-p1-web`, 2026-10-23; shl Δ7, r3di Δ4, Δ5, r3-report#R3): `types/cloud-connector.md`
  § 7 rewritten (four-step ladder, gate closed by default, health-fault refusal, commissioning item, OAuth2 route),
  `types/issues-and-gotchas.md` H2 corrected. **Premise corrected (kit text):** "a non-driver client receives a
  `ServiceException`" — code wins: httpClient-rt has no such exception; `permittedByService` catches the refusal,
  writes a health fault and returns false, `sendSync()` returns `""` (vineflower httpClient-rt 4.14.0.162
  `BHttpClientService.java:231-252`, `BHttpClient.java:277,302-306`; javap `httpClient-rt.jar` 4.15.6.20 same message,
  default false). The leftover "okhttp from `net-rt`" in both passages removed (C27). Unmounted-client NPE stays
  `[INFER]` (B1346-G1). Pins KPW1, KPW2; owed Go lint G20 `http-client-mount`.

### C44 — Driver health: failed polls do not ping-fail a device; ping rules
- **Sources:** r3di#Δ1 (primary) · drp#Δ7 extends (faulted device never pinged; alarm after retries + 5-min startup delay)
- **Evidence:** `BProxyExt.java:923-930`, `BBacnetDevice.java:1096-1102`, `BPingMonitor.java:484-488,526,532`, corpus B1348 §1348.1, B1337.
- **Target:** `types/driver-authoring.md` §5 after `:182`.
- **Lint-able:** partial. **Fold type:** doc section.
- **Fold status:** FOLDED (`docs/r8-p1-drivers`, 2026-10-23; r3di Δ1, drp Δ7): `types/driver-authoring.md` § 5 (a failed
  poll is not a failed ping; the BACnet `ping()` + `maxPollTimeouts` pattern; ping-monitor rules). **Premise checked:**
  the 4.15.6.20 guide's "in most drivers a poll with no response notes a ping fail" (`Drivers/driver-PingMonitor.txt:37-40`)
  vs code — code wins for behaviour: `BProxyExt.readFail` makes no device call (vineflower driver-rt 4.14.0.162
  `BProxyExt.java:463-469`); BACnet does it in its own poll path (vineflower bacnet-rt 4.14.0.162
  `BBacnetDevice.java:1093-1101`). 4.13/4.15 code not verified. No lint (policy in the module's own comm code).

### C45 — BACnet `statusFlags` facet or field alarm/fault never reaches Niagara status
- **Sources:** drp#Δ4 (primary) · r3di#Δ3 corrects scope (COV and polled alike; cost note)
- **Evidence:** `BBacnetProxyExt.java:1404-1409`, `CovHandler.java:266-270`, `Bacnet/TroubleshootingBACnet-4FF09FBA.txt:86`, corpus B1327 row 3, B1348 §1348.2.
- **Target:** `types/driver-authoring.md:129`, `types/dashboard.md` health read.
- **Lint-able:** yes — bog check. **Fold type:** doc section + bog lint.
- **Fold status:** FOLDED (`docs/r8-p1-drivers`, 2026-10-23; drp Δ4, r3di Δ3): `types/driver-authoring.md` § 2 BACnet
  `statusFlags` paragraph (vineflower bacnet-rt 4.14.0.162 `BBacnetProxyExt.java:808-822`; COV and polled alike per
  r3di Δ3's scope correction of drp Δ4, corpus B1348 §1348.2). The dashboard side is a pointer. Owed Go lint G36
  `bacnet-status-flags`.

### C46 — oBIX exposure: licence, priority writes, watches, hidden ≠ protected
- **Sources:** drp#Δ9 (primary) · r3wsh#Δ6 extends (HIDDEN removes from listings only; served by path)
- **Evidence:** `Obix/LicenseRequirement-OBIXDriver-9F25675C.txt:19`, `obixDriver-ObixExport.txt:61`, `ObixEncoder.java:234-236,588-589`, corpus B1343, B1347. Applies to the PANCCADIA oBIX read/write users.
- **Target:** `types/dashboard.md:23`, `types/logic-authoring.md:554`, `types/security.md` (oBIX exposure).
- **Lint-able:** partial. **Fold type:** doc section.
- **Fold status:** FOLDED (`docs/r8-p1-web`, 2026-10-23; drp Δ9, r3wsh Δ6): `types/security.md` § 2.5 (licence and
  410/500 from the 4.15.6.20 Obix guide, `ObixExport` priority writes, `set` vs `override`, watch leases, HIDDEN ≠
  protected from obixDriver-rt 4.14.0.162 vineflower `ObixEncoder.java:234-236,588-589`). The direct-path read of a
  HIDDEN slot stays `[INFER]` (not executed). No lint (no deterministic signal in module source).

### C47 — Canonical `module-permissions.xml` template and Security Manager card
- **Sources:** afl#Δ2 (primary: template, fixture, scaffold) · has#Δ5 extends (debug switches, permission-free homes, `<opt-permission>`, Policy Spy)
- **Evidence:** course `Slide Code/…/module-permissions.xml`, `Labs/Lab9 Multithreaded/module-permissions.xml`; devguide `security/securityManager.txt:44-124`, `requestingPermissions.txt:677-730`; corpus B1312 §1312.1.
- **Target:** new `templates/module-permissions.xml`, `types/structure.md` § module-permissions.xml, fixture, `toolbelt/scaffold-module.sh`.
- **Lint-able:** yes — WARN non-`<permissions>` root, `<req-permission>` without `<purposeKey>`; WARN `new File("/` outside homes. **Fold type:** template + doc section + lint.
- **Fold status:** PARTIALLY FOLDED (`docs/r8-p1-sign`, 2026-10-23; afl Δ2, has Δ5): new `templates/module-permissions.xml`
  (devkit-wb 4.14.0.162 wizard shape `gradle/includes/niagara/module-permissions.xml.vm:1-19` + course Lab9 requests),
  `types/structure.md` § module-permissions.xml (template, `AccessControlException` path, Security Manager card from
  devguide 4.15.6.20 `security/securityManager.txt:44-124`, `requestingPermissions.txt:40-80,142,596,653`). LEFT: the
  fixture copy and `scaffold-module.sh` emission (outside this unit's edit surface); the lint half is owed Go lint G21
  `permissions-xml`. Pin KPS2.
  **Doc half of the fixture FOLDED** (U11, 2026-10-23; afl Δ2): `types/structure.md` § What the scaffold emits today — the fixtures'
  `<modulePermissions/>` root is copied into `META-INF/module.xml` as is and never read (baja 4.14.0.162 `NModule.java:442-452`
  reads only `<permissions>`). The fixture replacement is owed as G98 `fixture-permissions`.

### C48 — Signing hardening and verification modes
- **Sources:** bdps#Δ4 (primary: `allowDefaultProfile.set(false)`, `RestrictedSigningProfile`, TSA, Module Info) · pps#Δ6 extends (Signing Service ≠ module signing; mode × status table; read `niagara.moduleVerificationMode`) · pps#Δ7 extends (program objects, RSA 3072, "Unable to load" causes) · rsb#Δ1 extends (verify-module `--expect-signer <pem>` fingerprint check; WARN on the default `Niagara4Modules` signer) · rsb#Δ2 extends (signing-profile XML is a secret store: never print it, keep it out of repos) · rsb#Δ3 extends (the two root blocks are the portable signing unit; `--info` proof lines)
- **Evidence:** corpus B1234 §1234.3/§1234.6, B1332; `ModuleSign/VerificationModesModuleSigning-182D0E86.txt:25-36`.
- **Target:** `build-verify.md` § signing, `types/distribution.md:100`, `types/logic-authoring.md:917`, `toolbelt/lint-structure.sh`.
- **Lint-able:** yes — WARN signing plugin without `allowDefaultProfile.set(false)`. **Fold type:** doc section + lint.
- **Fold status:** FOLDED (`docs/r8-p1-sign`, 2026-10-23; bdps Δ4, pps Δ6, Δ7, rsb Δ1, Δ2, Δ3): `build-verify.md`
  § Signing per deploy target (read the mode; mode × status table from the 4.15.6.20 ModuleSign guide; Signing
  Service ≠ module signing; `RestrictedSigningProfile` + RSA 3072 + `allowDefaultProfile`; clean + Module Info check;
  profile-XML secrecy; "Unable to load" causes; Jar Signer Tool), `types/distribution.md` pointer,
  `types/logic-authoring.md` § BProgram. **Premise checked:** pps Δ6 "the guide's default Low is stale" — code wins:
  `ModuleVerificationMode.make` returns `medium` for ≥ 4.9 (vineflower baja 4.14.0.162 `Nre.java:752-763`, decompiled
  nre-ext 4.14 `ModuleVerificationMode.java:28-33`) against the guide's `VerificationModesModuleSigning-182D0E86.txt:17`;
  the kit's existing JACE/supervisor line was already right. **Premise corrected (kit text):** "an unsigned
  `BRobotCode` blob is rejected at instantiation" — code wins: `BCode` verifies only when a signature is present or
  `program.requireSigning` is true, else logs `program.notSigned` (vineflower program-rt 4.14.0.162 `BCode.java:202-215`);
  the guide agrees enforcement is off (`User/ProgramObjectSigning-10F25D1D.txt:16`). The allowDefaultProfile WARN is
  L15 (C85). Owed Go lints G22 `signer-identity` (rsb Δ1, replaces the verify-module `--expect-signer` flag) and G23
  `signing-secrets` (rsb Δ2). rsb Δ3's `--info` proof lines not folded (no source line located). Pin KPS1.

### C49 — `BSimple` encoding must be locale-safe (Spanish stations)
- **Sources:** poh#Δ2 (primary)
- **Evidence:** corpus B1252 §1252.4 (`BPair.encodeToString()` via `toString()` → `(1,00,2,00)` on `es`).
- **Target:** `types/value-types.md:123-144`.
- **Lint-able:** yes — WARN `String.format(`/`.toString()` on a `B*` field in `encodeToString`. **Fold type:** doc section + lint.
- **Fold status:** FOLDED (`docs/r8-p1-drivers`, 2026-10-23; poh Δ2): `types/value-types.md` § 3 Custom BSimple (encode
  with encoders: vineflower baja 4.14.0.162 `BObject.java:111-113`, `BDouble.java:131-133,261-262,354-381`; the course
  `BPair.java:78-81` as the failing example). The `es` corruption is `[INFER]` from the code (not executed). Owed Go
  lint G37 `bsimple-encode-locale`.

### C50 — Staging comparisons need an explicit NaN/null fallback
- **Sources:** rcf#Δ9 (primary)
- **Evidence:** corpus B1336 §1336.1 row 5, **B1353 §1353.2** (`EngNotes/aHandlingNull_ExprComponent.txt:23-27`, `[CERT-doc]`).
- **Target:** `types/logic-authoring.md` (staging).
- **Lint-able:** partial. **Fold type:** doc section.
- **Fold status:** FOLDED (`docs/r8-p1-control`, 2026-10-23; rcf Δ9): `types/logic.md` § Zero-demand (every staging
  comparison names its fallback; Expr null handling per the 4.15.6.20 guide `EngNotes/aHandlingNull_ExprComponent.txt:13-17`,
  `[CERT-doc]`, not verified in code). No lint: a NaN guard near a comparison has no deterministic signal.

### C51 — Kit process: register the course, reference-first, sweep absolute claims
- **Sources:** cex#Δ1 (primary: SOURCES.md + corpus-index row) · cex#Δ2 extends (reference-first Orient rule) · cex#Δ3 extends (K-rule + one-time "never"/"forbidden" sweep; C01, C03, C09, C11 are its worked examples) · rsb#Δ6 extends (delta totals come from a counting script, never a hand sum)
- **Evidence:** corpus B1242 §1242.3/§1242.7 K13, B1238 §1238.8; user session 2026-10-15.
- **Target:** `SOURCES.md`, `corpus-index.md`, `BUILD-LOOP.md` §0, `skill/SKILL.md` step 2, `METHODOLOGY.md` K-rules.
- **Lint-able:** partial — report of absolute claims citing only inference. **Fold type:** doc + report script.
- **Fold status:** PARTIALLY FOLDED (`docs/r8-p1-process`, 2026-10-23; cex Δ2, Δ3, rsb Δ6): `BUILD-LOOP.md` § 0 reference-first
  rule (+ the course as a primary exemplar, B1230-B1258), `METHODOLOGY.md` K33 (absolute claims checked against Tridium
  training; C01/C03/C09/C11 as worked examples) and K34 (delta totals counted by script; this register's 178-vs-168 hand
  sum as the example). LEFT: cex Δ1's `SOURCES.md` § Exemplar source and `corpus-index.md` rows, and the `skill/SKILL.md`
  step-2 line — all three files are outside this unit's edit surface; the one-time "never"/"forbidden" sweep. Owed Go
  lint G38 `absolute-claims` and Go tool G39 `count-retro-deltas`.
  U11 (2026-10-23): cex Δ1 still outside the surface; its exact `SOURCES.md` and `corpus-index.md` rows are recorded as owed
  pointer P1 in `ledger.d/2026-10-23-r8-u11-leftovers.md` (`[deferred: …Δ1]`); hwf Δ1 (copy after a diff) FOLDED in
  `METHODOLOGY.md` § Hands-on work, helper owed G93 `diff-vs-reference`.

### C52 — Client parity ticket (course convention wins on five points)
- **Sources:** poh#Δ3 (primary)
- **Winning text:** client follow-ups, not kit rules: `<Type>.displayName` keys, palette roots, logger name = moduleName (this one also becomes a kit rule in `types/observability.md` §1), private lock in `BDashboardService`, operator text via `LexiconModule`.
- **Evidence:** read-only parity of `Cliente/panccadia-leon` 7ca1b57; corpus B1236 §1236.3, B1255 §1255.7.
- **Target:** `toolbelt/kit-ticket.sh` + `types/observability.md` §1.
- **Lint-able:** yes via C10/C11 checks + new logger-name WARN. **Fold type:** kit-ticket + doc line.
- **Fold status:** PARTIALLY FOLDED (`docs/r8-p1-process`, 2026-10-23; poh Δ3): the kit half, logger name = exact
  `moduleName`, in `types/observability.md` § 1 (course `BLoggingExample.java:89`, Lab 7 `BWorkOrderService.java:271`).
  LEFT: the client parity ticket, (a) `<Type>.displayName` keys, (b) `b:UnrestrictedFolder` palette roots, (d)
  `BDashboardService` private lock (C37 names it), (e) operator text via `LexiconModule.getText` — client follow-ups for
  `Cliente/panccadia-leon`; `toolbelt/kit-ticket.sh` is outside this unit's surface. Owed Go lint G40 `logger-name`.

### C85 — Root `build.gradle.kts` signing block: template, scaffold and root-aware lint
- **Sources:** bsw#Δ4 (primary). Extends bdps Δ4 (C48): the lint reads the ROOT `build.gradle.kts` (the wizard folder is the Gradle root project).
- **Winning text:** WARN when `com.tridium.niagara-signing` is applied without both `signingServices { signingProfileFactory { allowDefaultProfile.set(false) } }` and `niagaraSigning { aliases.set(…); signingProfileFile.set(…) }`; ship the block as `templates/signing-root.gradle.kts` and emit it from `scaffold-module.sh`.
- **Evidence:** `[ev: code devkit-wb.jar!gradle/build.gradle.kts.vm:18]`.
- **Target:** `toolbelt/lint-structure.sh`, `build-verify.md` § Signing per deploy target, `templates/`, `toolbelt/scaffold-module.sh` (+ scaffold-diff fixtures).
- **Lint-able:** yes. **Fold type:** template + lint + doc section.
- **Fold status:** FOLDED (WU-P0-9, 2026-10-23; bsw Δ4, issue #266): `lint-structure.sh` L15 (WARN, uncommented-code
  rule shared by table with `preflight.sh` sample-sign, LS15-sync), `templates/signing-root.gradle.kts`, both fixture
  roots (so `scaffold-module.sh` emits it; the script prints the two gradle properties), `build-verify.md`. The
  property-driven `aliases`/`signingProfileFile` form (the course form uses literals) was built offline (U3b; N4
  4.14.0.162 mirror, plugin 7.6.17, Gradle 7.6, via `build.sh`): it compiles, configures, and fails closed (no
  properties → `:jar` "signingAwareArchive.aliases doesn't have a configured value", exit 30; missing profile file →
  `:jar` "doesn't exist", exit 30; `build.sh` needs `ORG_GRADLE_PROJECT_*`, kit issue #308) `[CERT-live local build]`.
  A throwaway self-signed key + profile signs the jar (jarsigner verified), and `allowDefaultProfile.set(false)` alone
  refuses a profile-less build ("The default profile is not available"); without it the plugin silently signs with an
  auto-created Niagara4Modules dev key `[CERT-live local build]`. A production CA-issued profile was not exercised.

### C86 — Jar-lock diagnosis must be route-independent; always run qualified tasks
- **Sources:** bsw#Δ5 (primary)
- **Winning text:** `build.sh` pattern `FileAlreadyExistsException.*modules[/\\].*\.jar` (Windows `gradlew.bat` prints backslash paths); run qualified tasks (`:<MOD>-<part>:jar`), never a bare `gradlew jar`; leftover `java.exe` processes are Gradle daemons (`gradlew --stop`), find the real holder with `tasklist` for `station.exe`/`wb.exe`.
- **Evidence:** `[ev: transcript 2026-10-16 exam P1 FileAlreadyExistsException]`.
- **Target:** `toolbelt/build.sh` (:255), `BUILD-LOOP.md` § 4 Post-jar copy lock (:131), `build-verify.md` § Raw gradle (:30).
- **Lint-able:** yes (pattern, bats row). **Fold type:** build.sh fix + bats + doc fix.

### C87 — Sample / course project retarget preflight
- **Sources:** bsw#Δ6 (primary). Extends cex Δ4 (C58), which covers only the missing slot-o-matic region.
- **Winning text:** before building a project the user did not generate, check `gradle.properties` `niagara_home` exists and matches the installed version, `niagara_user_home` holds no `<…>` placeholder, the ROOT has the signing block (C85), and no shipped `build/` directory is present.
- **Evidence:** `[ev: code Labs/Lab11 NDriver/trafficLightServer.zip!gradle.properties:5,8]`.
- **Target:** `toolbelt/preflight.sh` (new check), `build-verify.md` § Build target & plugin version.
- **Lint-able:** yes. **Fold type:** preflight check + bats + doc line.

### C88 — Wizard-stub lint for NDriver modules (silent-success `doPing`)
- **Sources:** ndh#Δ2 (primary)
- **Winning text:** FAIL when a `doPing()` body's only executable statement is `pingOk();` or still holds `TODO - add ping implementation`; WARN on the other wizard markers in shipped `src/` (`TODO add poll support`, `TODO - convert linkMessage`, a message factory that only `return null;`, `TODO add code to detect beginning and end of message`).
- **Evidence:** `[ev: code devkit-wb.jar!gradle/ndriver/BNfooDevice.java.vm:104-111]`.
- **Target:** new `toolbelt/lint-driver-scaffold.sh` wired into `report-module.sh`, `types/driver-authoring.md` § 5.
- **Lint-able:** yes. **Fold type:** new lint + bats + doc line.
- **Fold status:** FOLDED (WU-P0-5, 2026-10-22; ndh Δ2): `toolbelt/lint-driver-scaffold.sh` + bats + § 5 line. `toolbelt/INDEX.md` row
  regenerated (6260fca). OWED: the `report-module.sh` wiring (outside the unit's surface).

### C89 — Client drivers: `BTcpCommConfig.serverEnabled` defaults to true
- **Sources:** ndh#Δ4 (primary)
- **Winning text:** a new network binds the listening port itself, so a CLIENT driver faults `Address already in use: JVM_Bind` when the real server runs on the same host; set `serverEnabled=false` at commissioning, and in a client-only driver override the default in the `B<X>TcpCommConfig` subclass (`[INFER]`, not yet built).
- **Evidence:** `[ev: engram obs 11476]`.
- **Target:** `types/driver-authoring.md` (NDriver section, C60) + commissioning checklist.
- **Lint-able:** partial — WARN a `BTcpCommConfig` subclass of a client-only module that keeps the default `[INFER]`. **Fold type:** doc section + checklist line.
- **Fold status:** FOLDED CORRECTED (WU-P0-5, 2026-10-22; ndh Δ4): the framework default is `false`
  (`BTcpCommConfig.java:94`); § 1.6 says a client driver keeps it false and commissioning checks it. The proposed WARN is moot.

### C90 — Operator-entered bounds must handle `min > max` explicitly
- **Sources:** slo#Δ4 (primary). Confirms scc Δ3 (C36: `isRunning()` guard first) live.
- **Winning text:** fault the output or refuse the write instead of silently pinning to one bound (the exam limiter held `out` at 75 for every input after a swapped entry); the remedy is `[INFER]`.
- **Evidence:** `[ev: live 2026-10-16 BLimiter Max=25 Min=75]`.
- **Target:** `types/logic.md` § Safety fail-modes & timers.
- **Lint-able:** no. **Fold type:** doc rule.
- **Fold status:** FOLDED (U11, 2026-10-23; slo Δ4): `types/logic.md` § Operator-entered bounds (symptom derived from
  `BLimiter.java:159-176`: `out` = 75 below 75 and 25 otherwise; remedy `[INFER]`; a validator misses link-fed bounds).

---

## P2 — new reference sections and templates

### C53 — Templates: shipping a template module, what breaks on upgrade
- **Sources:** pps#Δ2 (primary) · pps#Δ3 extends
- **Evidence:** `Templates/UpgradingADeployedTemplateTemplates-9A17381D.txt:25`, corpus B1325.
- **Target:** `types/logic-authoring.md:208-211`, `METHODOLOGY.md:44-52`. **Lint-able:** partial. **Fold type:** new doc section.
- **Fold status:** FOLDED (U10, `docs/r8-u10-platform-templates`, 2026-10-23; pps Δ2, Δ3): `types/logic-authoring.md` § Templates (palette entry ≠ `.ntpl`, Shipping a template module, What breaks on a module or template upgrade). Landed in logic-authoring only; the `METHODOLOGY.md:44-52` pointer is owed (outside the unit's surface). Owed Go lint G92 `ntpl-manifest-deps`. The `BConfigBinding` slot-rename breakage stays `[INFER]` (B1325-G1).

### C54 — Lexicon resolution chain and lexicon-only modules
- **Sources:** lux#Δ2 (primary) · pps#Δ5 extends
- **Evidence:** `Lexicon.java:313-334`, devguide `localization.txt:59-70,189-214`, `Lexicon/devkit-LexiconModuleBuilder.txt:14-43`, corpus B1305, B1330.
- **Target:** `types/structure.md` L4, `types/dashboard.md` i18n. **Lint-able:** partial. **Fold type:** new doc section.
- **Fold status:** FOLDED (U6, 2026-10-23; lux Δ2, pps Δ5): `types/lexicon.md` § 1-2 (lookup chain, locale order, lexicon-only modules, `!lexicon` overrides). Landed in a new card, not `types/structure.md` L4 (outside the unit's surface; a pointer line there is owed). Owed Go lint G71 `lexicon-manifest`.

### C55 — Runtime lexicon idiom and localizable exceptions
- **Sources:** lex#Δ2 (primary: `LexiconModule.getText`, `lint-lexicon-keys.sh`) · lux#Δ3 extends (`LocalizableRuntimeException` + `toString(cx)`)
- **Evidence:** course `Lab7/BWorkOrderService.java:107-185,272`, corpus B1236 §1236.4/§1236.7, B1305 §1305.1.
- **Target:** `types/observability.md` §3, new `toolbelt/lint-lexicon-keys.sh`. **Lint-able:** yes. **Fold type:** doc section + lint.
- **Fold status:** FOLDED (U6, 2026-10-23; lex Δ2, lux Δ3): `types/lexicon.md` § 3 (`LexiconModule`, MessageFormat, `LocalizableRuntimeException` + `toString(cx)`, `LexiconText.toLocalizable`); `types/observability.md` § 3 pointer owed (outside the surface). Owed Go lint G70 `lexicon-key` (replaces the proposed `toolbelt/lint-lexicon-keys.sh`).

### C56 — ux i18n: `lex!` plugin; lexicon content is public
- **Sources:** lux#Δ5 (primary)
- **Evidence:** `BLexiconRpc.java:44` (`unrestricted`), jsdoc `lex_lex.js.txt:157-208`.
- **Target:** `types/frontend-standard.md` § i18n. **Lint-able:** yes. **Fold type:** doc section + lint.
- **Fold status:** FOLDED (U6, 2026-10-23; lux Δ5): `types/lexicon.md` § 4 + `types/dashboard.md` § Module packaging pointer. **Added:** JS `lex.get` returns `def` or an empty string for a missing key (jsdoc `lex_lex.js.txt:422-436`). `types/frontend-standard.md` pointer owed (outside the surface). Owed Go lint G72 `lexicon-public`.

### C57 — Station diagnostics card, log levels, Spanish console levels
- **Sources:** poh#Δ4 (primary) · lux#Δ4 duplicate (lexicon diagnostics switch) · lex#Δ5 extends (INFO default, DebugService, FINE in hot paths) · lex#Δ4 extends (`triage-console.sh` `DETALLADO` etc.)
- **Evidence:** corpus B1255-B1257, B1251 §1251.9, B1236 §1236.2; `toolbelt/triage-console.sh:99-111`.
- **Target:** `types/observability.md` (new § Station diagnostics, §1), `types/issues-and-gotchas.md`, `toolbelt/triage-console.sh` + bats.
- **Lint-able:** yes (triage patterns, Spanish names; INFO in poll bodies). **Fold type:** doc section + script fix.
- **Fold status:** FOLDED doc half (U6, 2026-10-23; poh Δ4, lux Δ4, lex Δ5, lex Δ4 doc half): `types/diagnostics.md` (new card, in place of a new § in `types/observability.md`, outside the surface) + `types/issues-and-gotchas.md` K1. **Premise corrected:** "Cannot stop component" is any throwable out of `stopped()` (`EngineUtil.java:105,123`), missing `MANAGE_EXECUTION` is the course case; the JDK 8 `es` names (`Configurar`, `Muy Detallado`, `Más Detallado`) are now read from OpenJDK 1.8.0 `rt.jar`, no longer `[INFER]`. Owed: G73 (`triage-console.sh` levels + patterns), G74 `info-in-poll`.

### C58 — Build reference: supertype modules, part gradle.kts, JDK-8 preflight, region-less sources
- **Sources:** bdps#Δ1 (primary: supertype rule + build.sh hint) · afl#Δ4 extends (part build script, bajadoc packages) · bdps#Δ5 extends (`javaToolchains` preflight) · cex#Δ4 extends (K15: `slotomatic` before `jar` for region-less sources) · rsb#Δ4 extends (Windows-native build route from WSL for any Windows-side Gradle root: cmd.exe + gradlew.bat slotomatic/jar; back up `<niagara_home>/modules` first)
- **Evidence:** course `Lab Instructions.pdf` p81-86, `Slides/6` p6-18; `javap` on 4.15.6.20 jars; corpus B1235 §1235.2.
- **Target:** `types/module-wiring.md`, `types/structure.md`, `toolbelt/build.sh`, `toolbelt/preflight.sh`, `build-verify.md`, `METHODOLOGY.md` K15. **Lint-able:** yes. **Fold type:** doc section + script.
- **Fold status:** FOLDED doc (`docs/r8-u4-c58-build-reference`, 2026-10-23; bdps Δ1, Δ5, afl Δ4, cex Δ4, rsb Δ4, plus hw Δ3
  in-place refinement of C12): `types/module-wiring.md` § 1.1 (supertypes verified by javap on 4.15.6.20; part script from
  course `slides-rt.gradle.kts`), `build-verify.md` § Raw gradle, `BUILD-LOOP.md` § 4.b, `types/moduleTest.md` § Running
  from WSL, METHODOLOGY K15. Lands in `module-wiring.md`, not `structure.md`. OWED code: G60 supertype-hint (`build.sh`),
  G61 jdk-toolchain (`preflight.sh`), G62 bajadoc-packages, in `ledger.d/2026-10-23-r8-u4-c58-build-reference.md`.

### C59 — API stability: `com.tridium.*` is private; registry and agent lookups
- **Sources:** bdps#Δ3 (primary) · has#Δ9 extends (apiInfo labels, `getDef`, `AgentFilter`)
- **Evidence:** course `Lab Instructions.pdf` p83; devguide `apiInfo.txt:29-40,117-118`; corpus B1313.
- **Target:** `types/module-wiring.md` (new §), `toolbelt/verify-module.sh`. **Lint-able:** yes — WARN `import com.tridium.` outside allow-list. **Fold type:** doc section + lint.
- **Fold status:** DOC FOLDED, LINT OWED (doc: `docs/r8-pointer-lines`, 2026-10-23; bdps Δ3 doc half, has Δ9):
  `types/module-wiring.md` § 4.4 (defs, agents) and § 4.5 (stability labels, private API) + `types/structure.md`
  § Naming pointer. The WARN with the ndh Δ8 allow-list is owed Go lint G65 `private-api-import` (U5).

### C60 — NDriver section with the correlation rule
- **Sources:** ord#Δ3 (primary)
- **Evidence:** corpus B1239 §1239.12-13; course `Labs/Lab11 NDriver`.
- **Target:** `types/driver-authoring.md` (new §). **Lint-able:** yes. **Fold type:** new doc section + lint.
- **Fold status:** FOLDED doc (U5, 2026-10-23; ord Δ3, ndh Δ5, Δ7): `types/driver-authoring.md` § 1.7 + § 2 enum range
  step; the § 2 example's `write()` return was wrong (`false` for an enqueued write) and is corrected. OWED: lints G66, G67.

### C61 — BQL literal rules in anti-injection
- **Sources:** ord#Δ2 (primary)
- **Evidence:** corpus B1237 §1237.6, B1242 §1242.3 rules 62-63.
- **Target:** `types/security.md` §4. **Lint-able:** no. **Fold type:** doc section.
- **Fold status:** FOLDED (U5, 2026-10-23; ord Δ2): `types/security.md` § 4.1a; BQLTIME narrowing kept `[INFER]`.

### C62 — File-export recipe
- **Sources:** ord#Δ5 (primary)
- **Evidence:** corpus B1254 §1254.5 traps 11-15; course `Lab10 Jobs/BHistoriesToCsvJob.java`.
- **Target:** `types/logic-authoring.md` (new §). **Lint-able:** yes — WARN `.getBytes()` without charset, `BajaFileUtil.createFile(`. **Fold type:** doc section + lint.
- **Fold status:** FOLDED doc (U5, 2026-10-23; ord Δ5, shh Δ1): `types/logic-authoring.md` § File export. OWED: lints G68
  (`file-export`) and G69 (`lint-bql-string-concat.sh` history-prefix precision).

### C63 — Serving recipe B: `WEB-INF/web.xml` servlets and filters
- **Sources:** has#Δ2 (primary; its lint half is rejected — "every `<servlet-class>` needs `CsrfProtectedFilter`" fails 8 of 14 shipped Tridium `web.xml`; use wuh Δ1's scope: POST/PUT servlets only, see C31)
- **Evidence:** devguide `servlets/webModules.txt:12`; corpus B1315 §1315.1 rank 2, B1318 §1318.3 C3-C4.
- **Target:** `types/dashboard.md:29`. **Lint-able:** yes (scoped). **Fold type:** doc section + lint.
- **Fold status:** FOLDED (`docs/r8-u7-web-ui`, 2026-10-23; has Δ2): `types/dashboard.md` § ux web-module recipe
  (devguide `servlets/webModules.txt`), CSRF scope re-measured: 6 of the 14 shipped 4.15.6.20 `web.xml` map
  `CsrfProtectedFilter`, only POST, POST,PUT or GET,POST. OWED: lint G75 `web-xml-csrf-scope`
  (`ledger.d/2026-10-23-r8-u7-web-ui.md`); the `POST,PUT,DELETE` example in `types/security.md` § 3.1.

### C64 — SPA live-data subscription contract (bajaScript)
- **Sources:** nsto#Δ4 (primary)
- **Evidence:** corpus B1306 §1306.1 row 11, B1311 §1311.1 row 14.
- **Target:** `types/dashboard.md` (new § live data). **Lint-able:** yes (JS twin of subscribe-without-unsubscribe). **Fold type:** doc section + lint.
- **Fold status:** FOLDED (`docs/r8-u7-web-ui`, 2026-10-23; nsto Δ4): `types/dashboard.md` § Reading the station
  directly with BajaScript, subscription lifetime (bajaScript-ux `Subscriber.js`, `Component.js`). OWED: lint G76
  `spa-subscribe-without-unsubscribe`.

### C65 — Security hardening: FIPS, least privilege, platform TLS stores
- **Sources:** sta#Δ1 (primary, FIPS) · sta#Δ2 (least privilege) · sta#Δ4 (TLS stores)
- **Evidence:** `Fips/DevelopersNotes-51FBF6BE.txt:29-41`, `StationSecurity/role_*`, corpus B1326.
- **Target:** `types/security.md` (new §, §2, §10). **Lint-able:** yes — WARN two-arg `getInstance(…, "<provider>")`, `KeyStore.getInstance`. **Fold type:** doc sections + lint.
- **Fold status:** FOLDED (`docs/r8-u8-security-tags-analytics`, 2026-10-23; sta Δ1, Δ2, Δ4): `types/security.md` § 9.1 FIPS, § 2.6 least privilege, § 10.1
  TLS stores. Δ1 "RC4/DES" dropped (not in the guide). Lints owed G80 `jca-named-provider`, G81 `embedded-keystore`
  (`ledger.d/2026-10-23-r8-u8-sec-tags-analytics.md`). Same unit: § 3.1 CSRF example corrected to the shipped
  `httpMethod` values (no DELETE in any 4.15.6.20 `web.xml`).

### C66 — Equipment tagging for our logic points
- **Sources:** sta#Δ6 (primary)
- **Evidence:** `BContainmentRelation.java:65`, corpus B1329 rows 2-3.
- **Target:** `types/logic.md` tag recipe. **Lint-able:** partial. **Fold type:** doc section.
- **Fold status:** FOLDED (`docs/r8-u8-security-tags-analytics`, 2026-10-23; sta Δ6): `types/logic.md` § Our-modules tag recipe, code-verified
  (`BContainmentRelation.java:48-49`). No lint (partial, doc-only fold type).

### C67 — Analytics framework facts
- **Sources:** sta#Δ9 (primary)
- **Evidence:** `DeveloperAnalytics/buildingAlgorithmBlocks.txt:9,23`, `ordSchemes.txt:15,25`, corpus B1338.
- **Target:** `types/logic-authoring.md:31,101`. **Lint-able:** yes — grep `analytic:`. **Fold type:** doc section + lint.
- **Fold status:** FOLDED (`docs/r8-u8-security-tags-analytics`, 2026-10-23; sta Δ9): `types/logic-authoring.md` analytics bullet + § ORD scheme typology
  (no `analytic:` id, analytics-rt 4.15.6.20 `module.xml:106`). "Data definition per tag" not folded. Lint owed G82
  `analytic-ord-scheme`.

### C68 — BACnet/Modbus field rules
- **Sources:** drp#Δ5 (primary, BACnet export) · drp#Δ6 extends (MS/TP, Modbus byte order)
- **Evidence:** corpus B1327 rows 4-10.
- **Target:** `types/driver-authoring.md` (new §), `types/logic-authoring.md:641`. **Lint-able:** partial. **Fold type:** doc section.
- **Fold status:** FOLDED (U9, 2026-10-23; drp Δ5, Δ6): `types/driver-authoring.md` § 11.1-11.3, `types/logic-authoring.md` § Authoring a driver (`in8` vs the device array), `types/issues-and-gotchas.md` H4, H5. Owed Go lints G85 `bacnet-export-inN-contention`, G86 `bacnet-export-enum-range`.

### C69 — Cloud/MQTT telemetry and watch clients
- **Sources:** drp#Δ2 (primary, MQTT auth matrix + telemetry) · drp#Δ10 extends (lease formula, 503 = out of web threads, queue depth points)
- **Evidence:** corpus B1323 rows 3-6, B1336 rows 3,4,6.
- **Target:** `types/cloud-connector.md`, `types/driver-authoring.md`, `types/observability.md`. **Lint-able:** partial. **Fold type:** doc section.
- **Fold status:** FOLDED (U9, 2026-10-23; drp Δ2, Δ10): `types/cloud-connector.md` § 9 (auth matrix, 1883 default port, telemetry recipe), § 10 (lease formula, 503 back-off), `types/driver-authoring.md` § 11.4 (capability overrides, queue depth). `types/observability.md` pointer owed (outside the unit's surface). Owed Go lint G87 `mqtt-tls-port`.

### C70 — RDBMS: supported databases and export from code
- **Sources:** shl#Δ8 (primary) · r3di#Δ6 extends (`histories.getExportDescriptorType()`, licence on the database)
- **Evidence:** `Rdbms/DatabaseRequirement-Rdbms-7C2CBFED.txt:16,46`, `BHistoryExport.java:19-32`, `BRdbms.java:256-301`, corpus B1340, B1352.
- **Target:** `types/third-party-libraries.md:176-177`, `types/utility-lib.md:8`, `types/cloud-connector.md`. **Lint-able:** partial. **Fold type:** doc section.
- **Fold status:** FOLDED (U9, 2026-10-23; shl Δ8, r3di Δ6): `types/cloud-connector.md` § 11. Landed there, not at `types/third-party-libraries.md:176-177` / `types/utility-lib.md:8` (outside the unit's surface; pointer lines owed). Owed Go lint G88 `rdbms-dialect-descriptor`.

### C71 — Extending jsonToolkit
- **Sources:** shl#Δ9 (primary: decision rule + extension points) · r3pec#Δ2 corrects (`overrideValueEncoding`, not final `doOverride`) · r3-report#R4
- **Evidence:** `BIJsonProperty.java:15-30`, `javap BTypeOverride`, corpus B1322, B1350.
- **Target:** `types/dashboard.md:29`, `types/cloud-connector.md` §1, new § "Extending jsonToolkit". **Lint-able:** yes. **Fold type:** new doc section + lint.
- **Fold status:** FOLDED (U9, 2026-10-23; shl Δ9, r3pec Δ2, r3-report#R4): new `types/json-toolkit.md` § 1-3 + `types/cloud-connector.md` pointer. `types/dashboard.md:29` pointer owed (outside the unit's surface). Owed Go lint G89 `jsontoolkit-licence-note`; the `doOverride`/dependency halves are compiler-enforced.

### C72 — Web UI reference: profiles, Px rules, charts, Mobile deprecated
- **Sources:** wuh#Δ3 (primary, profiles) · wuh#Δ4 (Px rules) · wuh#Δ8 (stock ChartWidget first) · wuh#Δ9 (Mobile profile/Px deprecated since 4.6)
- **Evidence:** corpus B1328, B1339, B1342.
- **Target:** `types/dashboard.md`, `types/wb-widgets.md:360`, `types/frontend-standard.md:22,175`. **Lint-able:** partial. **Fold type:** doc sections + lint.
- **Fold status:** FOLDED (`docs/r8-u7-web-ui`, 2026-10-23; wuh Δ3, Δ4, Δ8, Δ9): `types/dashboard.md` § ux Px rules,
  § Charts on an HMI, § Deploy on a JACE (web profile); `types/frontend-standard.md` § Profiles, § Technology
  choices. Δ4 landed in `types/dashboard.md`, not `types/wb-widgets.md` (outside the unit's surfaces). OWED: lints G77
  `px-portability`, G78 `mobile-deprecated`; the `types/wb-widgets.md` § PX authoring "all positions are absolute" line.

### C73 — Platform install and runtime profiles
- **Sources:** pps#Δ9 (primary)
- **Evidence:** `Platform/aPlat4PlatformAdminConfigureRuntimeProfiles.txt`, corpus B1341 rows 2-6.
- **Target:** `types/distribution.md` Software Manager §, `BUILD-LOOP.md:207`. **Lint-able:** no. **Fold type:** doc section.
- **Fold status:** FOLDED (U10, 2026-10-23; pps Δ9): new `types/platform.md` § 1 (runtime profiles, Configure Runtime Profiles wizard) and § 2 (Software Manager install/upgrade/uninstall, Station Copier); pointer from `types/distribution.md` intro. `BUILD-LOOP.md:207` pointer owed (outside the surface).

### C74 — Workbench Commands card
- **Sources:** rcf#Δ4 (primary)
- **Evidence:** `Command.java:61-64`, corpus B1317 §1317.1.
- **Target:** `types/wb-widgets.md` (new §). **Lint-able:** partial. **Fold type:** doc section.
- **Fold status:** FOLDED (U10, 2026-10-23; rcf Δ4): `types/wb-widgets.md` § Commands (`Command.java:61-64` four keys, null on a missing key, a bad accelerator only prints). Owed Go lint G91 `command-lexicon-keys`.

### C75 — `BHysteresis`/`BLimiter` are 4.15+
- **Sources:** rcf#Δ8 (primary)
- **Evidence:** `BHysteresis.java:35,292`, corpus B1319 §1319.3 D3.
- **Target:** `types/logic.md` (version note). **Lint-able:** yes. **Fold type:** doc note + lint.
- **Fold status:** FOLDED (U10, 2026-10-23; rcf Δ8): `types/logic.md` after the kitControl status families (`@since Niagara 4.15`; jar check: absent in 4.3.58.18 / 4.13.2.18 / 4.14.0.162, present in 4.15.3.28 / 4.15.6.20; swapped limits fault; latch not persisted). No kit lint checks type availability per target release; owed Go lint G90 `since-version-type`.

### C91 — NDriver device/point manager and discovery recipe
- **Sources:** ndh#Δ6 (primary). Extends C60 (NDriver section) and the manager recipe in `types/wb-widgets.md`.
- **Winning text:** device columns by property facet `SfUtil.incl(SfUtil.MGR_EDIT)`; point manager = `@AgentOn(types = {"<mod>:<X>PointDeviceExt", "<mod>:<X>PointFolder"})` extending `BNPointManager` with `NPointModel`/`NPointController`; extension columns via `MgrColumn.PropPath(new Property[] {BControlPoint.proxyExt, …}, MgrColumn.EDITABLE)`; `getNewTypes()` restricts New; `doSubmitDiscoveryJob(prefs, cx)` submits with `job.submit(cx)`.
- **Evidence:** `[ev: engram obs 11479]`.
- **Target:** `types/driver-authoring.md` (NDriver section), `types/wb-widgets.md` § Manager recipe.
- **Lint-able:** yes — WARN `.submit(null)` inside a `doSubmitDiscoveryJob` body. **Fold type:** doc section + lint.
- **Fold status:** FOLDED (WU-P0-5, 2026-10-22; ndh Δ6): § 1.6; the `.submit(null)` WARN lives in `lint-driver-scaffold.sh`.
  OWED: the `types/wb-widgets.md` § Manager recipe half.

### C92 — Orient: report the newest working-tree file before trusting a summary
- **Sources:** hwf#Δ2 (primary)
- **Winning text:** when resuming hands-on work, list the most recently modified files and read the latest transcript before trusting a session summary; the orient line gains `newest: <file> <mtime>`.
- **Evidence:** `[ev: engram obs 11477]`.
- **Target:** `BUILD-LOOP.md` § 0.a Orient from BUILD-STATE; optional `toolbelt/orient-guard.sh --recent <root>`.
- **Lint-able:** no (procedure); the orient-guard option is the mechanical form. **Fold type:** doc procedure (+ optional helper).
- **Fold status:** FOLDED (U11, 2026-10-23; hwf Δ2): `METHODOLOGY.md` § Hands-on work (resume from the workspace; `newest:` in the orient line).
  Landed there, not in `BUILD-LOOP.md` § 0.a (outside the surface; pointer P2 owed). Owed G94 `orient-recent`.

### C93 — Shared-host rule: no heavy background jobs while Workbench or a station is open
- **Sources:** hwf#Δ4 (primary)
- **Winning text:** defer or cap (`nice`, `-j` at half the cores) index builds, full suites and parallel batches while `wb.exe` or `station.exe` runs on the same laptop.
- **Evidence:** `[ev: engram obs 11095]`.
- **Target:** `ORCHESTRATION.md` § 3 Delegation triggers, `toolbelt/preflight.sh`.
- **Lint-able:** partial — preflight WARN when `tasklist.exe` shows `wb.exe`/`station.exe` before a heavy job. **Fold type:** doc rule + preflight row.
- **Fold status:** FOLDED (U11, 2026-10-23; hwf Δ4): `METHODOLOGY.md` § Hands-on work (shared host; `wb.exe`, `wb_w.exe`, `station.exe` from the
  4.15.6.20 `bin/`). Landed there, not in `ORCHESTRATION.md` § 3 (pointer P3 owed). Owed preflight row G95 `shared-host`.

---

## P3 — horizons and nice-to-have

### C76 — Horizons lists
- **Sources:** poh#Δ6 (primary, course capabilities not yet used) · has#Δ10 extends (promiseUtils, `LocalizableError`, CSRF constant, login branding, popup editors)
- **Target:** `corpus-index.md`, `toolbelt/kit-ticket.sh`, `types/dashboard.md`, `types/theme.md`, `types/wb-widgets.md`. **Lint-able:** no. **Fold type:** kit-tickets + small doc notes.
- **Fold status:** FOLDED (U10, 2026-10-23; poh Δ6, has Δ10): new `types/horizons.md` (§ 1 course capabilities with the kit recipe per row, § 2 promiseUtils, `LocalizableError`, CSRF header constant, login branding) + `types/wb-widgets.md` popup-editor note. Owed outside the surface: `corpus-index.md` row group, `kit-ticket.sh` tickets, pointers from `types/dashboard.md` DUX-TEST1 and `types/theme.md`.

### C77 — Preserve author-set slot flags on rebuild (scope corrected)
- **Sources:** sta#Δ10 (primary, accepted-corrected) · r3-report#R5
- **Winning text:** keep as a design note only; its evidence (`forceUpdate` resets flags) applies to R2 oBIX proxy points, not to N4 drivers in general.
- **Evidence:** `EngNotes/aForceUpdateAction.txt:2,13-15,27`, corpus B1353 §1353.2 row 8.
- **Target:** `types/value-types.md` or `types/logic-authoring.md`. **Lint-able:** no. **Fold type:** doc note.
- **Fold status:** FOLDED (`docs/r8-u8-security-tags-analytics`, 2026-10-23; sta Δ10, r3-report R5): `types/logic-authoring.md` § Programmatic slot-name
  rules, design note [INFER] with the R2-oBIX-only scope.

### C78 — ExportTags makes a copy, not a proxy
- **Sources:** sta#Δ8 (primary). **Target:** `types/distribution.md`. **Lint-able:** no. **Fold type:** doc note.
- **Fold status:** FOLDED (`docs/r8-u8-security-tags-analytics`, 2026-10-23; sta Δ8): landed in `types/logic.md` § Ship a tag dictionary; `types/distribution.md`
  was outside the unit's edit surface; its one-line pointer landed with the pointer unit (`docs/r8-pointer-lines`).

### C79 — Coexisting with the Tridium DashboardService
- **Sources:** wuh#Δ10 (primary). **Evidence:** `EngNotes/dashboard-DashboardService.txt:14`. **Target:** `types/dashboard.md`. **Lint-able:** partial. **Fold type:** doc note.
- **Fold status:** FOLDED (`docs/r8-u7-web-ui`, 2026-10-23; wuh Δ10): `types/dashboard.md` § Deploy on a JACE
  (clean-text anchor `dashboard-DashboardService.txt:7`). The lint half stays human review: the state lives in the
  station home, not the module tree.

### C80 — NRE memory budget and flash writes on a JACE
- **Sources:** pps#Δ10 (primary). **Evidence:** corpus B1341 row 7, B1284. **Target:** `types/math-control-libraries.md:43`. **Lint-able:** no. **Fold type:** doc note.
- **Fold status:** FOLDED (U10, 2026-10-23; pps Δ10): new `types/platform.md` § 3 (NRE pool defaults 0/384/128/32 MB, reboot to apply, flash writes; flash-wear advice `[INFER]`). `types/math-control-libraries.md` GC-note pointer landed with the pointer unit (`docs/r8-pointer-lines`).

---

## Open decisions and verifications

1. **Count of record:** 168, not 178 (see § Count). The ffcddad message is wrong; the retros are right.
2. **C01 lint downgrade** turns a FAIL gate (`jdk-thread`) into a WARN for most code — this changes what blocks client modules.
   **Answered 2026-10-22 (Integrador):** FAIL stays only for a thread-modifying call in a module that requests no
   `MANAGE_EXECUTION`; everything else WARNs. Landed on `docs/r8-t9b-p0-lints`.
3. **C13** retires the DashboardPan headers gap (ODA2-G1/G2) — the client tracking that cites it needs the same close.
4. **C28** unsigned-group outcome on 4.15 (WARNING skip vs station halt) needs a target check before the kit states it.
5. **C31** live proof of the keep-alive route (niagara-research B1315-G1) is still open.
6. **C52 / tpb Δ3** touch the client: palette roots, logger names (DebugService categories change) and the client root signing config — client decisions, not kit folds.

---

## Proof

Command (from the worktree root):

```
bash build-n4-module-kit/odd/tasks/kit-improvement-candidates-2026-10-15.check.sh
```

Recorded result (2026-10-15, worktree `retros/2026-10-15-course-devguide`, retros at ffcddad):

```
retro files: 23
source deltas: 168
mapped: 168
missing: 0
extra (in appendix, not in retros): 0
duplicate appendix rows: 0
undefined C-ids: none
tier mismatches: none
C-items defined: 80
source deltas per tier: P0=56 P1=64 P2=42 P3=6
RESULT: PASS (missing=0)
```

The check has teeth: deleting one Appendix A row gives `missing: 1` → FAIL; changing one row's tier gives
`tier mismatches: C41(P2≠P1)` → FAIL; renaming a C heading gives `undefined C-ids: C80` → FAIL (each mutation run once,
then restored).

---

## Appendix A — coverage: every source delta → its C-item

Status: `accepted` · `accepted-corrected` (a later source or evidence corrects part of it; the C-item states the winner) ·
`partial-reject` (one named part is rejected with evidence, the rest is kept) · `superseded-in-part`.

| Source ID | Target kit file(s) | Summary | C-item | Tier | Status |
|---|---|---|---|---|---|
| `devguide415-horizons-authoring-security#Δ1` | types/dashboard.md, types/security.md | Session lifetime for kiosk/HMI; polls do not reset idle timeout | C31 | P1 | superseded-in-part (poll-servlet route rejected, wuh Δ1 / r3wsh Δ1) |
| `devguide415-horizons-authoring-security#Δ2` | types/dashboard.md:29 | Serving recipe B via WEB-INF/web.xml | C63 | P2 | partial-reject (CSRF-on-every-servlet lint; scoped by wuh Δ1) |
| `devguide415-horizons-authoring-security#Δ3` | types/logic-authoring.md:536-545, types/dashboard.md | History `_cfg0` split rule; exporter CSV URL | C35 | P1 | accepted |
| `devguide415-horizons-authoring-security#Δ4` | build-verify.md:132, types/distribution.md:110-115, types/third-party-libraries.md:148-151 | Six signing-required permission groups; trust text | C28 | P0 | accepted (4.15 outcome to verify) |
| `devguide415-horizons-authoring-security#Δ5` | types/structure.md | Security Manager card | C47 | P1 | accepted |
| `devguide415-horizons-authoring-security#Δ6` | types/structure.md:8,338, types/module-wiring.md | module-include.xml holds only types | C25 | P0 | accepted |
| `devguide415-horizons-authoring-security#Δ7` | types/structure.md:9,176-250, types/third-party-libraries.md:38-69 | Doc part allowed, help.guide.base optional, uberjar first | C26 | P0 | accepted |
| `devguide415-horizons-authoring-security#Δ8` | types/structure.md | `se` = Java SE APIs; part dependency matrix | C24 | P0 | accepted |
| `devguide415-horizons-authoring-security#Δ9` | types/module-wiring.md | API stability labels; registry/agent lookups | C59 | P2 | folded (pointer unit, `types/module-wiring.md` § 4.4-4.5) |
| `devguide415-horizons-authoring-security#Δ10` | types/dashboard.md, types/theme.md, types/wb-widgets.md | Small horizons (promiseUtils, LocalizableError, login branding) | C76 | P3 | folded (U10, `types/horizons.md` § 2 + `types/wb-widgets.md` popup editor) |
| `devguide415-lexicons-ux-i18n#Δ1` | types/dashboard.md:203, types/value-types.md:51, toolbelt/slot-coverage.sh, toolbelt/generate-wiring-map.sh | Slot/enum keys are bare names; drop Type.slot advice | C10 | P0 | accepted |
| `devguide415-lexicons-ux-i18n#Δ2` | types/structure.md L4 | Lexicon resolution and override chain | C54 | P2 | folded (U6, `types/lexicon.md` § 1); manifest lint owed G71 |
| `devguide415-lexicons-ux-i18n#Δ3` | types/observability.md | LocalizableRuntimeException idiom | C55 | P2 | folded (U6, `types/lexicon.md` § 3); key lint owed G70 |
| `devguide415-lexicons-ux-i18n#Δ4` | types/issues-and-gotchas.md | Lexicon diagnostics triage note | C57 | P2 | folded (U6, `types/issues-and-gotchas.md` K1) |
| `devguide415-lexicons-ux-i18n#Δ5` | types/frontend-standard.md | ux i18n with lex!; LexiconRpc unrestricted | C56 | P2 | folded (U6, `types/lexicon.md` § 4); lint owed G72 |
| `devguide415-nav-subscriptions-tags-ords#Δ1` | types/driver-authoring.md:54-70 | Fix inverted readSubscribed/readUnsubscribed | C15 | P0 | accepted |
| `devguide415-nav-subscriptions-tags-ords#Δ2` | types/logic-authoring.md:9-24 | Subscribed semantics, leases, SELF_EVENTS ids | C20 | P0 | accepted |
| `devguide415-nav-subscriptions-tags-ords#Δ3` | types/driver-authoring.md:234-240 | Virtual component ORD form and cache | C20 | P0 | accepted |
| `devguide415-nav-subscriptions-tags-ords#Δ4` | types/dashboard.md | SPA bajaScript subscription contract | C64 | P2 | accepted |
| `devguide415-nav-subscriptions-tags-ords#Δ5` | types/structure.md:253-287 | Nav file version + cache refresh | C17 | P0 | accepted |
| `devguide415-nav-subscriptions-tags-ords#Δ6` | types/logic-authoring.md:37-39,142, types/security.md | getNavChildren permission-blind; nav: needs base | C18 | P0 | accepted |
| `devguide415-nav-subscriptions-tags-ords#Δ7` | types/logic-authoring.md:76,120, types/moduleTest.md | BOrd.make only wraps; unmounted BQL test | C18 | P0 | accepted |
| `devguide415-nav-subscriptions-tags-ords#Δ8` | types/security.md, types/module-wiring.md | OrdTarget permission rule; custom scheme contract | C39 | P1 | accepted |
| `devguide415-nav-subscriptions-tags-ords#Δ9` | types/logic.md:359-376, types/security.md:220 | Rewrite tag-dictionary recipe; NEQL | C21 | P0 | accepted |
| `devguide415-nav-subscriptions-tags-ords#Δ10` | types/logic.md, types/logic-authoring.md:148-154 | Tags-in-code card; hierarchy: client-only | C21 | P0 | accepted |
| `devguide415-r2-control-alarms-schedules#Δ1` | types/logic.md:324-325 | ackRequired version-qualified; timeDelay; stale | C22 | P0 | accepted |
| `devguide415-r2-control-alarms-schedules#Δ2` | types/logic.md | Alarm-ext chooser and alarmInhibit recipe | C32 | P1 | accepted |
| `devguide415-r2-control-alarms-schedules#Δ3` | types/logic-authoring.md:536-545 | History ext rules (disabled, historyName, capacity) | C35 | P1 | accepted |
| `devguide415-r2-control-alarms-schedules#Δ4` | BUILD-LOOP.md | Size the file alarm DB at commissioning | C35 | P1 | accepted |
| `devguide415-r2-control-alarms-schedules#Δ5` | types/logic.md | Reuse LeadLag/Sequence/Interstart | C33 | P1 | accepted-corrected (Interstart min on/off default 0, r3pec Δ4) |
| `devguide415-r2-control-alarms-schedules#Δ6` | types/logic.md | Timer outputs TRANSIENT; Counter persists | C33 | P1 | accepted |
| `devguide415-r2-control-alarms-schedules#Δ7` | types/logic.md:332-333 | LoopPoint zero gains, 0.5 s ticket | C33 | P1 | accepted |
| `devguide415-r2-control-alarms-schedules#Δ8` | types/logic.md | Tstat no lockout; ext ordering | C33 | P1 | accepted |
| `devguide415-r2-control-alarms-schedules#Δ9` | types/logic.md | TriggerSchedule for wall-clock defrosts | C34 | P1 | accepted |
| `devguide415-r2-control-alarms-schedules#Δ10` | types/logic-authoring.md:298 | Weekly schedule defaultOutput; sharing | C34 | P1 | accepted |
| `devguide415-r2-drivers-protocols#Δ1` | types/cloud-connector.md, types/third-party-libraries.md:143,171 | MQTT native first; module abstractMqttDriver | C27 | P0 | accepted |
| `devguide415-r2-drivers-protocols#Δ2` | types/cloud-connector.md | MQTT auth matrix and telemetry recipe | C69 | P2 | folded (U9, `types/cloud-connector.md` § 9); lint owed G87 |
| `devguide415-r2-drivers-protocols#Δ3` | types/driver-authoring.md:127-132 | staleTime 0 = stale immediately | C16 | P0 | accepted |
| `devguide415-r2-drivers-protocols#Δ4` | types/driver-authoring.md, types/dashboard.md | BACnet statusFlags facet for health | C45 | P1 | accepted-corrected (COV and polled alike, r3di Δ3) |
| `devguide415-r2-drivers-protocols#Δ5` | types/driver-authoring.md, types/logic-authoring.md:641 | Exposing values over BACnet | C68 | P2 | folded (U9, `types/driver-authoring.md` § 11.1, H5); lints owed G85, G86 |
| `devguide415-r2-drivers-protocols#Δ6` | types/driver-authoring.md | MS/TP and Modbus rules | C68 | P2 | folded (U9, `types/driver-authoring.md` § 11.2-11.3, H4) |
| `devguide415-r2-drivers-protocols#Δ7` | types/driver-authoring.md | Ping rules (faulted never pinged, alarm delay) | C44 | P1 | accepted |
| `devguide415-r2-drivers-protocols#Δ8` | types/driver-authoring.md:121-122,205-213 | pollEnabled on poll service; tuning sizing | C16 | P0 | accepted |
| `devguide415-r2-drivers-protocols#Δ9` | types/dashboard.md:23, types/logic-authoring.md:554, types/security.md | oBIX exposure rules | C46 | P1 | accepted |
| `devguide415-r2-drivers-protocols#Δ10` | types/cloud-connector.md, types/driver-authoring.md, types/observability.md | Watch/lease clients; 503 back-off | C69 | P2 | folded (U9, `types/cloud-connector.md` § 10, `types/driver-authoring.md` § 11.4) |
| `devguide415-r2-packaging-platform-signing#Δ1` | types/distribution.md:35-45 | Fix dist attribute table (noStation) | C23 | P0 | accepted-corrected (missing attribute = false, r3pec Δ1) |
| `devguide415-r2-packaging-platform-signing#Δ2` | types/logic-authoring.md:208-211 | Shipping a template module | C53 | P2 | folded (U10, `types/logic-authoring.md` § Shipping a template module); lint owed G92 |
| `devguide415-r2-packaging-platform-signing#Δ3` | METHODOLOGY.md:44-52 | What breaks on module upgrade | C53 | P2 | folded (U10, `types/logic-authoring.md` § What breaks on a module or template upgrade) |
| `devguide415-r2-packaging-platform-signing#Δ4` | types/distribution.md, types/logic.md:185, types/structure.md:133 | Palette entry ≠ .ntpl; dangling §Templates; provisioning | C29 | P0 | accepted |
| `devguide415-r2-packaging-platform-signing#Δ5` | types/structure.md, types/dashboard.md | Lexicon-only modules | C54 | P2 | folded (U6, `types/lexicon.md` § 2); lint owed G71 |
| `devguide415-r2-packaging-platform-signing#Δ6` | build-verify.md, types/distribution.md:100 | Signing Service ≠ module signing; mode × status | C48 | P1 | accepted |
| `devguide415-r2-packaging-platform-signing#Δ7` | types/logic-authoring.md:917 | Program objects signed; load failure causes | C48 | P1 | accepted |
| `devguide415-r2-packaging-platform-signing#Δ8` | types/structure.md:300 | se is Supervisor (Windows) only | C24 | P0 | accepted |
| `devguide415-r2-packaging-platform-signing#Δ9` | types/distribution.md, BUILD-LOOP.md:207 | Runtime profiles and install rules | C73 | P2 | folded (U10, `types/platform.md` § 1-2) |
| `devguide415-r2-packaging-platform-signing#Δ10` | types/math-control-libraries.md:43 | NRE memory budget; flash writes | C80 | P3 | folded (U10, `types/platform.md` § 3) |
| `devguide415-r2-runtime-contracts-formats#Δ1` | types/actions.md | post() first-wins vs CoalesceQueue last-wins | C38 | P1 | accepted |
| `devguide415-r2-runtime-contracts-formats#Δ2` | types/security.md, types/logic-authoring.md:73 | Null Context = BPermissions.all | C39 | P1 | accepted |
| `devguide415-r2-runtime-contracts-formats#Δ3` | types/logic.md:111-143 | Call checkLink before adding links in code | C08 | P0 | accepted |
| `devguide415-r2-runtime-contracts-formats#Δ4` | types/wb-widgets.md | Commands card (four key suffixes) | C74 | P2 | folded (U10, `types/wb-widgets.md` § Commands); lint owed G91 |
| `devguide415-r2-runtime-contracts-formats#Δ5` | types/structure.md:253-287 | Nav files load only via explicit ORDs | C17 | P0 | accepted |
| `devguide415-r2-runtime-contracts-formats#Δ6` | types/logic.md:399-418 | BFormat line-level facts | C07 | P0 | folded (WU-P0-3 + lints) |
| `devguide415-r2-runtime-contracts-formats#Δ7` | types/logic.md:252-256,335 | kitControl status families / propagateFlags defaults | C05 | P0 | accepted |
| `devguide415-r2-runtime-contracts-formats#Δ8` | types/logic.md | BHysteresis/BLimiter 4.15+ | C75 | P2 | folded (U10, `types/logic.md` 4.15+ note); lint owed G90 |
| `devguide415-r2-runtime-contracts-formats#Δ9` | types/logic-authoring.md | NaN/null fallback before staging comparisons | C50 | P1 | accepted |
| `devguide415-r2-security-tags-analytics#Δ1` | types/security.md | FIPS-mode constraints | C65 | P2 | folded (U8, `types/security.md` § 9.1); lint owed G80 |
| `devguide415-r2-security-tags-analytics#Δ2` | types/security.md §2 | Least privilege for module users | C65 | P2 | folded (U8, `types/security.md` § 2.6) |
| `devguide415-r2-security-tags-analytics#Δ3` | types/security.md §3, types/dashboard.md | 4.15 absolute logoff ends HMI sessions | C31 | P1 | accepted |
| `devguide415-r2-security-tags-analytics#Δ4` | types/security.md §10 | Use platform TLS stores | C65 | P2 | folded (U8, `types/security.md` § 10.1); lint owed G81 |
| `devguide415-r2-security-tags-analytics#Δ5` | types/logic.md, types/cloud-connector.md:206-217 | Tag dictionary licence (2 dictionaries) | C21 | P0 | accepted |
| `devguide415-r2-security-tags-analytics#Δ6` | types/logic.md | Equipment tagging with direct equipRef | C66 | P2 | folded (U8, `types/logic.md` § Our-modules tag recipe) |
| `devguide415-r2-security-tags-analytics#Δ7` | types/logic.md, types/logic-authoring.md:181 | Tag rules and hierarchies | C21 | P0 | accepted |
| `devguide415-r2-security-tags-analytics#Δ8` | types/distribution.md | ExportTags ComponentTag copies | C78 | P3 | folded (U8, `types/logic.md` § Ship a tag dictionary) |
| `devguide415-r2-security-tags-analytics#Δ9` | types/logic-authoring.md:31,101 | Analytics facts and ORD schemes | C67 | P2 | folded (U8, `types/logic-authoring.md` analytics + § ORD scheme typology); lint owed G82 |
| `devguide415-r2-security-tags-analytics#Δ10` | types/value-types.md or types/logic-authoring.md | Preserve slot flags on rebuild | C77 | P3 | accepted-corrected (forceUpdate scope = R2 oBIX only, B1353); folded (U8, `types/logic-authoring.md` § Programmatic slot-name rules) |
| `devguide415-r2-shipped-libraries#Δ1` | types/third-party-libraries.md, types/module-wiring.md | bin/ext first; nre("g:a") | C42 | P1 | accepted |
| `devguide415-r2-shipped-libraries#Δ2` | types/cloud-connector.md:61-63, types/third-party-libraries.md:167,174 | okhttp ships in bin/ext, not net-rt | C27 | P0 | accepted |
| `devguide415-r2-shipped-libraries#Δ3` | types/third-party-libraries.md | bin/ext version drift | C42 | P1 | accepted |
| `devguide415-r2-shipped-libraries#Δ4` | types/third-party-libraries.md | Tridium wrapper module table | C42 | P1 | accepted |
| `devguide415-r2-shipped-libraries#Δ5` | types/third-party-libraries.md:30,143,159-160,171 | Stale versions and MQTT module name | C27 | P0 | accepted |
| `devguide415-r2-shipped-libraries#Δ6` | types/module-wiring.md | Wrapper does not grant permissions | C42 | P1 | accepted |
| `devguide415-r2-shipped-libraries#Δ7` | types/cloud-connector.md §7, types/issues-and-gotchas.md | Outbound-HTTP decision ladder | C43 | P1 | accepted-corrected ((b) gate closed by default, r3di Δ4) |
| `devguide415-r2-shipped-libraries#Δ8` | types/third-party-libraries.md:176-177, types/utility-lib.md:8 | Qualify native rdb (no Postgres) | C70 | P2 | folded (U9, `types/cloud-connector.md` § 11); lint owed G88 |
| `devguide415-r2-shipped-libraries#Δ9` | types/dashboard.md:29, types/cloud-connector.md | jsonToolkit decision rule and extension card | C71 | P2 | folded (U9, `types/json-toolkit.md` § 1-2, corrected per r3pec Δ2); lint owed G89 |
| `devguide415-r2-web-ui-hmi#Δ1` | types/security.md:138-152, types/dashboard.md | Correct has Δ1/Δ2 (UserActivityFilter, CSRF scope) | C31 | P1 | accepted |
| `devguide415-r2-web-ui-hmi#Δ2` | types/theme.md | Theme module rules | C30 | P0 | accepted |
| `devguide415-r2-web-ui-hmi#Δ3` | types/dashboard.md | Web profiles and theme per profile | C72 | P2 | accepted |
| `devguide415-r2-web-ui-hmi#Δ4` | types/wb-widgets.md:360 | Px rules | C72 | P2 | accepted |
| `devguide415-r2-web-ui-hmi#Δ5` | types/security.md | Hx headers; embed via frame-ancestors | C13 | P0 | accepted |
| `devguide415-r2-web-ui-hmi#Δ6` | types/dashboard.md | Kiosk panel user settings | C31 | P1 | accepted |
| `devguide415-r2-web-ui-hmi#Δ7` | types/security.md | Client-certificate SSO for panels | C31 | P1 | accepted |
| `devguide415-r2-web-ui-hmi#Δ8` | types/dashboard.md, types/frontend-standard.md:175 | Stock ChartWidget first | C72 | P2 | accepted |
| `devguide415-r2-web-ui-hmi#Δ9` | types/frontend-standard.md:22, types/dashboard.md | Mobile profile/Px deprecated | C72 | P2 | accepted |
| `devguide415-r2-web-ui-hmi#Δ10` | types/dashboard.md | Coexisting with DashboardService | C79 | P3 | accepted |
| `devguide415-r3-drivers-integration#Δ1` | types/driver-authoring.md §5 | readFail does not affect device health | C44 | P1 | accepted |
| `devguide415-r3-drivers-integration#Δ2` | types/logic-authoring.md:646 | pingEnabled is ON by default | C14 | P0 | accepted |
| `devguide415-r3-drivers-integration#Δ3` | types/driver-authoring.md:129 | statusFlags applies to COV and polled | C45 | P1 | accepted |
| `devguide415-r3-drivers-integration#Δ4` | types/cloud-connector.md §7 | enableNonDriverClients default false; mountAtOrd | C43 | P1 | accepted |
| `devguide415-r3-drivers-integration#Δ5` | types/cloud-connector.md, types/security.md | OAuth2 via oauth2-rt | C43 | P1 | accepted |
| `devguide415-r3-drivers-integration#Δ6` | types/cloud-connector.md | RDBMS export from code | C70 | P2 | folded (U9, `types/cloud-connector.md` § 11); lint owed G88 |
| `devguide415-r3-packaging-extension-control#Δ1` | types/distribution.md:37 | noRunningApp enforced; missing = false | C23 | P0 | accepted |
| `devguide415-r3-packaging-extension-control#Δ2` | types/ (new §) | Extending jsonToolkit (overrideValueEncoding) | C71 | P2 | folded (U9, `types/json-toolkit.md` § 3) |
| `devguide415-r3-packaging-extension-control#Δ3` | types/logic.md | LeadLag feedback links mandatory | C33 | P1 | accepted |
| `devguide415-r3-packaging-extension-control#Δ4` | types/logic.md | Interstart min on/off inherited, default 0 | C33 | P1 | accepted |
| `devguide415-r3-web-session-headers#Δ1` | types/dashboard.md | Session lifetime ladder; a poll is not activity | C31 | P1 | accepted |
| `devguide415-r3-web-session-headers#Δ2` | types/security.md, commissioning | 4.15 absolute logoff flags | C31 | P1 | accepted |
| `devguide415-r3-web-session-headers#Δ3` | types/security.md:360-380, toolbelt/lint-servlet.sh | Station headers already apply; remove gap + snippet | C13 | P0 | accepted |
| `devguide415-r3-web-session-headers#Δ4` | types/theme.md, types/frontend-standard.md | Servlet SPA not themed automatically | C30 | P0 | accepted |
| `devguide415-r3-web-session-headers#Δ5` | types/security.md | Never X-Frame Any; frame-ancestors | C13 | P0 | accepted |
| `devguide415-r3-web-session-headers#Δ6` | types/security.md | oBIX HIDDEN is not access control | C46 | P1 | accepted |
| `devguide415-runtime-api-corrections#Δ1` | types/logic.md:346, types/math-control-libraries.md:86 | Remove BWorker.post/postAsync; ExecutorUtil | C02 | P0 | folded (WU-P0-1 + lints); rt WARN rejected |
| `devguide415-runtime-api-corrections#Δ2` | types/logic-authoring.md:216-233 | Replace BJobs.submit; BRunnableJob | C03 | P0 | folded (WU-P0-1 + lints) |
| `devguide415-runtime-api-corrections#Δ3` | types/logic.md | Absolute schedules need clockChanged | C34 | P1 | accepted |
| `devguide415-runtime-api-corrections#Δ4` | types/logic.md:233-243 | Fix the BStatus bit table | C04 | P0 | accepted |
| `devguide415-runtime-api-corrections#Δ5` | types/logic.md:252-256 | Fix propagateFlags recipe (no BStatus.and) | C05 | P0 | accepted |
| `devguide415-runtime-api-corrections#Δ6` | types/logic.md:336 | Alarm ext owns ALARM/FAULT/UNACKED bits | C32 | P1 | accepted |
| `devguide415-runtime-api-corrections#Δ7` | types/logic.md LC1 | Writable primitives: activeLevel, overrides, min on/off | C33 | P1 | accepted |
| `devguide415-runtime-api-corrections#Δ8` | types/logic-authoring.md:489-534 | Regenerate the BFacets key card | C06 | P0 | folded (WU-P0-3 + lints) |
| `devguide415-runtime-api-corrections#Δ9` | types/logic.md:399-418, types/logic-authoring.md:190-200 | Fix BFormat card; cursor recipe | C07 | P0 | accepted-corrected (step order per rcf Δ6); folded card (WU-P0-3); cursor recipe owed (C19) |
| `devguide415-runtime-api-corrections#Δ10` | types/logic.md:111-143 | Fix link rules | C08 | P0 | accepted |
| `n4-course-as-exemplar-source#Δ1` | SOURCES.md, corpus-index.md | Register the course as a P0 source | C51 | P1 | accepted |
| `n4-course-as-exemplar-source#Δ2` | BUILD-LOOP.md §0, skill/SKILL.md | Reference-first rule at Orient | C51 | P1 | accepted |
| `n4-course-as-exemplar-source#Δ3` | METHODOLOGY.md | K-rule: re-check absolute claims; one-time sweep | C51 | P1 | accepted |
| `n4-course-as-exemplar-source#Δ4` | build-verify.md, METHODOLOGY.md K15 | slotomatic before jar for region-less sources | C58 | P2 | folded (U4 C58) |
| `n4-course-authoring-files-and-libraries#Δ1` | types/third-party-libraries.md | Path 0 uberjar; library → permission table | C26 | P0 | accepted |
| `n4-course-authoring-files-and-libraries#Δ2` | templates/module-permissions.xml, types/structure.md, fixture | Canonical permissions template | C47 | P1 | accepted |
| `n4-course-authoring-files-and-libraries#Δ3` | types/structure.md:133-180, fixture palette, toolbelt/verify-module.sh | Fix palette example and format card | C11 | P0 | accepted |
| `n4-course-authoring-files-and-libraries#Δ4` | types/structure.md or types/module-wiring.md | Part gradle.kts reference; bajadoc packages | C58 | P2 | folded (U4 C58, module-wiring § 1.1); lint owed G62 |
| `n4-course-authoring-files-and-libraries#Δ5` | types/module-wiring.md §2, toolbelt/verify-module.sh | module-include.xml lifecycle; dangling types | C25 | P0 | accepted |
| `n4-course-build-deps-permissions-signing#Δ1` | types/module-wiring.md, toolbelt/build.sh | Supertype module rule | C58 | P2 | folded doc (U4 C58); build.sh hint owed G60 |
| `n4-course-build-deps-permissions-signing#Δ2` | types/structure.md, toolbelt/verify-module.sh | Permission-group roster with triggers | C28 | P0 | accepted |
| `n4-course-build-deps-permissions-signing#Δ3` | types/module-wiring.md, toolbelt/verify-module.sh | Private-API rule (com.tridium.*) | C59 | P2 | folded doc (`types/module-wiring.md` § 4.5; lint owed G65) |
| `n4-course-build-deps-permissions-signing#Δ4` | build-verify.md, toolbelt/lint-structure.sh | Signing hardening recipe | C48 | P1 | accepted |
| `n4-course-build-deps-permissions-signing#Δ5` | toolbelt/preflight.sh, build-verify.md | JDK-8 toolchain preflight | C58 | P2 | folded doc (U4 C58); preflight owed G61 |
| `n4-course-build-deps-permissions-signing#Δ6` | types/logic-authoring.md | BFacets multiLine, IDirectory, @Facet syntax | C06 | P0 | folded (WU-P0-3) |
| `n4-course-drivers-bql-ords-files#Δ1` | types/logic-authoring.md, toolbelt/lint-null-context-write.sh | BQL consumer pattern with Context | C19 | P0 | accepted |
| `n4-course-drivers-bql-ords-files#Δ2` | types/security.md §4 | BQL literal rules | C61 | P2 | folded (U5) |
| `n4-course-drivers-bql-ords-files#Δ3` | types/driver-authoring.md | NDriver section | C60 | P2 | folded (U5; lints owed G66, G67) |
| `n4-course-drivers-bql-ords-files#Δ4` | types/logic-authoring.md | ORD call rules | C18 | P0 | accepted |
| `n4-course-drivers-bql-ords-files#Δ5` | types/logic-authoring.md | File-export recipe | C62 | P2 | folded (U5; lint owed G68) |
| `n4-course-lexicon-logging-testing#Δ1` | types/structure.md, toolbelt/slot-coverage.sh | Lexicon key catalogue; Type.displayName | C10 | P0 | accepted-corrected (slot row per lux Δ1) |
| `n4-course-lexicon-logging-testing#Δ2` | types/observability.md, toolbelt/lint-lexicon-keys.sh | Runtime lexicon idiom; key-exists lint | C55 | P2 | folded (U6, `types/lexicon.md` § 3); key lint owed G70 |
| `n4-course-lexicon-logging-testing#Δ3` | types/moduleTest.md | moduleTest layout and idioms | C41 | P1 | accepted |
| `n4-course-lexicon-logging-testing#Δ4` | toolbelt/triage-console.sh | Spanish JDK level names | C57 | P2 | doc half folded (U6, `types/diagnostics.md` § 3); script fix owed G73 |
| `n4-course-lexicon-logging-testing#Δ5` | types/observability.md | Log-level operating notes | C57 | P2 | folded (U6, `types/diagnostics.md` § 2); lint owed G74 |
| `n4-course-parity-observability-horizons#Δ1` | types/logic.md | Callback thread map | C37 | P1 | accepted |
| `n4-course-parity-observability-horizons#Δ2` | types/value-types.md | Locale-safe BSimple encoding | C49 | P1 | accepted |
| `n4-course-parity-observability-horizons#Δ3` | toolbelt/kit-ticket.sh, types/observability.md | Client parity ticket | C52 | P1 | accepted |
| `n4-course-parity-observability-horizons#Δ4` | types/observability.md, toolbelt/triage-console.sh | Station diagnostics card | C57 | P2 | folded (U6, `types/diagnostics.md`); triage patterns owed G73 |
| `n4-course-parity-observability-horizons#Δ5` | types/logic-authoring.md:540, types/logic.md | History/tag commissioning items | C35 | P1 | accepted |
| `n4-course-parity-observability-horizons#Δ6` | corpus-index.md, toolbelt/kit-ticket.sh | Course horizons list | C76 | P3 | folded (U10, `types/horizons.md` § 1) |
| `n4-course-slots-components-contracts#Δ1` | types/actions.md:24-26 | Frozen action handler = type-load error | C09 | P0 | accepted |
| `n4-course-slots-components-contracts#Δ2` | types/security.md §2 | Flags are UI, not security | C36 | P1 | accepted |
| `n4-course-slots-components-contracts#Δ3` | types/logic.md | Callback isRunning() guard | C36 | P1 | accepted |
| `n4-course-slots-components-contracts#Δ4` | types/logic.md | Validators skip link writes | C36 | P1 | accepted |
| `n4-course-slots-components-contracts#Δ5` | types/logic-authoring.md or types/logic.md | Slot-API traps | C36 | P1 | accepted |
| `n4-course-slots-components-contracts#Δ6` | types/logic.md | Write/state discipline | C36 | P1 | accepted |
| `n4-course-threading-jobs-doctrine#Δ1` | types/logic.md:344, types/math-control-libraries.md | Replace the JDK-thread ban with Tridium ladder | C01 | P0 | folded (WU-P0-1) |
| `n4-course-threading-jobs-doctrine#Δ2` | toolbelt/lint-timers.sh:477-485 | Re-scope jdk-thread FAIL | C01 | P0 | folded (WU-P0 lints) |
| `n4-course-threading-jobs-doctrine#Δ3` | types/actions.md §2 | Action threading; Invocation offload | C38 | P1 | accepted |
| `n4-course-threading-jobs-doctrine#Δ4` | types/logic.md, toolbelt/lint-intrinsic-lock.sh | No intrinsic locks on B* classes | C37 | P1 | accepted |
| `n4-course-threading-jobs-doctrine#Δ5` | types/logic.md, toolbelt/lint-timers.sh | Thread naming/priority hygiene | C37 | P1 | accepted |
| `n4-course-threading-jobs-doctrine#Δ6` | types/logic-authoring.md | Correct and extend Background jobs | C03 | P0 | folded (WU-P0-1 + lints) |
| `testing-parity-btestng#Δ1` | types/moduleTest.md, BUILD-LOOP.md:162-165 | Native Windows niagaraTest route | C12 | P0 | accepted |
| `testing-parity-btestng#Δ2` | toolbelt/run-niagara-test.sh | Runner that refuses 0-test passes | C40 | P1 | accepted |
| `testing-parity-btestng#Δ3` | types/moduleTest.md, build-verify.md | Test-jar signing prerequisite | C12 | P0 | accepted |
| `testing-parity-btestng#Δ4` | types/moduleTest.md, METHODOLOGY.md | Station tests for component behaviour | C41 | P1 | accepted |
| `testing-parity-btestng#Δ5` | toolbelt/lint-structure.sh | Structural-test advisory lint | C41 | P1 | accepted |
| `testing-parity-btestng#Δ6` | types/moduleTest.md, toolbelt/scaffold-module.sh | moduleTest-include.xml template | C41 | P1 | accepted |
| `review2-signing-build-route#Δ1` | build-verify.md § Verify, toolbelt/verify-module.sh | Signer-identity check by fingerprint | C48 | P1 | accepted |
| `review2-signing-build-route#Δ2` | types/security.md § 3 | Signing-profile files are secret stores | C48 | P1 | accepted |
| `review2-signing-build-route#Δ3` | build-verify.md § Signing per deploy target | Portable signing root blocks | C48 | P1 | accepted |
| `review2-signing-build-route#Δ4` | build-verify.md § Raw gradle | Windows-native build route from WSL | C58 | P2 | folded (U4 C58) |
| `review2-signing-build-route#Δ5` | types/security.md § 2.4, types/logic-authoring.md | Worker threads keep the caller's Context | C39 | P1 | accepted |
| `review2-signing-build-route#Δ6` | BUILD-LOOP.md § 7, toolbelt/ | Computed retro delta counts | C51 | P1 | accepted |

## Appendix B — r3-report corrections → C-item

| ID | C-item |
|---|---|
| r3-report#R1 | C13 |
| r3-report#R2 | C14 |
| r3-report#R3 | C43 |
| r3-report#R4 | C71 |
| r3-report#R5 | C77 |
| r3-report#R6 | C23 |

Re-run after the 24th retro (15ad06f, alias `rsb`):

```
retro files: 24
source deltas: 174
mapped: 174
missing: 0
extra (in appendix, not in retros): 0
duplicate appendix rows: 0
undefined C-ids: none
tier mismatches: none
C-items defined: 80
source deltas per tier: P0=56 P1=69 P2=43 P3=6 
RESULT: PASS (missing=0)
```

## Appendix C — 2026-10-16 hands-on deltas → C-item

Same columns as Appendix A. `new` = the delta opens C81–C93; every other row confirms, refines or extends an existing
C-item (relation in the Summary column). Outside the scope of `kit-improvement-candidates-2026-10-15.check.sh`, which
reads only Appendix A and the 2026-10-15 retros; the no-loss result for this appendix is recorded below the table.

| Source ID | Target kit file(s) | Summary | C-item | Tier | Status |
|---|---|---|---|---|---|
| `n4-course-handson-build-signing-windows#Δ1` | toolbelt/verify-module.sh, build-verify.md, METHODOLOGY.md:67 | new: `signed` accepts alias-named `.SF` + key file (#260) | C81 | P0 | accepted |
| `n4-course-handson-build-signing-windows#Δ2` | toolbelt/verify-module.sh (:214, :223, :411), tests/ | new: pipefail/SIGPIPE false negatives in membership tests | C81 | P0 | accepted |
| `n4-course-handson-build-signing-windows#Δ3` | toolbelt/verify-module.sh (:202-212), types/third-party-libraries.md | new: bytecode check skips META-INF/versions | C81 | P0 | accepted |
| `n4-course-handson-build-signing-windows#Δ4` | toolbelt/lint-structure.sh, templates/, toolbelt/scaffold-module.sh | new: root signing block template + root-aware lint (extends bdps Δ4) | C85 | P1 | accepted |
| `n4-course-handson-build-signing-windows#Δ5` | toolbelt/build.sh:255, BUILD-LOOP.md § 4, build-verify.md § Raw gradle | new: route-independent jar-lock pattern; qualified tasks | C86 | P1 | accepted |
| `n4-course-handson-build-signing-windows#Δ6` | toolbelt/preflight.sh, build-verify.md | new: sample/course project retarget preflight | C87 | P1 | accepted |
| `n4-course-handson-ndriver#Δ1` | types/driver-authoring.md:15, § 5 | new: `void doPing()`, void `pingOk()`/`pingFail()` | C83 | P0 | accepted |
| `n4-course-handson-ndriver#Δ2` | new toolbelt/lint-driver-scaffold.sh, report-module.sh | new: wizard-stub silent-success lint | C88 | P1 | accepted |
| `n4-course-handson-ndriver#Δ3` | types/driver-authoring.md § 5 | extends C83: reachability-only ping recipe; confirms bdps Δ2 | C83 | P0 | accepted |
| `n4-course-handson-ndriver#Δ4` | types/driver-authoring.md, commissioning checklist | new: `serverEnabled` defaults to true on client drivers | C89 | P1 | accepted (override `[INFER]`) |
| `n4-course-handson-ndriver#Δ5` | types/driver-authoring.md § 2, commissioning checklist | extends C60: enum proxy facets "Use Frozen Enum in Range" | C60 | P2 | folded (U5) |
| `n4-course-handson-ndriver#Δ6` | types/driver-authoring.md, types/wb-widgets.md | new: NDriver manager + discovery recipe | C91 | P2 | accepted |
| `n4-course-handson-ndriver#Δ7` | types/driver-authoring.md, ord Δ3 lint | confirms C60: static subscription set in the course reference | C60 | P2 | folded (U5; lint owed G66) |
| `n4-course-handson-ndriver#Δ8` | toolbelt/verify-module.sh --src, types/module-wiring.md | refines C59: allow-list `com.tridium.ndriver.**` | C59 | P2 | deferred (U5; owed lint G65) |
| `n4-course-handson-services-history-threading#Δ1` | types/logic-authoring.md, lint-bql-string-concat.sh | confirms C62/C19: history → CSV recipe; BQL-concat lint refinement | C62 | P2 | folded (U5; lint owed G69) |
| `n4-course-handson-services-history-threading#Δ2` | types/actions.md § 2 | confirms C38: `Flags.ASYNC` lint stays WARN | C38 | P1 | accepted |
| `n4-course-handson-services-history-threading#Δ3` | toolbelt/lint-timers.sh | refines C01: multi-line class-header detection | C01 | P0 | folded (WU-P0 lints) |
| `n4-course-handson-services-history-threading#Δ4` | types/logic.md:344 | confirms C01: Thread/ExecutorService live under MANAGE_EXECUTION | C01 | P0 | folded (WU-P0-1) |
| `n4-course-handson-services-history-threading#Δ5` | types/third-party-libraries.md | confirms C26: `uberjar` pulls transitives; multi-release result | C26 | P0 | accepted |
| `n4-course-handson-services-history-threading#Δ6` | types/logic-authoring.md § Background jobs | confirms C03: job contract worked example | C03 | P0 | folded (WU-P0-1) |
| `n4-course-handson-slotomatic-authoring#Δ1` | METHODOLOGY.md K15, toolbelt/build.sh | new: K15 marker mechanism + pre-gradle marker check | C84 | P0 | accepted |
| `n4-course-handson-slotomatic-authoring#Δ2` | types/logic-authoring.md § New-type checklists | extends C36: `BDouble` properties get primitive accessors | C36 | P1 | folded (U11, logic-authoring § Generated accessors) |
| `n4-course-handson-slotomatic-authoring#Δ3` | types/logic-authoring.md, toolbelt/verify-module.sh --src | extends C36: slot declaration order (low before high) | C36 | P1 | folded doc (U11); lint owed G96 |
| `n4-course-handson-slotomatic-authoring#Δ4` | types/logic.md § Safety fail-modes | new: `min > max` bounds inversion rule | C90 | P1 | folded (U11, logic.md § Operator-entered bounds; remedy `[INFER]`) |
| `n4-course-handson-slotomatic-authoring#Δ5` | toolbelt/lint-agent-on-shape.sh, types/module-wiring.md | refines C25: processor never updates a stale `<type>` line | C25 | P0 | accepted |
| `n4-course-handson-slotomatic-authoring#Δ6` | toolbelt/verify-module.sh `check_palette` | new: palette count misses non-first `n=` | C82 | P0 | accepted |
| `n4-course-handson-workflow#Δ1` | BUILD-LOOP.md § 0, new toolbelt/diff-vs-reference.sh | confirms C51: reference-first + copy decision helper | C51 | P1 | folded doc (U11, METHODOLOGY § Hands-on work); helper owed G93 |
| `n4-course-handson-workflow#Δ2` | BUILD-LOOP.md § 0.a | new: orient reports the newest working-tree file | C92 | P2 | folded doc (U11, METHODOLOGY § Hands-on work); owed G94, pointer P2 |
| `n4-course-handson-workflow#Δ3` | build-verify.md, types/moduleTest.md | refines C12: run `gradlew.bat` in place on `/mnt/<drive>` | C12 | P0 | folded doc (U4 C58); run-niagara-test.sh skip waits on that script |
| `n4-course-handson-workflow#Δ4` | ORCHESTRATION.md § 3, toolbelt/preflight.sh | new: shared-host heavy-job rule | C93 | P2 | folded doc (U11, METHODOLOGY § Hands-on work); owed G95, pointer P3 |

No-loss result for Appendix C (2026-10-22, T9a; ad-hoc comparison of the `| Δn |` rows of the five 2026-10-16 retros
with the rows above, and of every C-id above with its `### Cnn` heading and tier):

```
hands-on retro files: 5
source deltas: 30
mapped: 30
missing: 0
extra: 0
duplicate rows: 0
undefined C-ids / tier mismatches: none
new C-items (C81-C93) defined: 13
C-items defined: 93
source deltas per tier: P0=13 P1=10 P2=7 
```

The comparison has teeth: deleting the `n4-course-handson-workflow#Δ4` row gives `missing: 1`; changing the C93 row's tier
to P1 gives `C93(P1≠P2)` (each mutation run once, then restored).
