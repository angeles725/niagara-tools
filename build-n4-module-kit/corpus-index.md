# Corpus index — the curated map of the niagara-research authoring corpus (B729–B760)

The niagara-research corpus carries a COMPLETE N4 module-authoring body — the RT campaign (B729–B746),
the Wire-Sheet + organization work (B747–B750), the WB/UX authoring taxonomy (B751–B753), and the
module-authoring axes (B754–B760: versioning/upgrade-safety, bits, build/signing, integration, tags,
lexicon, and a consolidated audit). This index points the builder at the block that answers each need.

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

## P2 — read when that feature/decision comes up

| Block | What it gives the builder | Layer |
|---|---|---|
| **B743** | Testing timer-arming/lifecycle — the scheduler seam + `BTestNgStation` | build / test |
| **B747** | The Wire Sheet as a flow surface: pin = `SUMMARY` exactly; live values render with facets + status color | wire-sheet |
| **B748** | Interactivity / low-cognitive-load playbook (curate pins, compose, icons, palette, tags) | organization |
| **B757** | Station integration — authoring a `BAbstractService` (register-by-placement) + the nav tree | rt / service |
| **B732** | Authoring real alarms — `BAlarmSourceExt` is a point extension; the offnormal/fault algorithm family | rt / alarms |
| **B733** | Modulating (0-10V) outputs, `kitControl.BLoopPoint` PID, the math block family | rt / control |

## Research-tooling caveats (A18)

Before concluding a topic is undocumented: a tool returning zero results is not proof of absence — run a control query or fall back to `rg`/`mem_context`/the source (S1 [ev: retro rt-authoring-campaign Δ4]). Never mark a claim `[CERT]` from a mangled decompiled body — prefer the vineflower/procyon tree or mark `[INFER]` (S2 [ev: retro rt-authoring-campaign Δ3]).
| **B758** | Tags/relations authoring + northbound data exposure (tag dictionary, oBIX agent, Fox/BOX, BQL-from-code cursor) | ux / data |
| **B731** / **B742** | Our own modules' audit + the consolidated deploy-safe refactor backlog | reference |

---

Maintenance: this index is a curated pointer, not a copy — when a block's content is folded into a kit guide
(METHODOLOGY, BUILD-LOOP, `types/*.md`, build-verify), that guide carries the rule and cites the block; this
index stays the "what to read" map. Source: retro `corpus-index-rt-authoring-and-organization-blocks`.
