<!-- review-status: pending -->
# 2026-09-21 · kit · secure-authoring-isoperational-gate

**Session**: niagara-research security-enforcement-seams focus (corpus B1143–B1156) closed the full N4 runtime enforcement chain, then pivoted to `secure-module-authoring` (B1157). That block distilled 10 enforcement fissures into 5 candidate kit deltas (SA-1..SA-5). This retro is the SMA3 step: verify the 5 candidates against the SHIPPED kit before proposing. Result: 4 of 5 are already covered; 1 genuine gap survives.
**Delta count**: 1

## What happened
The research chain proved the framework's runtime license enforcement is **advisory**, not blocking: `BAbstractService` faults the status on an unlicensed/expired feature but does NOT stop the service, and the engine caller (`ServiceManager.startService`) runs the subclass `serviceStarted()` unconditionally after the framework's own `checkLicense()` — with no guard between them. Well-behaved Tridium services therefore gate their ongoing work on `if (isOperational())`; a service that omits the guard runs unlicensed while displaying a fault.

The kit's `types/security.md §5 "Licensing your own module"` teaches the right STARTUP guard (call `feature.check()` in `serviceStarted()` so it throws and the service start aborts). But it stops there: it does not tell the author to gate ONGOING callbacks (`changed()`, timers, servlet handlers) on `isOperational()`. A `serviceStarted()` throw guards only startup; later callbacks still fire unlicensed. That is the one genuine gap.

The other four candidates were verified as already covered — recorded below so they are not re-proposed.

## Evidence
- Framework license enforcement is advisory: `checkLicense()` sets `fatalFault` + SEVERE log but only `updateStatus()` runs; the base class never stops the service. `[ev: corpus B1143 §1143.2-3 — BAbstractService.java:332-393,519-523]`
- Engine caller runs work regardless: `ServiceManager.startService` does `fw(15)`=checkLicense THEN `serviceStarted()` with no fatal-fault guard between (`:297-298`); wrapped in `catch(Throwable)` that logs and continues station boot. `[ev: corpus B1145 — ServiceManager.java:291-322]`
- Real services self-enforce by convention: `BAlarmService`/`BSearchService`/`BHierarchyService`/`BTagDictionaryService`/`BBatchJobService`/`BBoxService`/`BCloudConnector` all gate work on `isOperational()` / `isFatalFault()`; nothing forces it. `[ev: corpus B1146 — BAlarmService.java:289,397,448,578 et al.]`
- Kit gap: `types/security.md §5` shows the `serviceStarted()` `feature.check()` throw but no `isOperational()` ongoing-gate rule. `[ev: kit types/security.md:230-254]`

### Candidates verified ALREADY COVERED (not proposed — SMA3 dedup)
| Candidate | Already covered by | Note |
|---|---|---|
| SA-2 cancel `Clock` ticket in `stopped()` | `toolbelt/lint-timers.sh` check `timer-ticket` (+ `discarded-ticket`, `companion-flag`) | B1154 (BWbEdeService, a shipped `-wb` leak) is a live confirmation of this lint's value — no new lint needed |
| SA-3 `unsubscribe` in `stopped()` what you `subscribe` in `started()` | `toolbelt/lint-subscribe-without-unsubscribe.sh` | exact match; drop |
| SA-4 signing discipline (`skipModuleValidation`, moduleVerificationMode, trusted alias) | `types/distribution.md §Hard signing gate` + `build-verify.md §Signing per deploy target` | distribution.md:86,109-117 already covers moduleVerificationMode=medium, the permission-group hard gate, and skipModuleValidation semantics |
| SA-5 server-side servlet RBAC on the write path + `post()` drops Context | `types/security.md §2.1` (BPermissions bit table) + `§2.4` (post() drops RBAC context) | already thorough; drop |

## Proposed kit deltas (propose-never-apply)
| Δ | Delta | Target file / § | Token |
|---|---|---|---|
| Δ1 (SA-1) | Augment `§5 Licensing your own module` with the ONGOING-work rule: the framework's license fault is ADVISORY — it faults the status but does not stop the service, and `serviceStarted()`'s `feature.check()` throw guards only STARTUP. So a licensed service must also gate its ongoing callbacks (`changed()`, timers, servlet write handlers) on `if (isOperational())` (= `!isFatalFault() && !isDisabled() && !isFault()`), the way `BAlarmService`/`BSearchService` do. Add a one-line note that relying on the framework to withhold start is wrong. Optional follow-on: a lint candidate `lint-license-isoperational-gate` (a class with non-null `getLicenseFeature()` whose callbacks act without an `isOperational()`/`isFault()` guard) — propose only after checking it does not overlap `lint-status-parity`/`lint-silent-protection`. | `types/security.md` §5 | `[ev: corpus B1143/B1145/B1146]` |

## Lessons
- The SMA3 verify-before-propose step earned its keep: 4 of 5 candidates were already shipped (2 lints + 2 doc sections). Proposing them would have been noise against the kit's own anti-duplication gate. Always diff candidates against `toolbelt/lint-*.sh` + `types/*.md` before writing the delta table.
- The kit's licensing guidance guarded STARTUP but not ONGOING work — a subtle half-rule. The research (B1143/B1146) showed the framework fault is advisory, which is the missing "why" behind the `isOperational()` gate.
- A shipped `-wb` service (`BWbEdeService`, B1154) trips the existing `timer-ticket` lint — evidence the lint targets a real, in-the-wild pattern, not a hypothetical.

---
**Status**: PENDING — INDEX row appended.
