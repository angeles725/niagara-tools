<!-- review-status: pending -->
<!-- kit-retro -->
# Retro — Campaign 6 PR7: research fold (exemplar-backed author-side SPIs, point extensions, web-tier pointers, lintable-vs-advisory)

Date: 2026-09-05 · Module: kit (self) · SDD change: build-n4-module-campaign6 PR7

## What was folded

27 research deltas from four niagara-research retros (exemplars focus B772–B785 + own-modules focus + web-tier audit) folded into the kit core:
- `types/logic.md`: 10 new sections — Author-side SPIs (idiom + 3 instances), point extension (B772), child-tree containers (B779), grouping/relating postures (B781), query/search/index surface (B782), templates as artifact production (B783), background jobs (B774), watchdogs and timers (B775), action protection (B776), minimal module copy-start (B790).
- `types/dashboard.md`: module palette+lexicon conventions (B780), web-tier exemplar pointer table (DUX-WEB1, B791), DashboardPan divergences (DUX-WEB2, B791/B763).
- `types/wb-widgets.md`: dual-surface `@AgentOn` registration (B780).
- `METHODOLOGY.md`: module.xml profile+dependency conventions (B784), permissions-inline correction (B777), watchdog-note correction (B775), conformance-rules lintable-vs-advisory + human-review checklist (B787/B788/B789, B776/B789), kit-continuity folded-as-code line.
- `corpus-index.md`: 27-row delta covering B762–B791 grouped by kit destination.

## T7.5 — fold-audit strict switch

Two folded retros had no `[ev: retro …]` citation in the kit core:
- `junit-standalone-cached-jar-locations-for-wsl-pure-tests`: content lives in `build-verify.md` line 106 (Gradle cache tip + `run-pure-test.sh`). Citation was `[ev: retro rt-hardening #5; junit-standalone · B11]` — the semicolon-separated format was not parsed by the audit grep. Fixed to `[ev: retro rt-hardening] [ev: retro junit-standalone]` proper brackets.
- `kit-continuity-and-retro-gate-campaign`: content is code/schema (sweep-build-state.sh, BUILD-STATE.md, retros/INDEX.md, BUILD-LOOP.md §7). Added "folded as code" note in METHODOLOGY.md §Conformance rules with `[ev: retro kit-continuity]`.

`sweep-fold-audit.sh --strict` exits 0 after both fixes. CI step switched from non-strict to `--strict`.

## Bullet-vs-block disagreements found and fixed

None. The fold draft (authored by investigador1 from the source blocks) was accurate against all spot-checked blocks (B772, B775, B784, B791). Fidelity confirmed:
- B772: premise correction (`BAbstractPointExt` does not exist) preserved verbatim.
- B775: "NOT a 2s poll", "NO `javax.baja.sys.BTimer`", "EngineWatchdog is a SEPARATE layer" — all preserved.
- B784: `-doc` is a SEPARATE module (not a part), 3-part dep FLOOR vs 4-part build stamp — preserved.
- B791: DashboardPan CSRF divergence "DELIBERATE, STRONGER-than-vendor" — preserved.

## Rules found already present (grep-before-fold)

None — all 10 types/logic.md sections and all dashboard.md/wb-widgets.md/METHODOLOGY.md bullets were absent before folding (grep-before-fold confirmed 0 pre-existing hits for each pattern).

## Key learnings

1. The `[ev: retro token · B11]` mixed format was not parsed by the audit grep — always use `[ev: retro token]` as a standalone bracket.
2. Code-folded retros (shipped as scripts rather than prose rules) need a "folded as code: <script> [ev: retro <token>]" line in a prose kit file to be credited by the citation audit.
3. The B773 analytics-node exception (NO @AgentOn — registered by plain `<type>`) is a critical distinction that breaks the "one idiom for all SPIs" assumption — explicitly folded as a named EXCEPTION.
4. The strict fold-audit gate is a valid CI promotion: 38 folded, 38 cited, 0 uncited, exit 0.

## Lesson added at review time — source file vs inlined artifact

QA's independent spot-check found METHODOLOGY line 15 (B636: neutralize the wizard's `module-permissions.xml` scaffold) in tension with the promoted B777 row ("permissions are inline in module.xml, not a separate file"). Verified on DashboardPan-ux: the SOURCE tree carries `module-permissions.xml` (`<permissions/>`), and the BUILT jar's `META-INF/module.xml` line 15 carries the same `<permissions/>` inlined by the gradle plugin; the jar contains no separate permissions file. Both statements are true at different levels; the kit row now states the source→artifact relationship. B777's "corrects B721" claim needs a §14 addendum in niagara-research (B721 described the source level correctly). [ev: DashboardPan-ux jar META-INF/module.xml:15; corpus B777]
