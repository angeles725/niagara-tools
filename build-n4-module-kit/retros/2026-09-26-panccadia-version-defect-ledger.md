<!-- review-status: pending -->
# 2026-09-26 · kit · panccadia-version-defect-ledger

**Session**: `Cliente/panccadia-leon`, PANCCADIA León field support, git history
2026-09-23 (repo git-init baseline `116c49a`) through 2026-09-26 (`0956936` / working
tree `feat/coldroom-compressor-call-valve-open`). Reconstructed from module commit
history, ODD feature docs (`odd/tasks/*.md`), and 12 sibling kit retros covering the
same client (2026-09-02 through 2026-09-26). Deltas below are proposed only where no
sibling retro already covers the gap; every reused finding is cross-referenced by
retro file + Δ, not restated.

**Delta count**: 4

## What happened
Across four days this client shipped 27 distinct version states over three modules
(CompPan, ColdRoomPan, DashboardPan) through one production outage (a full station
load failure), one same-day shipped-then-reverted redesign, one packaged-but-never-
installed regression, three native-RDD correction rounds, and at least 21 individually
citable defects — yet none of that history lives in one place. It is scattered across
eight client feature docs and twelve kit retros, each covering one session's slice.
Nothing today lets a new session ask, in one read, "has this failure class hit this
client before, and did we already fix the kit-side gap?" — the exact question this
session had to answer by hand, retro by retro, to avoid re-proposing deltas the
sibling retros already cover (see the drop notes throughout). The version timeline
below is the ledger core; the root-cause taxonomy after it groups the 21 defects into
8 classes and shows which sibling retro/Δ already targets each class; the proposed
deltas close the one gap none of the 12 sibling retros propose: a durable, per-client,
continuously updated ledger artifact, not a one-time reconstruction.

## Evidence
- Module repo git history (version bumps + fixes), reconstructed via
  `git log --all -p -- '*/build.gradle.kts' | grep -E '^(commit|[-+].*defaultModuleVersion)'`
  and `git log --all --format="%h %ad %s" --date=format:"%Y-%m-%d %H:%M"`, all commits
  cited by hash below. `[ev: Cliente/panccadia-leon git log 116c49a..0956936]`
- Current uncommitted state (`git status --short`, 2026-09-26): three modified
  `ColdRoomPan-rt` sources + two new test files on branch
  `feat/coldroom-compressor-call-valve-open`; `Paccadia/build.gradle.kts` still reads
  `2.2.2` (2.3.0 not yet bumped). `[ev: Cliente/panccadia-leon git status 2026-09-26]`
- Feature docs read in full: `odd/tasks/comppan-fase2-amps-alarms.md`,
  `restart-seq-comp-lockout-hours.md`, `continuous-fan-post-defrost-delay.md`,
  `comppan-auto-lock-indicator.md`, `comppan-autolock-keeps-mode.md`,
  `coldroom-compressor-call-valve-open.md`, `defrost-sequencing-hmi-reload.md`
  (session-closed). `[ev: Cliente/panccadia-leon odd/tasks/*.md]`
- Sibling kit retros read in full (all cited by file name inline in the tables below):
  `2026-09-23-panccadia-defrost-sequencing-hmi-reload-deltas.md`,
  `2026-09-24-comppan-fase2-amps-alarms.md`,
  `2026-09-24-panccadia-restart-seq-comp-lockout-hours.md`,
  `2026-09-25-panccadia-unknown-unit-outage.md`,
  `2026-09-25-panccadia-commissioning-lessons.md`,
  `2026-09-25-continuous-fan-post-defrost-delay.md`,
  `2026-09-26-comppan-auto-lock-indicator.md`,
  `2026-09-26-roll-forward-recovery.md`,
  `2026-09-26-change-tier-time-budgets.md`,
  `2026-09-26-behavior-decisions-ask-dont-assume.md`.
