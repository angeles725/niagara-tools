<!-- review-status: pending -->
# 2026-09-25 · module · panccadia-commissioning-lessons

**Session**: PANCCADIA León field support, tail of `feat/comppan-fase2-amps-alarms`
(`Cliente/panccadia-leon`, feature doc `odd/tasks/comppan-fase2-amps-alarms.md`,
commits `eeab639..c6077ca`). Same session already produced two other retros —
`retro/2026-09-24-panccadia-comppan-fase2-amps-alarms` (PR #157, 8 deltas, the
T1-T3 over-amps/no-amps-fault work) and `retro/2026-09-25-panccadia-unknown-unit-outage`
(commit `e32604e`, 4 deltas, the `pumpDownCutout` unit-id station outage) — this
retro covers the REMAINING frictions from the T4-T6 Fase-2 redesign and the
live-commissioning tail that neither of those two retros captures.

**Delta count**: 14 (0 dropped — all fourteen verified as still-open against the
current kit source and against both sibling retros; two live findings strengthen
an existing delta instead of becoming a new one — see Evidence)

## What happened
The 2026-09-25 redesign (T4 built and reverted, T5 rebuilt) and the live,
read-only oBIX commissioning pass that followed it exposed further gaps the
other two retros do not cover, plus a second, deliberately-fast pass over the
same session's process itself. (1) A facade link-in status-slot pattern
(`SUMMARY|READONLY`) that Workbench has never been able to link into, discovered
while trying to commission the drip/freeze-active status links and confirmed
against stock `LinkCheck.java`. (2) "Fase 2" was redefined three times in one
session because the client's phrase "setpoint de baja" meant the low-pressure
CUTOUT (pump-down/LP stop), not a control setpoint — resolved only after the
field refrigeration technician (Luis) restated the required sequence directly.
(3) The client's "keep one compressor always on" request shipped combined with
an already-disabled LP floor, reproducing the exact "compressors pulling with
all solenoids closed" field complaint that triggered the whole redesign. (4) The
live oBIX audit after each Workbench step caught a wrong link source, missing
links, and a rotation-liveness edge case that no static kit check could have
caught, plus a persisted run-hours counter that reset to ~0 on the very first
boot of the new backup feature. (5) The HMI panel used for this deployment is a
different vendor/OS than the kit's exemplar kiosk and silently failed to render
its login screen — a modern CSS/JS feature its embedded Chromium does not
support. (6) BUILD-SKILL step 1c (a design shard before a new state machine or
new facet/unit) was skipped all session for speed, and both the pump-down state
machine (3 design defects found only by native review, after implementation)
and the unknown-unit outage trace back to that same skipped step. (7) Several
changes silently redefined what a downstream consumer (the web viewer, over
oBIX) reads at the same point path — amps threshold, hours-counting predicate,
`pressureFallback`'s meaning, `minStagesOn`'s default, unlinked alarm slots —
and the consuming team had to be notified by hand, four separate times. (8) The
operator could not find slots in Workbench because the facade has no lexicon
entries (English auto-names) while the control side has Spanish ones, and the
commissioning link tables were written with internal names and had to be
redone with display names. (9) The v4/v5 jars went straight to the production
JACE (stock Tridium 4.15.3.28) after gates that ran only on an OEM Honeywell
4.14 overlay — the same distribution mismatch that caused the unknown-unit
outage, still with no smoke step naming the target distribution explicitly.
(10) Two alarm/cutout thresholds (`overAmpsLimit`, `pumpDownCutout`) shipped
disabled by default because the nameplate/field value was never obtained, with
no tracked place recording that the value is still owed. (11) The user asked
for maximum speed and several lints/sweep scenarios were skipped as a result,
with no named floor of checks that stays non-skippable regardless.

## Evidence
- Feature doc (full task-by-task evidence): `[ev: Cliente/panccadia-leon/odd/tasks/comppan-fase2-amps-alarms.md]`
- Commits: T4 `b01b1fd` (reverted by `f40db14`), T5 `9c80d53`, T6 `49c79ef`,
  live-commissioning fix `ae98f79`, doc `77c29e7`/`c6077ca`.
  `[ev: Cliente/panccadia-leon git log eeab639..c6077ca]`
- Direct read of the live module source confirming Δ1: `Dashboard/DashboardPan/
  DashboardPan-rt/src/com/angeles/DashboardPan/BRoomPanel.java:301-311`
  (`evap1/2/3FreezeActive`, `evap1/2/3InDrip`, all `Flags.SUMMARY | Flags.READONLY`)
  against the SUMMARY-only slots that already link fine (`comp1State:291`,
  `comp2State`, `intercambiadorState`) in the same file. `[ev: BRoomPanel.java:291,301-311]`
- Stock source confirming the mechanism: `javax/baja/sys/LinkCheck.java:148`
  (`if (Flags.isReadonly(target, targetSlot)) return invalid("linkcheck.propReadonly", cx);`).
  `[ev: docSource-doc/vineflower/baja/javax/baja/sys/LinkCheck.java:148]`
