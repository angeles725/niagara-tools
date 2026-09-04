# Retro (PROPOSED delta — propose-never-apply) — a curated corpus index for build-n4-module: the RT-authoring + organization blocks the skill should take into account

- **Date**: 2026-09-03
- **Origin**: a niagara-research `/research-sdd` run that reconstructed the RT block model, the Wire Sheet, and the Honeywell module-organization taxonomy (corpus blocks **B729–B750**). Operator asked: survey the documentation and tell build-n4-module which docs serve it, which blocks, what logic to take into account.
- **Status**: PROPOSED. This retro adds no rule and edits no methodology; it proposes a curated block index + specific wiring so a human can accept it. Following the skill's own step 6 (append PROVEN lessons as PROPOSED deltas).

---

## Finding

The kit already tells the builder "**corpus-nav FIRST**" and references a handful of blocks (B166, B179, B180, B194, B202, B203, B213, B636, B724, **B729, B730**). But the niagara-research corpus now carries a COMPLETE RT-authoring body — the campaign **B729–B746** plus the Wire-Sheet + organization work **B747–B750** — and only **B729/B730** of it are wired into the kit. That is the single largest block of directly-relevant documentation the skill is not yet pointing the builder at.

"corpus-nav FIRST" is necessary but not sufficient: lexical search finds a block only if the builder already knows the term. A CURATED INDEX (block → what it gives → when to read it) turns the corpus from "searchable if you know the word" into "the reading list for building a module."

Several of these lessons ALREADY have kit retros (so they are partly covered): B729 → `self-firing-timer-needs-started-*`; B739 → `slot-type-change-rompe-bog-*`; B741 → `qa-stack-pure-tests-*`; B746 → `module-palette-and-build-target`; the B740 cross-module-enum lesson is already in `types/logic.md` ("Linking across custom modules"). The gap is the rest, and the lack of one place that maps them.

## Proposed deliverable: `$KIT/corpus-index.md` (a new file), referenced from SKILL step 2 and `types/logic.md`

A curated map. Priority = how load-bearing it is for a correct build (P0 = read before building, P1 = read for the relevant layer, P2 = read when the feature/decision comes up).