- Research-side confirmation of the same defects from an independent angle (framework
  root-cause, not commissioning): `niagara-research/docs/pending-research/
  2026-09-25-niagara-module-internals.md` (current) and its prior committed version
  (`git show HEAD:...`) — same unit-id, LinkCheck/READONLY, static-initializer-cascade,
  slotomatic-comment, and rotation-liveness items, framed as open research questions
  rather than kit deltas. `[ev: niagara-research docs/pending-research/2026-09-25-
  niagara-module-internals.md]`
- No existing kit or client artifact aggregates version + defect + root-cause-class
  across sessions for one client: `BUILD-STATE.md` tracks only the current
  deployed-baseline per module (one row, overwritten each update, per
  `2026-09-23-panccadia-defrost-sequencing-hmi-reload-deltas.md` Δ3); no
  `odd/VERSION-LEDGER.md` or equivalent exists in `Cliente/panccadia-leon`.
  `[ev: Cliente/panccadia-leon odd/ directory listing, 2026-09-26]`

## Version timeline

Legend for Status: **deployed** = confirmed live on the JACE by explicit session
evidence; **built** = compiled/verified/packaged, confirmed not installed; **reverted**
= shipped in source then reverted before any deploy; **in progress** = uncommitted
working-tree change, not yet bumped; **?** = no explicit deploy/non-deploy statement
found in the cited sources — marked, not guessed.

### CompPan (Compresores)