- Feature doc's own still-open TODO recording Δ1 before this retro existed:
  `odd/tasks/comppan-fase2-amps-alarms.md:186` ("P1 DashboardPan: facade status
  slots ... are `Flags.SUMMARY | Flags.READONLY` ... so Workbench refuses to link
  into them ... Fix: drop READONLY on these link-in status slots").
  `[ev: odd/tasks/comppan-fase2-amps-alarms.md:186]`
- Control-semantics churn (Δ2) and the LP-protection combination defect (Δ3):
  `odd/tasks/comppan-fase2-amps-alarms.md:168-174` ("Field feedback (Luis,
  refrigeration tech): with all solenoids closed the rack kept one compressor
  running (\"jalando sin nada\") because minStagesOn=1 and suctionLowLimit=0
  live"); T4 commit `b01b1fd` (rooms-base + ±1 pressure trim) reverted by
  `f40db14`; T5 commit `9c80d53` (Fase 1 staging + pump-down on stop, immediate
  stop bypassing min-on). `[ev: odd/tasks/comppan-fase2-amps-alarms.md:42-43,168-182]`
- Live oBIX commissioning findings (Δ4): `odd/tasks/comppan-fase2-amps-alarms.md:182`
  ("comp1/2/3State relinked from condenserNRunning (relay links removed) ...
  Pending: inDrip -> evapNInDrip links; suctionLowLimit still 0; compressor 3
  commanded with condenser3Fault=true (burned unit)"); this session's rotation
  read confirmed the burned unit (0.08 A, `condenser3Fault=true`, ~0.013 h) was
  picked FIRST by the least-hours rotation — a LIVE confirmation of the exact
  consequence `retro/2026-09-24-panccadia-comppan-fase2-amps-alarms.md` Δ6
  already proposed a rotation-consequence check for (T1b's commanded->running
  hour-integration change); cited here as added evidence for that delta, not
  proposed again. `[ev: odd/tasks/comppan-fase2-amps-alarms.md:73,163,182]`
  `[ev: retro 2026-09-24-panccadia-comppan-fase2-amps-alarms.md Δ6]`
- HMI browser-compatibility finding (Δ6): live commissioning session observation
  on the Exor UN78 panel HMI (Linux 4.14, Config OS 2.0.560, 2022) — the Domo
  login theme's background did not render (CSS `inset:0` unsupported by the
  panel's embedded Chromium, producing a 0x0 box) and the login button did
  nothing; diagnostic pending, not yet root-caused in source. `[ev: <live HMI
  commissioning session, 2026-09-25, Exor UN78 panel>]`
- Skipped design shard (Δ7): native RDD review of the pump-down state machine
  (T5) found 3 design defects only AFTER implementation — no `suctionValid`
  gate on pump-down entry, `minOn` bypassed for ALL units instead of only the
  last one, `pumpDownMaxTime<=0` allowed — recorded as review follow-up P2 in
  the feature doc; the unknown-unit outage (`retro/2026-09-25-panccadia-unknown-
  unit-outage.md`) also traces to a writer choosing a facet unit with no design
  review beforehand. `[ev: odd/tasks/comppan-fase2-amps-alarms.md:187 review-
  0130c696f9e8cc6e]; [ev: retro 2026-09-25-panccadia-unknown-unit-outage.md
  root-cause commit 49c79ef]`
- Downstream consumer-impact churn (Δ8): the same feature doc's own commit
  history changed the live meaning of `condenserNRunning` (5 A -> 2 A proof
  threshold), `condenserNHours` (commanded -> running predicate; resets to ~0
  at the hours-backup feature's first activation), `pressureFallback` (Fase 2 =
  band -> Fase 2 = pump-down), and `minStagesOn` (default 1 -> 0), plus left
  new alarm slots unlinked — each a live redefinition of a point the web-viewer
  consumer already polls, requiring four separate manual notifications this
  session. `[ev: odd/tasks/comppan-fase2-amps-alarms.md:56,68,124,139,169-182]`
- Operator findability (Δ9): `report-module.sh`'s T3 table (this same feature
  doc) already records `slot-coverage (type-set) DashboardPan-rt WARN 66.7%,
  missing CompressorPanel` as a PRE-EXISTING gap, confirming `BCompressorPanel`
  ships with no lexicon entries while the control-side `BCompressorControl`
  does — the commissioning link tables in this feature doc are themselves
  written with internal slot names only, with no display-name column.
  `[ev: odd/tasks/comppan-fase2-amps-alarms.md:103 slot-coverage row]`
- Cross-distribution deploy without a matching boot smoke (Δ10): the same
  schema-risk BEFORE/AFTER workflow in T3 built and verified every jar on
  `~/niagara-mirror-hon414` (an OEM Honeywell 4.14 overlay) and packaged v3/v4
  straight to `/mnt/c/Users/.../Downloads/PANCCADIA-modulos-*` for the
  production JACE (confirmed stock Tridium 4.15.3.28 by the unknown-unit
  retro's own console log) — the exact vendor/version mismatch axis that retro
  `2026-09-25-panccadia-unknown-unit-outage.md` Δ4 already names as unprovable
  by any check in this kit; this delta proposes the missing PROCEDURAL step
  (a boot smoke ON the target distribution) that Δ4 stopped short of.
  `[ev: odd/tasks/comppan-fase2-amps-alarms.md:124 "confirmed vendorVersion"]`
  `[ev: retro 2026-09-25-panccadia-unknown-unit-outage.md Δ4]`
- Values owed by the field (Δ11): `overAmpsLimit` shipped default 0 = disabled
  with no nameplate RLA value obtained, and `pumpDownCutout` shipped default 0
  with the feature doc noting "Luis has no value yet"
  (`odd/tasks/comppan-fase2-amps-alarms.md:174`) — both real, but nowhere is
  there a standing, closeable record of which values are still owed by the
  field. `[ev: odd/tasks/comppan-fase2-amps-alarms.md:56,174]`
- Speed-pressure skipped checks (Δ12): the feature doc's own T3 table records
  `lint-write-path` as SKIP/ERROR (no `docs/write-path-matrix.md` in the whole
  repo, never created) and the HMI no-scroll/banner sweep was re-derived from
  scratch twice in the same session (already retro'd as `retro/2026-09-24-
  panccadia-comppan-fase2-amps-alarms.md` Δ3) — both are checks that lapsed
  under an explicit speed request, with no kit-declared floor of checks that
  must run regardless. `[ev: odd/tasks/comppan-fase2-amps-alarms.md:63 lint-
  write-path ERROR/exit 3]`
- Rotation-liveness, second live occurrence (added evidence for `retro/2026-09-
  24-panccadia-comppan-fase2-amps-alarms.md` Δ6, not a new delta): after the
  v5 restart (~02:32), the least-hours rotation again commanded the burned
  compressor 3 first (0.08 A, `condenser3Fault=true`, ~0.013 h), running the
  rack one compressor short until the operator manually set it to `Apagar` —
  the same consequence Δ6 already proposed a check for, now confirmed live
  twice in the same night. `[ev: odd/tasks/comppan-fase2-amps-alarms.md:182,189]`
  `[ev: retro 2026-09-24-panccadia-comppan-fase2-amps-alarms.md Δ6]`
- Kit-side verification of each candidate against the CURRENT kit source (this
  session, 2026-09-25): `types/dashboard.md` (link-direction doctrine line 11,
  the READONLY timer-anchor pattern line 20, the "HMI kiosk" section lines
  157-163, no `READONLY`/`LinkCheck` doctrine anywhere else in the kit);
  `types/logic.md` (line 10 slot-flags convention, lines 70-71 HOA/minOn
  doctrine, no field-technician/ambiguous-term or proof-fault-rotation
  doctrine anywhere in the kit); `toolbelt/lint-config-sanity.sh` (CS1/CS2/CS3
  — none of the three checks a cross-slot "permanent minimum + disabled LP
  floor" combination); `toolbelt/bog-audit.sh` CHECK7/CHECK9/CHECK11
  (structural link validity — dangling/orphan/no-fallback — never "is this the
  DECLARED correct source"); `toolbelt/commissioning-verify.sh` and `toolbelt/
  generate-wiring-map.sh` (wiring-map columns carry internal slot names only,
  no display-name column; no live-oBIX diff against a declared expected-link
  source); `toolbelt/rc-scan.sh` (ord-literal/host-literal/bare-catch/
  null-branch only — no legacy-Chromium CSS/JS feature check); `toolbelt/slot-
  coverage.sh` (`--strict` promotes WARN to FAIL uniformly, with no
  facade-specific default); `ORCHESTRATION.md` / `skill/SKILL.md` step 1c
  (advisory-only — "open a design shard when..." — no gate, no checklist
  artifact, nothing that fails a session that skipped it); `METHODOLOGY.md:53`
  (the 4-layer assurance stack already names "a live cold-boot smoke" but does
  not require it match the TARGET's distribution/version); no `hoursBackup`/
  persistence-seeding, consumer-impact, values-owed, or non-skippable-minimum-
  set doctrine anywhere in the kit (`types/logic-authoring.md:578` is an
  unrelated EPHEMERAL-write note).

## Proposed kit deltas (propose-never-apply)
| Δ | Delta | Target file / § | Token |
|---|---|---|---|
| Δ1 | **New check + doctrine — a facade/link-in status slot must never carry `READONLY`, because `READONLY` blocks it as a Workbench Link TARGET.** Confirmed live: `BRoomPanel`'s `evap1/2/3FreezeActive` and `evap1/2/3InDrip` are declared `SUMMARY \| READONLY` and their own source comments claim they are "written by BLink" and name a "Commissioning link" — impossible per stock `LinkCheck.propertyToPropertyMake`, which rejects any link whose TARGET carries `READONLY` before it ever reaches type-checking. The sibling slots that DO link fine (`comp1State`, `comp2State`, `intercambiadorState`) are `SUMMARY` only. The kit's own existing `READONLY` doctrine (`types/dashboard.md:20`, `types/logic.md:35`) is for a DIFFERENT case — a timer/status anchor the OWNING component or its own reader sets itself, never an external Workbench Link target — and nothing in the kit currently distinguishes the two, which is exactly how this slot pattern shipped unlinkable. Fix: (a) add a lint (new `toolbelt/lint-link-target-flags.sh`, or a `lint-write-path.sh` extension) that FAILs when a slot listed as a link-in target in `docs/wiring-map.md` (from `generate-wiring-map.sh`) or matching the module's own "link-in"/"Linked from"/"Commissioning link" comment convention also carries `Flags.READONLY`; (b) add one clarifying bullet to `types/dashboard.md` right after the existing link-direction rule (line 11): a link-IN display slot is `SUMMARY` ONLY — `READONLY` is reserved for a slot the owning component/reader sets itself and that no external Link will ever target. | `toolbelt/lint-link-target-flags.sh` (new) + `types/dashboard.md` § "Link direction is fixed by role" (new bullet after line 11) | `[ev: BRoomPanel.java:291,301-311]; [ev: LinkCheck.java:148]; [ev: odd/tasks/comppan-fase2-amps-alarms.md:186]` |
| Δ2 | **New doctrine — restate a refrigeration/HVAC control-mode change in field terms and confirm with the field technician before coding it; keep a glossary of ambiguous client terms.** "Fase 2" was redefined three times in one session (suction-pressure band -> rooms-base + ±1 pressure trim, built and reverted -> Fase 1 staging + pump-down on stop) purely because the client's phrase "setpoint de baja" meant the LOW-PRESSURE CUTOUT (pump-down/LP stop), not a control setpoint, and this was only caught once the field refrigeration technician (Luis) described the required sequence directly. Before coding a control-mode change driven by an ambiguous field phrase, restate the sequence in concrete field terms (what each valve/compressor/solenoid does in each named state) and confirm it with the field technician, not just the requesting user; keep a short glossary entry in the feature doc for terms like "setpoint de baja" (= LP cutout / pump-down trigger, not a tunable control setpoint), "presostato" (= physical pressure switch, distinct from a soft cutout), and "pump-down" (ends with an immediate stop, bypassing min-on). | `types/logic.md` § RT control logic (new bullet near the existing HOA/minOn doctrine, lines 70-71) | `[ev: odd/tasks/comppan-fase2-amps-alarms.md:42-43,168-182]` |
| Δ3 | **New `lint-config-sanity.sh` check (CS4) + doctrine — a nonzero permanent-minimum/floor default must not coexist with a disabled LP/cutout-floor default.** The client's "keep one compressor always on" request (`minStagesOn` default 1) shipped WHILE `suctionLowLimit` (the LP protection floor, 0 = disabled by the class's own documented convention) was still 0 — reproducing the exact "compressors pulling with all solenoids closed" complaint that started the redesign. This is a cross-slot combination `lint-config-sanity.sh`'s existing CS1 (interval<=duration) and CS2 (zero OPERATOR setpoint with no min>0 facet) do not check: CS2 does not fire here because `suctionLowLimit` correctly carries a MIN facet as an OPERATOR tunable — the danger is the COMBINATION with a separate always-on floor slot, not either default in isolation. Proposed CS4 WARN: a nonzero `*MinStagesOn*`/`*MinOn*`-named default coexisting with a `*LowLimit*`/`*Cutout*`-named default of 0 in the same class. Pair with a doctrine bullet: an LP "limit" that sheds only ONE stage per `stageDelay` while respecting `minOn` (the kit's existing pattern) is NOT a safety cutout — a true cutout must override `minOn` and stop immediately, exactly as this session's pump-down redesign (T5) correctly implements. | `toolbelt/lint-config-sanity.sh` § new CS4 check (near CS1/CS2) + `types/logic.md` § "minOn / stageDelay guards" (new bullet, line 71 area) | `[ev: odd/tasks/comppan-fase2-amps-alarms.md:169,171]; [ev: toolbelt/lint-config-sanity.sh CS1/CS2 header]` |
| Δ4 | **New tool — an expected-links manifest cross-checked against a live, read-only oBIX audit.** The live oBIX commissioning pass after each Workbench step found `comp1/2/3State` linked from raw relay command outputs (`io34_4_3`/`ro1..3`) instead of the amps-proven `condenserNRunning`, plus missing `inDrip`/`freezeActive` links — neither is a structural defect `bog-audit.sh`'s CHECK7 (dangling target)/CHECK9 (orphan handle)/CHECK11 (proxy-link-safety, no-fallback) would catch, since a link from the WRONG-but-valid source is structurally fine; those checks never know what the CORRECT source should be. `generate-wiring-map.sh` already scaffolds the declared facade<->control slot pairing (`docs/wiring-map.md`) but nothing diffs it against what is actually linked live. Propose a new script (extend `niagara-research/tools/obix-nav.py` or add a thin `toolbelt/obix-link-audit.sh` wrapper) that reads `docs/wiring-map.md`'s declared source-slot column and a live read-only oBIX dump, then emits a per-slot MATCH/MISMATCH/MISSING row — read-only, no station write, usable during commissioning exactly as this session's manual pass was. | `toolbelt/obix-link-audit.sh` (new, extends `niagara-research/tools/obix-nav.py`) + `types/dashboard.md` § wiring-map bullet (line 17, cross-reference) | `[ev: odd/tasks/comppan-fase2-amps-alarms.md:182]; [ev: toolbelt/bog-audit.sh CHECK7/9/11 header]` |
| Δ5 | **New doctrine — a persistence/backup feature's first deploy must seed the backup from live values before relying on it.** The compressor run-hours counter, persisted via the new hours-backup feature (`hoursBackupPeriod`/`hoursBackupExpired`/`hoursBackupWriteFailed`), reset to ~0 on the FIRST boot after that feature was introduced (no backup file existed yet); a later restart correctly preserved the accumulated hours. This is a general activation-order hazard for any first-deploy backup/persistence mechanism, not covered anywhere in the kit (`types/logic-authoring.md:578`'s EPHEMERAL-write note is a different, unrelated propagation-order gotcha). Add a checklist item: on the FIRST deploy of a new persistence/backup slot, seed it from the current live value (or explicitly document the one-time reset as acceptable) before the feature is relied on for continuity-sensitive data such as wear counters used in rotation. | `BUILD-LOOP.md` § commissioning checklist (new bullet, near the existing "after a reload, triage the console" step) | `[ev: odd/tasks/comppan-fase2-amps-alarms.md session note, "hours reset to ~0 on first boot after hours-backup feature, no backup file yet"]` |
| Δ6 | **New `rc-scan.sh` WARN check — legacy-Chromium-incompatible CSS/JS features on an HMI-panel target.** The Exor UN78 panel HMI (Linux 4.14, Config OS 2.0.560, 2022) failed to render its login background (CSS `inset:0` unsupported, collapsing to a 0x0 box) and its login button did nothing (suspected modern JS). The kit's existing "HMI kiosk" doctrine (`types/dashboard.md:157-163`) is written and verified against the WEB-HMI10/CF-class exemplar panel only, with no stated Chromium-version floor and no check for CSS/JS features a different, older/embedded Chromium may not support; `rc-scan.sh` currently only checks ORD/host literals and error-swallowing patterns, never CSS/JS feature compatibility. Propose a new WARN rule, gated on the module declaring an HMI-panel target, for `inset`, `min()`/`max()`/`clamp()`, `backdrop-filter`, `aspect-ratio`, `:is()`/`:where()`, and optional chaining (`?.`) — plus a doctrine caveat that the kiosk section's guidance is verified against ONE panel class and a different vendor/OS panel needs its own Chromium-version confirmation before relying on modern CSS/JS. | `toolbelt/rc-scan.sh` (new WARN check) + `types/dashboard.md` § "HMI kiosk" (new caveat, lines 157-163) | `[ev: <live HMI commissioning session, 2026-09-25, Exor UN78 panel, Config OS 2.0.560>]` |
| Δ7 | **Make BUILD-SKILL step 1c (design shard) a hard gate for a new control state machine or a new facet/unit, with a short design-shard checklist.** Step 1c today is advisory prose only ("open a `sdd-design` shard ... when the change introduces a schema annotation, a new slot, or a new façade type") with no artifact, no gate, and nothing that fails a session that skips it. It was skipped all session for speed, and the cost showed up twice: the pump-down state machine (T5) shipped 3 design defects — no `suctionValid` gate on entry, `minOn` bypassed for ALL units instead of only the last, `pumpDownMaxTime<=0` allowed — caught only by native RDD review AFTER implementation (review follow-up P2); and the unknown-unit outage traces to a writer choosing a facet unit with no design review at all. Fix: turn 1c into a named checklist artifact (states, entry/exit guards, invalid-sensor behavior, min-on/min-off interplay, config bounds, boot-critical expressions such as a facet's `BUnit.getUnit` call) that a session must produce (or explicitly waive with a reason) before writing a new state machine or a new facet/unit — mirroring the retro's own `Retro: none (trivial: ...)` waiver pattern rather than leaving it silently skippable. | `skill/SKILL.md` § step 1c (make non-advisory) + `ORCHESTRATION.md` § delegation triggers (new design-shard checklist) | `[ev: odd/tasks/comppan-fase2-amps-alarms.md:187 review-0130c696f9e8cc6e]; [ev: retro 2026-09-25-panccadia-unknown-unit-outage.md]` |
| Δ8 | **New doctrine — every feature doc gets a "Consumer impact" section, generated or checked at close, plus a notify step in the commissioning checklist.** This session silently changed the live meaning of four points a downstream consumer (the web viewer, over oBIX/Supabase) already reads: `condenserNRunning` (5 A -> 2 A proof threshold), `condenserNHours` (commanded -> running predicate, and reset to ~0 at the backup feature's first boot), `pressureFallback` (Fase 2 = suction band -> Fase 2 = pump-down), and `minStagesOn` (default 1 -> 0) — plus new alarm slots left unlinked. None of this is a schema break (`schema-risk.sh` correctly reports every change SAFE, since it only checks `.bog` decode survival, not point SEMANTICS), so nothing in the kit currently surfaces a semantic meaning-change to whoever consumes these points off-module; the consuming team had to be told by hand four separate times this session. Add a mandatory "Consumer impact" section to the feature-doc template (point path, old meaning, new meaning, effective when) and a corresponding notify checklist item in `BUILD-LOOP.md` §6/§6.b, checked at close alongside the retro gate. | `BUILD-LOOP.md` § 6.b Commissioning-verify requirement (new checklist item) + feature-doc template / `METHODOLOGY.md` (new "Consumer impact" section requirement) | `[ev: odd/tasks/comppan-fase2-amps-alarms.md:56,68,124,139,169-182]` |
| Δ9 | **Facade lexicon completeness should FAIL (not WARN) for an operator-facing dashboard/facade module, and a commissioning link table must always carry the Workbench display name alongside the internal slot name.** This session's own T3 table already recorded `slot-coverage (type-set) DashboardPan-rt WARN 66.7%, missing CompressorPanel` as a pre-existing, un-escalated gap — `BCompressorPanel` ships with NO lexicon entries, so its slots render as raw English camelCase in Workbench while the control-side `BCompressorControl` shows Spanish lexicon names, and the operator could not find the slots being discussed ("no encuentro los slots"). The commissioning link tables this session wrote (and `generate-wiring-map.sh`'s own scaffolded columns) name only the internal slot, not the Workbench-visible display name, so they had to be redone by hand against the live tree. Fix: (a) treat `slot-coverage.sh` WARN as FAIL by default (not only under `--strict`) for a module whose type is a facade/dashboard panel (operator-facing, not an internal control class); (b) extend `generate-wiring-map.sh`'s table columns to carry "internal name + Workbench display name (from the lexicon) + full ord", ideally auto-filled from the module's own `.lexicon` file. | `toolbelt/slot-coverage.sh` (facade-module default-FAIL mode) + `toolbelt/generate-wiring-map.sh` (display-name column) | `[ev: odd/tasks/comppan-fase2-amps-alarms.md:103 slot-coverage row]` |
| Δ10 | **Strengthen the existing cold-boot-smoke requirement (`METHODOLOGY.md:53`) to require the smoke run on the TARGET distribution and version, not just any live station, before a production deploy.** This session's own T3 gates (build, verify, schema-risk) all ran against `~/niagara-mirror-hon414`, an OEM Honeywell 4.14 overlay, and the resulting jars were packaged straight for the production JACE (stock Tridium 4.15.3.28) — the exact vendor/version mismatch axis `retro/2026-09-25-panccadia-unknown-unit-outage.md` Δ4 already names as something no static or dynamic check in this kit can verify. That retro's Δ4 stops at documenting the gap and pointing to `lint-unit-ids.sh` (Δ1) as the one closeable sub-case; this delta proposes the complementary PROCEDURAL gate Δ4 does not: a mandatory boot smoke on a station of the TARGET distribution and version (e.g. a matching PRUEBAS station, or a 4.15 docker/snap image) in `BUILD-LOOP.md` §6, before ANY production deploy — closing the parts of the vendor/version mismatch that a static unit-id lint alone cannot (any static-init exception, not only a bad `BUnit` id). | `BUILD-LOOP.md` § 6 Deploy (new mandatory pre-deploy bullet) + `METHODOLOGY.md:53` (amend the cold-boot-smoke bullet to name target-distribution matching) | `[ev: odd/tasks/comppan-fase2-amps-alarms.md:124 "confirmed vendorVersion"]; [ev: retro 2026-09-25-panccadia-unknown-unit-outage.md Δ4]; [ev: METHODOLOGY.md:53]` |
| Δ11 | **New commissioning-checklist artifact — a "values owed by the field" table that stays open until filled.** `overAmpsLimit` shipped default 0 (disabled) with no nameplate RLA ever obtained, and `pumpDownCutout` shipped default 0 with the feature doc itself noting "Luis has no value yet" — both real, both safety-adjacent, and neither tracked anywhere as an OPEN item once the session that introduced them closes. Add a required table to the §6.b commissioning-verify punch-list (slot, who provides the value, unit, current safe default) that `commissioning-verify.sh` prints as a `MANUAL` row when any such slot is still at its documented "disabled/no value yet" default, so it cannot silently age out once the feature doc itself is archived. | `BUILD-LOOP.md` § 6.b Commissioning-verify requirement (new "values owed by the field" table) + `toolbelt/commissioning-verify.sh` (new MANUAL row) | `[ev: odd/tasks/comppan-fase2-amps-alarms.md:56,174]` |
| Δ12 | **Name a minimum non-skippable check set that survives an explicit user request for maximum speed, and record which checks were skipped and why.** Under this session's own speed pressure, `lint-write-path` stayed SKIP/ERROR all session (no `docs/write-path-matrix.md` ever created for either module, a whole-repo gap that predates and outlives this branch) and the HMI no-scroll/banner sweep was re-derived from scratch twice (already retro'd separately as Δ3 of `retro/2026-09-24-panccadia-comppan-fase2-amps-alarms.md`) — neither is currently distinguished from an optional check a time-pressured session may legitimately defer. Define a minimum set that remains non-skippable regardless of time pressure — build, `verify-module.sh`, the unit-id lint once Δ1 of the unknown-unit retro ships, `schema-risk.sh`, and the target-distribution boot smoke (Δ10) — and require a session under an explicit speed request to record, in the feature doc, exactly which OTHER checks were skipped and why, rather than leaving it implicit in the commit history. | `METHODOLOGY.md` § checklist (new "non-skippable minimum" list) + feature-doc template (new "checks skipped under time pressure" note) | `[ev: odd/tasks/comppan-fase2-amps-alarms.md:63 lint-write-path ERROR/exit 3]; [ev: retro 2026-09-24-panccadia-comppan-fase2-amps-alarms.md Δ3]` |
| Δ13 | **New rotation-safety requirement — least-hours rotation must not repeatedly command a unit with an active proof fault; add the regression test that proves it.** The burned compressor 3 (`condenser3Fault=true`) was picked FIRST by least-hours rotation on TWO separate live occasions this session (initial commissioning, ~0.013 h; again after the v5 restart at ~02:32) — not merely "stays first while idle" (the case `retro/2026-09-24-panccadia-comppan-fase2-amps-alarms.md` Δ6 already covers as doctrine), but ACTIVELY COMMANDED, running the rack one compressor short until the operator intervened by hand each time. `autoOffOnProofFault` already exists as an opt-in escape hatch but was OFF both times. Propose a concrete, permanent regression test (e.g. in `CompressorControlTest`, mirrored as a kit doctrine requirement wherever a least-hours rotation is authored) asserting that least-hours rotation SKIPS a unit with an active proof fault, or that shipping such a rotation without `autoOffOnProofFault` forced on is itself flagged — making Δ6's abstract "pair a rotation-predicate change with a consequence test" concrete for the specific proof-fault case, rather than leaving it to operator vigilance. | `types/logic.md` § Rotation (new named test-pattern requirement, adjacent to the existing rotation doctrine) | `[ev: odd/tasks/comppan-fase2-amps-alarms.md:182,189]; [ev: retro 2026-09-24-panccadia-comppan-fase2-amps-alarms.md Δ6]` |
| Δ14 | **Process umbrella — "speed without risk: tier the work by blast radius."** This session's own costs rank cleanly by blast radius, not by effort spent: the THREE-WAY Fase-2 redefinition (band -> ±1 trim, coded/tested/reviewed/thrown away -> pump-down) was the single most expensive item — two full delegated-writer cycles (~20-25 min each) plus native review (~5-10 min) plus a revert, avoidable by ~10 minutes with the field technician BEFOREHAND (Δ2); the unknown-unit station outage was a structural/facet mistake an automated check would catch in seconds, yet cost a fix + reinstall + restart cycle (retro `2026-09-25-panccadia-unknown-unit-outage.md` Δ1's `lint-unit-ids.sh`); manual commissioning (redoing link tables with display names, an operator unable to find slots, one structurally IMPOSSIBLE link (Δ1's READONLY target), repeated manual oBIX checks) cost the most WALL-CLOCK time of all three but the least design risk. No doctrine in this kit currently classifies a change by blast radius BEFORE choosing how much process to apply to it — `METHODOLOGY.md`'s delegation-trigger rules classify by file count, not by control/structural/cosmetic risk, and this retro's own Δ12 names a non-skippable check FLOOR without a tiering scheme to derive it from. Propose a tier table in `BUILD-LOOP.md`/`ORCHESTRATION.md`: **Small** (HMI text, a default value, a comment) — inline, no delegated writer, minimal gate; **Control** (a new/changed state machine — staging, rotation, protections) — a written "sequence of operation" in field terms confirmed by the field technician FIRST (Δ2), then a design shard (Δ7), tests, and review; **Structural** (a new slot, facet, unit, or flag) — fast automated checks BEFORE deploy: the unit-id exists on the target (retro `2026-09-25-panccadia-unknown-unit-outage.md` Δ1), no `READONLY` on a link-in target (Δ1), and a consumer-impact/semantic-change notice (Δ8). Plus two cross-tier closers this session repeated manually: a same-distribution/version test station (stock Tridium 4.15, matching the JACE) for a boot smoke before any production deploy (Δ10), and commissioning automation — a generated link table with Workbench display names (Δ9) plus a ~10 s read-only oBIX link/value audit (Δ4). This delta is the umbrella that ties Δ1/Δ2/Δ4/Δ7/Δ8/Δ9/Δ10/Δ12 and the sibling unknown-unit retro's Δ1 together by WHEN each applies; it proposes no new check of its own beyond the tier table itself. | `BUILD-LOOP.md` § 0 Orient / `ORCHESTRATION.md` § delegation triggers (new blast-radius tier table, cross-referencing Δ1/Δ2/Δ4/Δ7/Δ8/Δ9/Δ10/Δ12) | `[ev: odd/tasks/comppan-fase2-amps-alarms.md:42-43,168-182 (Fase-2 redefinition cost)]; [ev: retro 2026-09-25-panccadia-unknown-unit-outage.md Δ1 (structural-tier check)]` |

## Lessons
- `READONLY` has two unrelated meanings in this kit's own doctrine — "the owning
  component/reader sets this itself" (fine) vs "an external Workbench Link must
  target this" (broken by `Flags.isReadonly` in stock `LinkCheck`) — and nothing
  distinguished them until a facade shipped unlinkable slots authored with
  comments claiming a Link that could never exist (Δ1).
- A client's own vocabulary for a control concept ("setpoint de baja") can mean
  something structurally different from what it sounds like (a cutout, not a
  setpoint); restate the sequence in field terms and confirm with the field
  technician before coding, not after two write/revert cycles (Δ2).
- A safety-adjacent default is not safe read in isolation — the danger showed up
  only in the COMBINATION of an always-on floor and a disabled LP cutout, a
  cross-slot pattern no single-slot lint checks (Δ3).
- A structural link-validity check (dangling/orphan/no-fallback) cannot catch a
  link from the WRONG-but-valid source; only a live read-only oBIX diff against
  a declared expected source closes that gap, and it is exactly the manual work
  this session repeated by hand (Δ4).
- A brand-new persistence/backup mechanism starts EMPTY on its first boot by
  construction — relying on it for continuity-sensitive data (a wear counter
  driving rotation) without seeding it first silently discards real history (Δ5).
- The kit's HMI doctrine is calibrated against one exemplar panel's embedded
  Chromium; a different vendor/OS panel is a distinct compatibility surface that
  needs its own check, not an assumption that "HMI kiosk" guidance transfers (Δ6).
- An advisory-only design-shard step is exactly as reliable as the time pressure
  of the moment it is needed most — both the pump-down design defects and the
  unknown-unit outage trace to the same skipped step; a real gate needs an
  artifact and a waiver-with-reason, not prose alone (Δ7).
- `schema-risk.sh`'s SAFE verdict proves `.bog` decode survival, not semantic
  stability — a point can keep its type and slot and still silently change what
  it MEANS to a consumer that never re-reads the source; that gap needs its own
  section, not an assumption that SAFE covers it (Δ8).
- A facade's own lexicon completeness is an operator-facing correctness
  requirement, not a cosmetic WARN — "I can't find the slot" is the direct,
  reproduced cost of treating it as optional (Δ9).
- Gating every check on the WRITER's dev/mirror station proves nothing about the
  DEPLOY TARGET when the two differ by vendor and version — this is the same
  lesson the unknown-unit outage already paid for, still unenforced as a
  procedural gate two commits later in the same session (Δ10).
- A safety-adjacent default shipped "disabled, no value yet" needs a standing,
  closeable record of who owes the real value — not just a comment in a feature
  doc that ages out when the doc is archived (Δ11).
- Speed pressure will find and skip the checks that lapse silently; naming a
  non-skippable floor and requiring an explicit skip-log makes the tradeoff
  visible instead of implicit in what a session happened not to run (Δ12).
- A rotation predicate "staying first while idle" and a rotation predicate
  "actively commanding a known-faulted unit" are different severities — the
  second happened twice live in one night with no test anywhere forbidding it
  (Δ13).
- This session's own costs rank by BLAST RADIUS, not by effort spent: the
  cheapest fix (~10 minutes with the field technician) would have avoided the
  most expensive mistake (two full write/test/review/revert cycles); classify a
  change as Small/Control/Structural BEFORE choosing how much process to apply
  to it, rather than discovering the right amount of process after paying for
  the wrong amount (Δ14).

---
**Status**: PENDING — INDEX row appended: `| 2026-09-25-panccadia-commissioning-lessons.md | module | 2026-09-25 | pending | 14 |`