| Block | What it gives the builder | Feeds kit file | Priority | Already wired? |
|---|---|---|---|---|
| **B744** | The ONE consolidated "what an RT block is" index (parts, format, rules, what it shows/does/discovers) — the entry point that links every other block | `types/logic.md` (start-here) | **P0** | no |
| **B737** | Engine thread + watchdog, and **composition**: group concerns into CHILD components instead of a flat slot wall (the fix for our 25-slot `BEvaporatorUnit`) | `types/logic.md` (new "Composition" section) | **P0** | no |
| **B730** | The rt idioms catalog (execute/changed discipline, timers, flags, degrade-honestly) | `types/logic.md` | **P0** | **yes** |
| **B729** | Timer lifecycle: arm in `started()` + `atSteadyState()`, `clockChanged` | `types/logic.md` | **P0** | **yes** (retro) |
| **B739** | Schema evolution — **ADD, never retype** an existing slot (retype breaks the `.bog`, station won't boot). Outage-prevention | `METHODOLOGY.md` (schema check) | **P0** | **yes** (retro) |
| **B740** | Cross-module links use a plain `double`, never a shared frozen enum (`Missing class …HoaMode` outage) | `types/logic.md` | **P0** | **yes** (in logic.md) |
| **B734** | Point-type taxonomy (4×2, point vs writable/priority array, proxy vs local, extensions) — decide component vs point | `types/logic.md` | P1 | no |
| **B735** | Slots/facets/links mechanics + **why SUMMARY/HIDDEN/BIUnlinkable curate the Link picker and the wire sheet** | `types/logic.md` + `METHODOLOGY.md` (flags) | P1 | partial |
| **B736** | The `BStatus` 8-bit model — set/propagate fault/down/stale/null/overridden honestly | `METHODOLOGY.md` (status check) | P1 | partial |
| **B745** | Units — `BUnit`/`UnitDatabase`, the `units` facet, how to put °C/kPa/% on a slot | `METHODOLOGY.md` (facets) | P1 | partial |
| **B738** | Practical how-to: add a proxyExt, facets, `propagateFlags`, an icon/SVG to a block | `types/logic.md` / how-to | P1 | no |
| **B746** | Module palette authoring (BOG XML) + **pre-wired assembly templates** so commissioning is drag-one-thing | `types/*` + palette retro | P1 | **yes** (retro) |
| **B749** | **The Honeywell block-organization taxonomy — the 10 recurring patterns** (see "logic to take into account" below) | new `types/organization.md` or `corpus-index.md` | **P0** | no |
| **B750** | The taxonomy applied to OUR modules — 5 actionable gaps + deploy-safe sequence | `types/organization.md` | P1 | no |
| **B747** | The Wire Sheet as a flow surface: **pin = SUMMARY exactly**; live values render with facets + status color | `types/logic.md` (pins) | P2 | no |
| **B748** | Interactivity/low-cognitive-load playbook (curate pins, compose, icons, palette, tags) | `types/organization.md` | P2 | no |
| **B732** | Authoring real alarms — `BAlarmSourceExt` is a point extension; the offnormal/fault algorithm family | `types/logic.md` (when adding alarms) | P2 | no |
| **B733** | Modulating (0-10V) outputs, `kitControl.BLoopPoint` PID, the math block family | `types/logic.md` (when adding PID/AO) | P2 | no |
| **B741** | The 4-layer QA/test stack; what's testable in WSL | `build-verify.md` | P1 | **yes** (retro) |
| **B743** | Testing timer-arming/lifecycle — the scheduler seam + `BTestNgStation` | `build-verify.md` | P2 | no |
| **B731** / **B742** | Our own modules' audit + the consolidated deploy-safe refactor backlog | reference | P2 | no |

## The LOGIC the skill must take into account (the non-obvious rules)

These are the rules a builder gets wrong without the docs — the "qué lógica" the operator asked for. Most are the distilled Honeywell patterns (B749) confirmed against our own outages:

1. **Compose into child components; don't sprawl flat slots.** One child `BComponent` per concern/domain (timing, outputs, hoa, freeze). This is B737 AND exactly how every Honeywell module distributes (B749 P2). Above ~12–15 slots, flat is wrong.
2. **Separate config from live-state.** Put tunables in a frozen `config` child; keep value + `BStatus` on the component (Honeywell `honIOBase` pattern, B749 P3 / B750). Our field outputs are BLinks to proxy points, so we skip Honeywell's third "wire-map `BStruct`" plane.
3. **SUMMARY = the wire-sheet pin, exactly** (`SlotBarGlyph.java:56 Flags.isSummary`). Curate pins: SUMMARY on real I/O, HIDDEN on internals, non-summary for interim state (B735/B747). This is what keeps a block from "desbordando."
4. **ADD, never retype a slot with saved data** (B739) — retype breaks the `.bog`, the station won't boot. A real outage. The single most important schema rule.
5. **Cross-module value = plain `double`, never a shared enum type** (B740) — a shared enum forces a hard inter-module dependency and left a `Missing class …HoaMode` on the live JACE.
6. **A self-armed timer arms in `started()`, not only `atSteadyState()`** (B729) — else it never fires on a late mount (the defrost-never-armed bug).
7. **Alarms NOTIFY, they never STOP control** — do not wire alarm-limit slots into the control decision (already in `types/logic.md`; grounded in B732).
8. **Ship pre-wired palette assembly templates** (B746, B749 P6) — Honeywell's whole Venom/IRM reuse layer is palette-only BOG trees. Bakes in correct nesting + flags (avoids the `hasDefrost=false → never defrosts` trap).
9. **Tag components for BQL discoverability** as a semantic overlay, not by nesting (B749 P9).
10. **Grouping folders are TYPED and self-validate** via `isParentLegal`/`isChildLegal` (B749 P4) — the tree rejects a wrong child.

## Proposed deltas (propose-never-apply — for human review)

1. **Create `$KIT/corpus-index.md`** = the table above (the curated reading list), and add one line to SKILL step 2 and to `types/logic.md`: "Before building, skim `corpus-index.md` — the P0 blocks (B744, B737, B730, B729, B739, B740, B749) are the reading list."
2. **Add a "Composition & organization" section to `types/logic.md`** pointing at B737 + B749/B750 (the file today is flat-slot oriented; it teaches idioms but not the tree shape). Include the one-line rule: distribute by containment with fixed roles; one child per domain; config separate from state.
3. **Optionally add `$KIT/types/organization.md`** if the organization body (B749/B750) grows past a section — the 10 patterns + the applied playbook are a distinct concern from per-layer idioms.
4. **Add to `METHODOLOGY.md`** two check items already grounded in P0 blocks: (a) "Compose concerns into child components above ~12–15 slots (B737)"; (b) the schema-safety line "ADD slots, never retype one with saved data (B739)" (currently only a retro).

No file above is edited by this retro — these are proposals. B739/B729/B741/B746 lessons already have retros; this consolidates the map and adds the missing composition/organization/reference blocks.

## Evidence
- Kit inspected: `SKILL.md`, `METHODOLOGY.md`, `types/logic.md`, `retros/` (2026-09-03). Existing block refs: B166/B179/B180/B194/B202/B203/B213/B636/B724/B729/B730.
- Corpus blocks surveyed: B729–B750 (niagara-research), read/cited this session; organization taxonomy from a 5-sweep Honeywell census (B749).
- Cross-checked which lessons already have kit retros to avoid duplicate proposals.