| Version | Date | Status | What changed | Defect found (symptom) | Root cause | Class | Resolved by | Prevention (delta/rule) |
|---|---|---|---|---|---|---|---|---|
| 2.1.0 (baseline) | 2026-09-23 | built, not installed | Debounce fix present in source (built 2026-09-21) | JACE still ran a same-numbered 2.1.0 without the debounce (built 2026-09-07); Software Manager reported "Up to Date" and refused install | No version bump on a behavior-only (non-schema) change | Process / version-deploy discipline | 2.1.1 | `2026-09-23-...-deltas.md` Δ2 (bump on every shipped-bytes change) |
| 2.1.1 | 2026-09-23 21:40 `66f62a8` | deployed | Version-only bump to force redeploy of the existing debounce | — (this commit is the fix for the row above) | — | — | — | — |
| 2.2.0 | 2026-09-24 11:54 `642cc90` (+ `53cdddc` 11:42) | built, not installed | Compressor-hours JSON backup; optional auto-off after sustained proof-of-run fault | 3 native RDD review rounds before ship; CRITICAL `R3-stale-bak-blocks-rename-away-on-windows` (a stale `.bak` blocks `File.renameTo` on Windows) | `File.renameTo` silently overwrites on POSIX, refuses on Windows; naive delete-then-move leaves a data-loss window | Persistence / restart / release-point gate | `1d4cb62` correction, same version | `2026-09-24-panccadia-restart-seq-comp-lockout-hours.md` Δ6 |
| 2.2.0 (same version, review-driven) | 2026-09-24 12:19-12:52 `010f0a8`,`d96ae44`,`9251f21`(ColdRoomPan) | folded pre-ship | HAND exclusion, reliable hours backup, strict parse; observe-not-infer release predicate | `R3-release-predicate-infers-fired-from-config`: a release predicate inferred "already fired" from configured durations instead of an observed flag | Reconstructing "did X already happen" from config/timing instead of setting a flag at the moment it happens | Persistence / restart / release-point gate | same version | `2026-09-24-panccadia-restart-seq-comp-lockout-hours.md` Δ9 |
| 2.3.0 | 2026-09-24 14:48 `e5d568c` | ? (no explicit deploy statement found) | Keep a minimum number of compressors on, HAND-aware, safety-overridden | none recorded against this specific bump | — | — | — | — |
| 2.4.0 | 2026-09-24 21:48 `53de4fb` (+`a4b8209` prove-run defaults) | built, not installed (packaged v3, "nothing deployed") | Over-amperage alarm (alarm-only); prove-run defaults tightened 5A/5s -> 2A/10s | `904469d` (22:35, same version): run-hours now counted only while amps prove running, not while commanded | Changed the predicate driving a wear-based rotation metric without a rotation-consequence check: a never-starting unit now accrues zero hours and is pinned first by least-hours rotation | Rotation-liveness / wear-metric consequence | not resolved this ledger period — confirmed live twice (see below) | `2026-09-24-comppan-fase2-amps-alarms.md` Δ6 |
| 2.5.0 | 2026-09-25 01:06 `b01b1fd` | **reverted** same session, never deployed | "Fase 2 = rooms-base +/-1 suction-pressure trim" | Client phrase "setpoint de baja" meant the LP cutout (pump-down trigger), not a control setpoint — wrong design, caught only after build+test+review | Ambiguous field term coded without restating the sequence to the field technician first | Requirements / behavior assumed, not asked | `f40db14` revert -> `9c80d53` redo | `2026-09-25-panccadia-commissioning-lessons.md` Δ2 |
| 2.4.0 (revert) | 2026-09-25 01:19 `f40db14` | n/a (git revert, pre-deploy) | Revert of the row above | — | — | — | — | — |
| 2.5.0 (redo) | 2026-09-25 01:39 `9c80d53` | deployed 2026-09-25 (live JACE, alongside DashboardPan 2.7.0/2.7.1) | Fase 2 pump-down on stop; drops the permanent one-compressor minimum | Native review (`review-0130c696f9e8cc6e`) found 3 design defects AFTER implementation: no `suctionValid` gate on entry, `minOn` bypassed for ALL units instead of only the last, `pumpDownMaxTime<=0` allowed | Design-shard step (BUILD-SKILL 1c) skipped for speed before writing a new state machine | Skipped design step for a new state machine | recorded as review follow-up P2, not yet shipped | `2026-09-25-panccadia-commissioning-lessons.md` Δ7 |
| 2.6.0 | 2026-09-26 01:54 `9c2bd0a` | deployed 2026-09-26 (station did NOT crash, user-confirmed) | Latched `condenserNAutoLocked` indicator per compressor | Native review (1 lens, reliability, approved): `R3-w35d-vacuous-manual-apagar` (negative test ran with the feature disabled) and `R3-unlinked-autolocked-reads-false` (unlinked indicator defaults to `false` = "not locked") | (a) negative-criterion test run under a disabled flag is vacuous; (b) a link-in indicator's unlinked default reads as healthy | (a) Test hygiene (b) Fail-open UX / status-trust discipline | non-blocking, not yet fixed | `2026-09-26-comppan-auto-lock-indicator.md` Δ2, Δ3 |
| 2.6.0 (live, post-deploy) | 2026-09-26 ~02:44 | live defect, not a version yet | — | Auto-lockout wrote `condenserNMode`; that slot is a link TARGET fed by dashboard `comp3Mode` which propagates only ON CHANGE — dashboard showed AUTO, pressing AUTO sent nothing, a station restart re-armed the burned compressor | Automatic action reused the manual-OFF write path without asking what a downstream link consumer sees afterward | Requirements / behavior assumed, not asked (+ framework rule unknown: link propagates only on change) | 2.6.1 | `2026-09-26-behavior-decisions-ask-dont-assume.md` item 2, Δ1 |
| 2.6.1 | 2026-09-26 03:20 `0956936` | **built, T1 only** — T2 (build/verify/RDD/handoff) explicitly not run this session; pending review, not deployed | Auto-lockout no longer writes `condenserNMode`; latch clears from HAND/OFF or a `faultReset` edge; mode stays AUTO | none new (this is the fix for the row above) | — | — | — | — |

### DashboardPan

