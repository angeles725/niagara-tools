<!-- review-status: pending -->
<!-- Marker lifecycle: maintainer flips 'pending' → 'applied <date> · kit <sha>' (or 'dismissed') once folded; sweep-retros.sh reads this (METHODOLOGY §18). -->
# Retro — niagara-research · research-sdd · 2026-09-05 · FOCUS CLOSE: `own-modules-vs-exemplars` (OMV1–OMV7) — conformance audit of our 3 modules vs the exemplar idioms

> **Focus-wide §18 close.** Audited ColdRoomPan / CompPan / DashboardPan against the exemplar authoring idioms
> (B772–B785 + B763), chihuahua as reference. 7 dimensions; a block written ONLY where the audit found something
> (no padding). 3 finding-blocks (B787 timers, B788 palette/lexicon, B789 children+subscription); OMV1 actions &
> OMV6 background-work CONFORM (no block); OMV7 write-surface already covered by B763. Every finding grep-verified.
> Two routings per finding: a CLIENT punch-list item + a KIT implication (biting-check for QA, or an advisory).
> READ-ONLY on both the modules and the build kit — PROPOSES only.

## CLIENT punch-list (module changes — out of kit scope; for the module owner)
| Module | Item | Block · severity |
|---|---|---|
| ColdRoomPan `BEvaporatorUnit` | Add `stopped(){ cancelTicket(); super.stopped(); }` (4 delay tickets not cancelled on stop); optional `isRunning()` guard on the 4 expiry callbacks | B787 · LOW-MED |
| ColdRoomPan `BColdRoom` | Add `isChildLegal` (only `BEvaporatorUnit` 1..3 + optional `BDefrostController`) — control-by-child-order breaks if an integrator drops an extra/foreign child | B789 · LOW-MED |
| ColdRoomPan `BDefrostController` | Replace the 5 s `resistanceTemp` poll with a `Subscriber` on the unit's slot (edge-terminate; drop the self-rescheduling ticket) — already self-TODO'd | B789 · LOW |
| ColdRoomPan-rt lexicon | Add missing keys: `fanMode`, `valveMode`, `resistanceMode`, `freezeSetpoint`, `freezeProtect`, `freezeDiff*`, BFanMode options | B788 · MED |
| DashboardPan-rt lexicon | Add ~55 missing `BRoomPanel` operator/config keys (~25% coverage today) | B788 · MED |
| DashboardPan-wb | Delete the empty scaffold palette/lexicon or populate it (B5 footgun — passes the gate, nothing to drag) | B788 · LOW |
| DashboardPan-ux servlet | Add a per-Ord write lock + HTTP 423; port chihuahua's pure `canWrite(boolean)` RBAC test seam | B763 §763.6 · LOW-MED |
| CompPan | CLEAN — no items | — |

## KIT implication — BITING CHECKS for `verify-module.sh` (QA RED-first), split by what can actually bite
**HARD-FAIL (deterministic, ship first):**
- **lexicon DUP bare keys** — `grep -v '^#' | cut -d= -f1 | sort | uniq -d`. A repeat is always a last-wins bug;
  near-zero FP. Clean on our corpus today → a good regression guard. (B788)

**STRUCTURAL / WARN (biteable with a small parser):**
- **Clock.Ticket-owner-without-`stopped()`-cancel** — a class with a `Clock.Ticket` field / a `Clock.schedule` call
  but no `stopped()` that cancels it. Fires on `BEvaporatorUnit` TODAY, passes `BDefrostController`/`BCompressorControl`
  → a real RED→GREEN fixture. The strongest new finder. (B787)
- **empty-palette WARN** — a `module.palette` with zero non-Folder `<p>` while the module ships ≥1 non-enum/non-editor
  `@NiagaraType` (catches DashboardPan-wb). (B788)
- **lexicon type/slot coverage-%** — a key per `@NiagaraType` + per user-facing (SUMMARY|OPERATOR, not TRANSIENT/HIDDEN)
  slot; WARN below ~80% (flag-filtered to avoid noise; servlet-UI modules legitimately run low). (B788)
- **discarded `Clock.schedule` return** grep — cheap regression guard (0 hits today). (B787)

**ADVISORY only — NOT statically lintable (a kit-methodology finding in itself):**
- action-without-`Flags.OPERATOR` (OMV1): write-vs-read isn't statically decidable → false-positives on every legit
  admin action; advisory review-line, not a fail.
- order-sensitive container without a legality guard (OMV3-1): a lint can't know a container is order-sensitive.
- poll-that-should-subscribe (OMV5-1): needs semantic judgment vs a legitimate heartbeat (CompPan tick).
→ META-delta: the kit checklist should DISTINGUISH conformance rules that can bite as a `verify-module.sh` lint
  (dup-keys, empty-palette, ticket-without-stopped-cancel, coverage-%) from those that must stay HUMAN-REVIEW
  (action-gating intent, container order-sensitivity, poll-vs-subscribe). Don't ship the un-liftable ones as hard fails.

## KIT CORRECTION (for campaign6)
- The build kit's T8 note "CompPan-rt.lexicon empty" is FALSE — CompPan-rt/module.lexicon has **56 populated keys**
  (B788). Fix or drop that note in BUILD-STATE/METHODOLOGY.

## Already covered (dedupe)
Every audit block CITES the idiom it tests (B772/B775/B776/B778/B779/B780/B759/B763/B774) + the audit corpus
(own-modules-audit, B760 punch-list) and adds only the conformance verdict + routings — no re-derivation.

## What went well (keep)
- The no-padding discipline held: 2 of 7 dimensions were CLEAN and got a one-line verdict, NOT a manufactured block
  (OMV1 actions, OMV6 background-work); OMV7 cited B763 instead of duplicating it. 3 blocks for 7 dimensions.
- The audit turned the documented idioms into ACTIONABLE output: a client punch-list (8 concrete module fixes, several
  already self-TODO'd in our source) + a QA biting-check menu that honestly separates lintable from advisory.
- grep-verify caught + corrected a stale corpus claim (CompPan lexicon "empty" → 56 keys) and kept my own second-read
  honest (retracted a mis-grep B772 note when companero pushed back).
