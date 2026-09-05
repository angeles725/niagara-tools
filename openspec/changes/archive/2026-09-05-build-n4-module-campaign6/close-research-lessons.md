# Campaign 6 — research-lane lessons (for the close retro)

**Author**: investigador1 (Opus 4.8). Kit-side lessons from the three research lanes (module-authoring-exemplars
B772–B785, own-modules-vs-exemplars B786–B790, module-web-tier-exemplars B791) + the PR7 fold. One line + evidence each.

## What the research produced (method + content lessons)
- **Compress before you enumerate: 5 SPIs collapse to ONE idiom.** "Subclass a framework base + register a `<type>`/
  agent in module.xml + hand back a self-describing SPI object; no central registry the author edits" covers services,
  ORD schemes, query/search/index providers, and rdb dialects — with analytics (register-by-`<type>`, no `@AgentOn`)
  as the ONE documented exception. Teach the idiom once, the surfaces as instances. [ev: B778/B782/B785/B777 + B773 exception]
- **A conformance rule is worth a lint ONLY if it is statically decidable; otherwise it is a human-review item.** Hard-fail
  on lexicon dup-bare-keys / a `Clock.Ticket` field with no `stopped()`-cancel / an empty palette on a component module /
  coverage-%. Keep operator-vs-admin intent, container order-sensitivity, and poll-vs-subscribe as review. [ev: B787-789;
  OMV1's NEGATIVE — a "non-HIDDEN action without OPERATOR → FAIL" lint false-positives on every legit admin action]
- **A decompiled-corpus claim describes the ARTIFACT; verify the SOURCE shape by BUILDING before it becomes a recipe.**
  Two reconciles hit this: B777 "permissions inline, corrects B721" was artifact-only (the plugin inlines the source
  `module-permissions.xml`'s `<permissions>` element — one file, two child kinds); B790's minimal template needed SIX
  build-forced fixes (module-include.xml not module.xml, module.lexicon source name, `<MOD>-rt.gradle.kts`, hand-written
  `do<Action>()`, //region markers, auto preferredSymbol). [ev: B777 §14 addendum + B636/CompPan-rt jar; B790 §14 + B793 build GREEN]
- **A second reader's grep hit is not a correction until the enclosing METHOD is read.** I mis-flagged B772's
  executeExtensions cite by grabbing the first of 7 `instanceof BPointExtension` sites (`:342` = an array-fill, not the
  `:385/:391/:396` execution loop); retracted. grep-verify still caught 8+ real seed/sweep errors (BAbstractPointExt
  absent, BComponentList absent, reorder IS used, no 2s poll, no baja BTimer, CompPan lexicon NOT empty). [ev: B772 companero catch; B779/B775/B788]
- **A clean audit dimension is a one-line verdict, not a block.** OMV1 (actions) + OMV6 (background-work) conformed → no
  block; the web-tier thread audited THIN → 1 index block, 0 investigation blocks. No-padding is the discipline. [ev: OMV1/OMV6/B791]

## What made the concurrent lanes safe (process lessons)
- **Numbering authority + reserved ranges + a worktree per lane keep concurrent writers off each other.** One session
  owns block-number arbitration; lanes reserve contiguous ranges (B762-771 / B772-791) and holes are tolerated, never
  refilled; long loops move to a sibling worktree so the shared checkout has one writer. [ev: the census two-lane split; B793 fresh worktree]
- **Last-pusher recomputes the envelope counters BY HAND; never `--sync-state` under shared-global.** A concurrent tool
  that writes the corpus total into a focus envelope is noise; edit only your own rows, pull --rebase before every push.
- **The corpus/ledger inventory is the authority for existence + count — INDEX/memory drift.** FOCUSES.md said kitControl
  "planned (0/12)" while its RESEARCH-STATE said stopped/investigable=0; the stale row would have cost a full re-loop.
  A sweep should flag FOCUSES-vs-RESEARCH-STATE drift. [ev: the kitControl fix, commit bbc2f7968]

## What the research lanes did NOT produce (honest gaps)
- No heartbeat/liveness watchdog exemplar exists in our modules (all monitors are threshold-only); MAE7-G1 (type-level
  `BComponentSpace.subscribe`), MAE1-G1 (live point-extension registration), and B793-G1 (deploy+boot on a station) all
  stay requires-execution — the read-only lanes proved the shapes, not the live boot. [ev: B775/B778/B772/B793 open gaps]