| Version | Date | Status | What changed | Defect found (symptom) | Root cause | Class | Resolved by | Prevention (delta/rule) |
|---|---|---|---|---|---|---|---|---|
| 2.4.2 (baseline) | 2026-09-23 | live, defective | — | HMI goes blank after a station restart: SPA `poll()` only marks data stale on error, never reloads/re-authenticates | No kiosk-recovery watchdog in the SPA polling loop | Framework rule unknown (kiosk pattern) | 2.4.3 | `2026-09-23-...-deltas.md` Δ1 (`lint-spa-poll-no-recovery`) |
| 2.4.3 | 2026-09-23 21:40 `f0ddfdc` | deployed 2026-09-23 (session closed) | Station-restart watchdog added | — (fix for the row above) | — | — | — | — |
| 2.5.0 | 2026-09-24 14:33 `43a055f` | ? | Cuarto 3 drip time and defrost-gap dashboard fields | none recorded against this specific bump | — | — | — | — |
| 2.6.0 | 2026-09-24 22:19 `ba39417` | built, not installed | Compressor alarm banner, Fase 2 config tab, phase display | `3c94e01` (22:47, same day): reader helper defaulted `st = p.st \|\| "ok"`, so a MISSING status read as "ok"; a `BStatusBoolean` default `false` made an unlinked/uncommissioned phase flag show "Fase 2 active" | A generated status/JSON UI point trusted without gating on BOTH status and type; a facade default not set to the safe fallback | Fail-open UX / status-trust discipline | 2.6.1 | `2026-09-24-comppan-fase2-amps-alarms.md` Δ5 |
| 2.6.1 | 2026-09-24 22:47 `3c94e01` | packaged v3, not installed | Fixes the row above: `st==="ok" && typeof v==="boolean"` gate; `pressureFallback` default flipped `false`->`true` | — | — | — | — | — |
| 2.6.2 | 2026-09-24 23:10 `8ac7c97` | built, "nothing deployed to the JACE" (per retro) | Review follow-ups for the fase2-amps-alarms feature | none recorded against this specific bump | — | — | — | — |
| 2.7.0 | 2026-09-25 01:52 `49c79ef` | **deployed 2026-09-25 02:15 CST — caused a full station load failure** | Pump-down config fields + phase line for the new Fase 2 | `pumpDownCutout` facet used `BUnit.getUnit("pound_force_per_square_inch")`, verified only against the writer's local Honeywell 4.14 overlay unit database, absent on the target's stock Tridium 4.15.3.28; the resulting `UnitException` aborted `BCompressorPanel.<clinit>`, cascading to `BDashboardService` and the whole station | A facet/default expression runs in a type's static initializer; an exception there is a whole-station failure, not a per-slot one; no check in the kit validates a unit id against the DEPLOY TARGET's own resource set | Framework rule unknown (static-init cascade + target/distribution mismatch) | 2.7.1 | `2026-09-25-panccadia-unknown-unit-outage.md` Δ1-Δ4; `2026-09-25-panccadia-commissioning-lessons.md` Δ10 |
| 2.7.1 | 2026-09-25 02:20 `ae98f79` | deployed 2026-09-25, station recovered (roll-forward, confirmed by jar-strings extraction) | Drops the unit facet from `pumpDownCutout` | — (fix for the row above) | — | — | — | — |
| 2.8.0 | 2026-09-26 01:54 `2456d41` | deployed 2026-09-26 alongside CompPan 2.6.0, station did NOT crash | `compNAutoLocked` badge on the compressor card | Shares CompPan 2.6.0's review findings: `R3-unlinked-autolocked-reads-false` applies to this slot's link-in default | Fail-open UX / status-trust discipline (unlinked indicator) | non-blocking, not yet fixed | `2026-09-26-comppan-auto-lock-indicator.md` Δ3 |
| 2.8.1 | 2026-09-26 02:31 `f12f0fe` | **built, not yet deployed** (per session brief) | Opens the second login from Configuración; renders on Chrome 83 HMI | (a) Save buttons were disabled with no session, and a disabled button fires no click, so Configuración never offered the login; (b) this HMI runs Chrome 83.0.4103.116 — CSS `inset` and flex `gap` unsupported, broke the login overlay | (a) UX assumed "no session => disable Save" without checking that a disabled control cannot re-open the gate it guards; (b) second live occurrence of a legacy-Chromium HMI (first: Exor UN78, Config OS 2.0.560, 2026-09-25) | (a) Requirements/behavior assumed, not asked (b) Framework/target-environment assumption (legacy Chromium), 2nd occurrence | not yet verified live | `2026-09-26-behavior-decisions-ask-dont-assume.md` item 5, Δ1/Δ6; `2026-09-25-panccadia-commissioning-lessons.md` Δ6 (reinforced, not re-proposed) |

### ColdRoomPan (Paccadia)

| Version | Date | Status | What changed | Defect found (symptom) | Root cause | Class | Resolved by | Prevention (delta/rule) |
|---|---|---|---|---|---|---|---|---|
| 2.1.2 (baseline) | 2026-09-23 | live, defective | — | Air-defrost evaporator fan HOA could never be forced OFF during defrost (`BEvaporatorUnit.setBool` comment: "their HOA never overrides defrost"); field workaround via kitControl Or/Equal bypass on the wire sheet | No documented HOA/automatic precedence contract per output | Requirements / behavior assumed, not asked | 2.1.3 | `2026-09-23-...-deltas.md` Δ4 |
| 2.1.3 | 2026-09-23 22:07 `286b77a` | deployed 2026-09-23 (session closed) | Room defrost sequencing, activation/end cycle anchors, manual HOA precedence | — (fix for the row above, plus cycle-anchor correction — duration-from-actuation, interval-from-end) | — | — | — | `2026-09-23-...-deltas.md` Δ6 |
| 2.1.4 | 2026-09-24 11:33 `3f5d15b` | folded into 2.2.0, not independently deployed | Restart sequence: valve first, fan after 10s | (see the observe-not-infer finding under CompPan 2.2.0 — same review round, `9251f21` companion fix in this module) | — | Persistence / restart / release-point gate | folded same version | `2026-09-24-panccadia-restart-seq-comp-lockout-hours.md` Δ9 |
| 2.2.0 | 2026-09-24 14:17 `96b846d` (+`006227b`) | **deployed 2026-09-24** (live JACE, confirmed by `niagara-research` pending-research doc header) | Restart gap uses `startDelay`; post-defrost drip phase; gap counts from drip end; leaves defrost state when drip begins | none recorded against this specific bump beyond the review round already logged under 2.1.4 | — | — | — | — |
| 2.2.1 | 2026-09-25 12:02 `4174bf6` (fix `35f84ca`) | **packaged (v6), never installed** — caught before deploy | Fan/valve re-energized in the SAME scan right after defrost/drip exit on `fanRunMode=continuous` units | `exitDefrost()`/`endDrip()` never opened the restart-sequencing window (`beginRestartSequencing()`) before re-applying outputs — a gate fixed for ONE release point (restart) was not extended to another (defrost exit) | Persistence / restart / release-point gate | 2.2.2 | `2026-09-25-continuous-fan-post-defrost-delay.md` Δ2 |
| 2.2.2 | 2026-09-25 12:29 `f0cf0c5` (fix `027e085`) | ? (verify/build evidence only; no explicit live-deploy statement) | Same-day regression fix: `postDefrostFanHold(airDefrost, fromDrip) = !airDefrost \|\| fromDrip` — only electric-defrost (Cuarto 3) units get the hold | 2.2.1 gated the hold UNCONDITIONALLY, so an air-defrost unit (fan never stops) got stopped-and-restarted for no reason on Cuartos 1/2/4 | A release-point gate must consider the OUTPUT'S STATE BEFORE release, not just "is this a release-point event" | Persistence / restart / release-point gate | this version | `2026-09-25-continuous-fan-post-defrost-delay.md` Δ3 |
| 2.3.0 | in progress (uncommitted, 2026-09-26) | not built, not tested | New `compressorCall` = OR(unit `valveOut`) per room, replacing `cooling`=OR(`runCmd`) as the CompPan demand source; 4 relinks planned | Live 2026-09-26 03:00-03:27: three units mid-defrost had `runCmd=true, valveOut=false`; only one had an open valve; demand stayed 3, suction fell 26.2->23.7 psig; compressor 2 (mechanical 35 psi suction cut-in) held off and was flagged `condenser2Fault=true` on a healthy unit | "Room calls for compressor" was coded as OR(any evaporator run order) instead of OR(any open valve) — an unstated field-behavior assumption | Requirements / behavior assumed, not asked | in progress this session | `2026-09-26-behavior-decisions-ask-dont-assume.md` item 1/3, Δ1, Δ2 |

## Root-cause classes (taxonomy, with counts)

| Class | Instances (row refs) | Already covered by (retro + Δ) |
|---|---|---|
| 1. Framework rule unknown (a stock Niagara/baja mechanism not known in advance) | 4 — DashboardPan 2.7.0 unit-id/static-init cascade; BRoomPanel READONLY link-target rejection (found live, same era, not tied to one version bump); CompPan 2.6.0 mode-write/"propagates only on change"; slotomatic comment-truncation (recurred 4x across T2b/T2c/T4/T5-T6) | `2026-09-25-panccadia-unknown-unit-outage.md` Δ1-Δ4; `2026-09-25-panccadia-commissioning-lessons.md` Δ1; `2026-09-26-behavior-decisions-ask-dont-assume.md` item 2; `2026-09-24-comppan-fase2-amps-alarms.md` Δ7 |
| 2. Requirements/behavior assumed, not asked with the field | 5 — CompPan 2.5.0 Fase-2 redefinition; CompPan 2.6.1 mode-write trigger; ColdRoomPan 2.3.0 demand=OR(runCmd); ColdRoomPan 2.1.2 HOA/air-defrost precedence; DashboardPan 2.8.1 disabled-Save-button UX | `2026-09-25-panccadia-commissioning-lessons.md` Δ2; `2026-09-26-behavior-decisions-ask-dont-assume.md` (whole retro, Δ1-Δ6); `2026-09-23-...-deltas.md` Δ4; `2026-09-26-comppan-auto-lock-indicator.md` Δ1 |
| 3. Persistence / restart / release-point gate design gaps | 5 — CompPan 2.2.0 Windows rename-race; CompPan/ColdRoomPan observe-vs-infer release predicate; DashboardPan hours-reset-on-first-backup-boot (documented, not version-pinned); ColdRoomPan 2.2.1 restart-only gate not extended to defrost exit; ColdRoomPan 2.2.2 gate-ignores-prior-output-state regression | `2026-09-24-panccadia-restart-seq-comp-lockout-hours.md` Δ5, Δ6, Δ9; `2026-09-25-panccadia-commissioning-lessons.md` Δ5; `2026-09-25-continuous-fan-post-defrost-delay.md` Δ2, Δ3 |
| 4. Rotation-liveness / wear-metric consequence | 1 class, 2 live confirmations — CompPan 2.4.0 hours-counting-predicate change pinned a burned compressor first in rotation; confirmed commanding it live twice (2026-09-24 ~00:xx and 2026-09-25 ~02:32) | `2026-09-24-comppan-fase2-amps-alarms.md` Δ6; `2026-09-25-panccadia-commissioning-lessons.md` Δ13 |
| 5. Fail-open UX / status-trust discipline | 2 — DashboardPan 2.6.0 `st \|\| "ok"` default; CompPan/DashboardPan 2.6.0/2.8.0 unlinked-indicator-reads-false | `2026-09-24-comppan-fase2-amps-alarms.md` Δ5; `2026-09-26-comppan-auto-lock-indicator.md` Δ3 |
| 6. Process / version-deploy discipline (built ≠ deployed; rollback assumption) | 2 — CompPan 2.1.0 no-bump-no-install; the "rollback = old jar" assumption contradicted by same-named-jar reality | `2026-09-23-...-deltas.md` Δ2, Δ3; `2026-09-26-roll-forward-recovery.md` (whole retro, Δ1-Δ4) |
| 7. Skipped design step for a new control state machine | 1 — CompPan 2.5.0 pump-down, 3 design defects found only by review after implementation | `2026-09-25-panccadia-commissioning-lessons.md` Δ7 |
| 8. Test hygiene (a negative-criterion test that is vacuous under a disabled flag) | 1 — CompPan 2.6.0 `w35d` | `2026-09-26-comppan-auto-lock-indicator.md` Δ2 |

Every class already has at least one sibling-retro delta targeting it. No new
prevention delta is proposed for the classes themselves in this retro — see the "What
the retro must contain" framing: the gap this retro closes is that none of those
deltas, once shipped, would have made classes 2 ("has this field behavior already
burned this client?"), 4 (repeated rotation-liveness hit), or 5 (repeated fail-open
hit twice with the SAME shape) visible to a session working the SAME CLIENT days
later, because nothing aggregates them across sessions per client.

## Proposed kit deltas (propose-never-apply)

| Δ | Delta | Target file / § | Token |
|---|---|---|---|
| Δ1 | **New living per-client artifact `odd/VERSION-LEDGER.md`**, one row per shipped/built version per module, columns: `Module, Version, Date, Status, What changed, Defect found, Root cause, Class, Resolved by, Prevention (retro+Δ)` — the same shape as this retro's version-timeline tables. Updated at every deploy/build handoff (the same moment `BUILD-STATE.md`'s per-module deployed-baseline is updated, per `2026-09-23-panccadia-defrost-sequencing-hmi-reload-deltas.md` Δ3) — not reconstructed after the fact from git log, as this retro had to do. Close-gate rule: a feature doc cannot move to `## Status: closed`/archived until it has appended its version row(s) to the client's ledger; `BUILD-LOOP.md`'s existing HARD close gate (§7) is the natural place to add this check, since it already gates on the retro. Scaffold the file from this retro's table structure so the first population is not a blank template. | `BUILD-LOOP.md` § `7. Retro + close (HARD close gate — not optional)` (new close-gate bullet) + client scaffold `odd/VERSION-LEDGER.md` (new, seeded from this retro) | `[ev: Cliente/panccadia-leon odd/ directory listing, 2026-09-26 — no ledger file exists]`; `[ev: 2026-09-23-panccadia-defrost-sequencing-hmi-reload-deltas.md Δ3]` |
| Δ2 | **Pre-flight step: read the client's `VERSION-LEDGER.md` (once Δ1 ships) before the first write of any new session on that client**, specifically checking the root-cause-class column against the new work's own likely classes (a new automatic action → check class 2/4/5 rows; a new persisted slot → check class 3; a new facet/unit → check class 1). This session had to do exactly this check by hand, reading 12 sibling retros in full, to confirm each candidate delta below was not already proposed — the commissioning-lessons retro's own Δ14 lesson ("verify a candidate against CURRENT kit source, not just the session's own friction log") extends here to "verify against the CLIENT's own history, not just this session's". Wire it as a named step in `BUILD-LOOP.md` §0 Orient, alongside the existing coverage-gate check (`tools/check-coverage.py`) this kit already runs before opening a research focus. | `BUILD-LOOP.md` § `0. Orient (before touching anything)` (new bullet, client-ledger read) | `[ev: this retro's own "What happened" — reconstructed by reading 12 retros in full because no ledger existed]`; `[ev: 2026-09-25-panccadia-commissioning-lessons.md Δ14 lesson]` |
| Δ3 | **Flag a REPEATED root-cause class on the same client as its own escalation signal, distinct from a first occurrence.** Two classes above already recurred with the SAME shape after their first sibling-retro delta was proposed but before it shipped: class 4 (rotation-liveness) hit live TWICE (2026-09-24 and 2026-09-25) before `2026-09-24-comppan-fase2-amps-alarms.md` Δ6's rotation-consequence test existed; class 1's legacy-Chromium HMI incompatibility hit TWICE (Exor UN78 2026-09-25, this HMI's Chrome 83 2026-09-26) before `2026-09-25-panccadia-commissioning-lessons.md` Δ6's `rc-scan.sh` check existed. Neither recurrence was itself flagged as elevated risk — each was treated as a fresh, separately-evidenced finding. Once Δ1's ledger exists, a session opening a new feature on a class with an unshipped prevention delta AND a prior live recurrence should treat that delta as PRIORITIZED (apply it now, ahead of its normal promotion queue), not merely re-cited as evidence. Add this as a one-line rule next to the ledger's close-gate check (Δ1). | `BUILD-LOOP.md` § `7. Retro + close` (new bullet, cross-referencing Δ1's close gate) | `[ev: 2026-09-24-comppan-fase2-amps-alarms.md Δ6 + 2026-09-25-panccadia-commissioning-lessons.md Δ13 — same class, 2 live hits, delta still unshipped]`; `[ev: 2026-09-25-panccadia-commissioning-lessons.md Δ6 + this retro's DashboardPan 2.8.1 row — same class, 2nd HMI, delta still unshipped]` |
| Δ4 | **Add an "Open items" appendix to the client ledger**, aggregating, per client, what today is scattered per-feature-doc and ages out when a doc archives: values owed by the field still at a disabled/zero default (`2026-09-25-panccadia-commissioning-lessons.md` Δ11's table), pending manual commissioning links not yet created in the station (this retro's CompPan 2.6.0/DashboardPan 2.8.0 row: 3 `condenserNAutoLocked` links still pending), and assumptions/behavior-decisions still open (`2026-09-26-behavior-decisions-ask-dont-assume.md` Δ4's per-doc register). Those three deltas each solve the WITHIN-ONE-FEATURE-DOC version of this problem; none aggregates ACROSS features for one client, so a value "owed" by session N is invisible to session N+5 unless that session happens to re-read session N's archived doc. Cross-reference, do not restate, the three source deltas. | client scaffold `odd/VERSION-LEDGER.md` § `Open items` (new section, sourced from the three cited deltas) | `[ev: 2026-09-25-panccadia-commissioning-lessons.md Δ11]`; `[ev: 2026-09-26-behavior-decisions-ask-dont-assume.md Δ4]`; `[ev: odd/tasks/comppan-auto-lock-indicator.md "Manual link the operator must create" — still pending 2026-09-26]` |

## Lessons
- A version timeline and a defect log are the same artifact read two ways — building
  one from git history alone (this retro) took reading 71 commits and 12 retros by
  hand; a client that had kept `odd/VERSION-LEDGER.md` from day one would have needed
  none of that reconstruction.
- The same root-cause class hitting a client twice before its sibling-retro delta ever
  ships (rotation-liveness, legacy-Chromium HMI) is a distinct, worse signal than a
  first occurrence — nothing in this kit currently treats a second live hit as reason
  to prioritize an already-proposed, still-unshipped delta.
- "Verify a candidate against current kit source" (2026-09-25 Δ14) and "verify a
  candidate against this client's own history" are two different checks; this session
  needed both, and only the first is written down anywhere in the kit.
- A value or link "owed by the field" or an "assumption still open" belongs to the
  CLIENT, not to the feature doc that first noticed it — three sibling retros each
  solved the per-doc version of this problem; none solved the cross-session, per-client
  version, which is exactly what ages out silently when a feature doc archives.
- Building this ledger surfaced no NEW defect and no NEW root-cause class — every one
  of the 21 instances traced to a sibling retro's own evidence. The gap this retro
  closes is purely structural: nowhere to see all 21 in one read, ordered by version,
  before starting the 22nd.

---
**Status**: PENDING — INDEX row appended: `| 2026-09-26-panccadia-version-defect-ledger.md | kit | 2026-09-26 | pending | 4 |`
