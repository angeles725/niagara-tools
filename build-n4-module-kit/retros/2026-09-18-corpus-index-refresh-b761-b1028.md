<!-- review-status: pending -->
# 2026-09-18 · kit · corpus-index-refresh-b761-b1028

**Session**: corpus-mining for build-n4-module (operator: index newer niagara-research docs for the kit).
**Delta count**: 26

## What happened

`corpus-index.md` mapped only B729–B760 in its P0/P1/P2 sections plus the Campaign-6 fold (B762–B791).
Since that fold, the niagara-research corpus grew four substantial module-authoring bodies that the kit has no
pointer to:

1. **B817** (module STRUCTURE STANDARD) — the lint checklist for how Tridium/Honeywell lay out an N4 module.
2. **B867–B932 (filtered)** — seven blocks from this range are genuine module-mechanics SPIs (type subscription,
   link direction, converters graph, driverUpgrade, dashboard-wb, uxBuilder, BComponentEvent catalog); the rest
   are driver wire-protocol internals and are NOT proposed here.
3. **B955–B961** — the first-party SDK example walkthroughs (envCtrlDriver rt/wb, typeExtensionDemo ux,
   componentLinks moduleTest, authClientExample, canonical scaffold, synthesis).
4. **B1015** (UmbrellaDashboard build) + **B1020–B1027** (N4 distribution: installer contract, overlays, OEM,
   dist payload, migration, shipped modules set, installer mechanics, SDK/security layout).
5. **docs/** — three synthesis guides written for the kit (how-to-create-an-n4-module.md, module-dev-workflow.md,
   module-best-practices.md) that are not indexed anywhere in corpus-index.md.

## Evidence

- B817: module structure standard lintable checklist `[ev: corpus B817]`
- B867: `BComponentSpace.subscribe(Type[], TypeSubscriber)` type-level subscription SPI `[ev: corpus B867]`
- B869: directional vs bidirectional communication (links vs writeback) `[ev: corpus B869]`
- B871: converters type-bridge graph, 104 classes, ~17 core value types `[ev: corpus B871]`
- B892: `driverUpgrade` Workbench per-driver upgrade SPI (`upgradeClass`, BOG rewrite) `[ev: corpus B892]`
- B894: `dashboard-wb` multi-target render agents (`@AgentOn` HX-bootstrap + PDF) `[ev: corpus B894]`
- B895: `uxBuilder` Px→bajaux embedding (embed JS widget in Px by ORD) `[ev: corpus B895]`
- B900: `BComponentEvent` id catalog, 21 named event ids `[ev: corpus B900]`
- B955–B957: envCtrlDriver SDK (rt+wb) + typeExtensionDemo SDK (ux) first-party source `[ev: corpus B955]` `[ev: corpus B956]` `[ev: corpus B957]`
- B958: componentLinks SDK example, BTestNg moduleTest framework (WSL-unsafe) `[ev: corpus B958]`
- B960: canonical N4 module build scaffold from SDK (Gradle, module.xml, Slotomatic hook) `[ev: corpus B960]`
- B961: SDK examples synthesis — what they teach + deltas for our modules `[ev: corpus B961]`
- B1015: UmbrellaDashboard build, facade + servlet + 3D SPA, green @4.14 `[ev: corpus B1015]`
- B1020–B1027: N4 Distribution ND1–ND8 (installer contract through SDK/security layout) `[ev: corpus B1020]` `[ev: corpus B1027]`
- `docs/how-to-create-an-n4-module.md`: confirmed at `/home/cristian/niagara-research/docs/how-to-create-an-n4-module.md` — master end-to-end guide from first principles to running JACE jar
- `docs/module-dev-workflow.md`: edit→build→sign→deploy→test runbook, every step citing a B-block
- `docs/module-best-practices.md`: evidence-grounded best-practices (B705+)
- corpus-index.md Campaign-6 section already covers B772–B791 and B817 is absent `[ev: kit corpus-index.md §Campaign-6]`

## Proposed kit deltas (propose-never-apply)

| Δ | Delta | Target file / § | Token |
|---|---|---|---|
| Δ1 | Add master guide P0 "start here" row (see fenced block A below) | `corpus-index.md` §P0 | `docs/how-to-create-an-n4-module.md` |
| Δ2 | Add B817 P0 row (see fenced block A below) | `corpus-index.md` §P0 | `[ev: corpus B817]` |
| Δ3 | Add B960 P0 row (see fenced block A below) | `corpus-index.md` §P0 | `[ev: corpus B960]` |
| Δ4 | Add docs/module-dev-workflow.md P1 row (see fenced block B below) | `corpus-index.md` §P1 | `docs/module-dev-workflow.md` |
| Δ5 | Add docs/module-best-practices.md P1 row (see fenced block B below) | `corpus-index.md` §P1 | `docs/module-best-practices.md` |
| Δ6 | Add B961 P1 row (see fenced block B below) | `corpus-index.md` §P1 | `[ev: corpus B961]` |
| Δ7 | Add B955 P1 row (see fenced block B below) | `corpus-index.md` §P1 | `[ev: corpus B955]` |
| Δ8 | Add B956 P1 row (see fenced block B below) | `corpus-index.md` §P1 | `[ev: corpus B956]` |
| Δ9 | Add B957 P1 row (see fenced block B below) | `corpus-index.md` §P1 | `[ev: corpus B957]` |
| Δ10 | Add B1015 P1 row (see fenced block B below) | `corpus-index.md` §P1 | `[ev: corpus B1015]` |
| Δ11 | Add B958 P2 row (see fenced block C below) | `corpus-index.md` §P2 | `[ev: corpus B958]` |
| Δ12 | Add B867 P2 row (see fenced block C below) | `corpus-index.md` §P2 | `[ev: corpus B867]` |
| Δ13 | Add B869 P2 row (see fenced block C below) | `corpus-index.md` §P2 | `[ev: corpus B869]` |
| Δ14 | Add B871 P2 row (see fenced block C below) | `corpus-index.md` §P2 | `[ev: corpus B871]` |
| Δ15 | Add B892 P2 row (see fenced block C below) | `corpus-index.md` §P2 | `[ev: corpus B892]` |
| Δ16 | Add B894 P2 row (see fenced block C below) | `corpus-index.md` §P2 | `[ev: corpus B894]` |
| Δ17 | Add B895 P2 row (see fenced block C below) | `corpus-index.md` §P2 | `[ev: corpus B895]` |
| Δ18 | Add B900 P2 row (see fenced block C below) | `corpus-index.md` §P2 | `[ev: corpus B900]` |
| Δ19 | Add B1020 P2 row (see fenced block D below) | `corpus-index.md` §P2 / new §distribution | `[ev: corpus B1020]` |
| Δ20 | Add B1021 P2 row (see fenced block D below) | `corpus-index.md` §P2 / new §distribution | `[ev: corpus B1021]` |
| Δ21 | Add B1022 P2 row (see fenced block D below) | `corpus-index.md` §P2 / new §distribution | `[ev: corpus B1022]` |
| Δ22 | Add B1023 P2 row (see fenced block D below) | `corpus-index.md` §P2 / new §distribution | `[ev: corpus B1023]` |
| Δ23 | Add B1024 P2 row (see fenced block D below) | `corpus-index.md` §P2 / new §distribution | `[ev: corpus B1024]` |
| Δ24 | Add B1025 P2 row (see fenced block D below) | `corpus-index.md` §P2 / new §distribution | `[ev: corpus B1025]` |
| Δ25 | Add B1026 P2 row (see fenced block D below) | `corpus-index.md` §P2 / new §distribution | `[ev: corpus B1026]` |
| Δ26 | Add B1027 P2 row (see fenced block D below) | `corpus-index.md` §P2 / new §distribution | `[ev: corpus B1027]` |

> **Kit-file text deltas (not corpus-index rows — fold separately):**
> - `README.md` §Layout: add one line → `- \`docs/how-to-create-an-n4-module.md\` in the niagara-research repo — the master end-to-end guide; read BEFORE any B-block.`
> - `METHODOLOGY.md` §P0 reference line: change `> Background reading, by layer and priority: **\`corpus-index.md\`** …` to append ` · master guide → \`docs/how-to-create-an-n4-module.md\` in niagara-research`

---

### Fenced block A — P0 rows to ADD (copy-paste under the existing P0 table)

```markdown
| **`docs/how-to-create-an-n4-module.md`** | **START HERE** — master end-to-end guide: scaffold → deps → slots → sign → test → deploy. Written from first principles; every step traces to a B-block. Lives in `niagara-research/docs/`. | reference |
| **B817** | The N4 module STRUCTURE STANDARD — how Tridium/Honeywell lay out rt/ux/wb/doc profiles; a lintable conformance checklist for our four modules | organization |
| **B960** | The canonical N4 module build scaffold from the first-party SDK examples — the Gradle layout, module.xml header, signing wiring, and Slotomatic hook as Tridium ships them | build / scaffold |
```

---

### Fenced block B — P1 rows to ADD (copy-paste under the existing P1 table)

```markdown
| **`docs/module-dev-workflow.md`** | Module dev runbook — the edit→build→sign→deploy→test loop with exact tool steps; every step cites the relevant B-block. Lives in `niagara-research/docs/`. | reference |
| **`docs/module-best-practices.md`** | Evidence-grounded best-practices guide — rules distilled from reference modules and corpus audits (B705+). Lives in `niagara-research/docs/`. | reference |
| **B961** | SDK dev-examples SYNTHESIS — what the four SDK examples collectively teach + the deltas for our module work; read after B960 | reference |
| **B955** | envCtrlDriver SDK example (rt) — a complete minimal N4 field driver, first-party source; the rt SPI contract in a real build | rt / exemplar |
| **B956** | envCtrlDriver SDK example (wb) — the Workbench device/point manager UI for the SDK driver exemplar | wb / exemplar |
| **B957** | typeExtensionDemo SDK example (ux) — a browser type-extension with JS build + Jasmine tests; the ux SPI patterns in a real build | ux / exemplar |
| **B1015** | Building `UmbrellaDashboard` — a dashboard-type N4 module (facade + servlet + 3D SPA), rt+ux built green @4.14; the DashboardPan exemplar in a second module | reference / dashboard |
```

---

### Fenced block C — P2 rows to ADD (copy-paste under the existing P2 table)

```markdown
| **B958** | componentLinks SDK example — link-lifecycle callbacks + the `BTestNg` moduleTest framework; the correct seam for station-level tests (WSL-unsafe, documents the boundary) | test |
| **B867** | Server-side TYPE-LEVEL component subscription (`BComponentSpace.subscribe(Type[], TypeSubscriber)`) — the SPI for observing every live instance of a type from a service | rt / service |
| **B869** | Directional vs bidirectional communication in N4 — links, writeback, and "who respects whom"; pick the correct direction before wiring a facade→rt slot | rt |
| **B871** | The `converters` type-bridge graph — 104 converter classes over ~17 core value types; read when two heterogeneous slot types need to wire across a link | rt / build |
| **B892** | `driverUpgrade` — the WB per-driver component-upgrade SPI (`upgradeClass`, BOG rewrite); relevant when providing a WB migration path on a schema change | wb / build |
| **B894** | `dashboard-wb` — the multi-target render agents for a `DashboardPane` (`@AgentOn` HX-bootstrap + PDF-paint via one model) | wb |
| **B895** | `uxBuilder` — rendering Workbench Px as browser bajaux widgets (embed a JS widget in Px by ORD; the agent-filter gate + the Px-serving servlet) | ux |
| **B900** | The `BComponentEvent` id catalog — 21 named component-event ids (0–20) + 4 reserved slots; read before writing a `Subscriber.event()` handler | rt |
```

---

### Fenced block D — Distribution P2 rows to ADD (copy-paste; suggest a new `## P2 — distribution` sub-heading)

```markdown
## P2 — distribution (read when shipping a module or targeting a specific N4 release)

| Block | What it gives the builder | Layer |
|---|---|---|
| **B1020** | N4 Distribution ND1 — the Installer Contract (`install.properties`, `version.properties`, EULA/licensing): what the customer's installer reads when deploying your module | distribution |
| **B1021** | N4 Distribution ND2 — shipped default config (`nre.properties`, `system.properties`, `units.xml`) vs installed values; how OEM defaults layer over stock N4 (refines B31) | distribution |
| **B1022** | N4 Distribution ND3 — Honeywell OEM overlay anatomy over stock 4.13.2.18 (branding, versioning, overlay extras); understand what an OEM bundle adds vs the stock install | distribution |
| **B1023** | N4 Distribution ND4 — the dist/ payload tree: JRE bundles, framework versions, Supervisor-as-installer mode | distribution |
| **B1024** | N4 Distribution ND5 — install-time transfer & migration (`conversion/`, AX→N4 path, `cleanDist`, station templates) | distribution |
| **B1025** | N4 Distribution ND6 — the shipped modules set: 721 jars, the 4.13.2.18.5 manifest, OEM-vs-stock families (vs the installed N4.14 corpus) | distribution |
| **B1026** | N4 Distribution ND7 — Installer mechanics: `Installer_x64.exe`, overlay-merge sequence, `niagarad`/Service setup | distribution |
| **B1027** | N4 Distribution ND8 — SDK dev examples (`dev/`) and default security material (`overlay/security/`) — file layout and what is shipped | distribution |
```

---

## Lessons

- Corpus-index.md had no pointer to the three synthesis docs in `niagara-research/docs/` — these are the most
  actionable entry points for a new builder and belong in P0/P1, not buried in the docs tree.
- B817 (module structure standard) is the lintable conformance skeleton for any N4 module; it belongs in P0
  alongside the verified-scaffold B790 and the new canonical-SDK B960.
- The B867–B932 range is mostly driver wire-protocol internals — only 7 of 66 blocks are relevant to module
  building; the remainder are NOT proposed to avoid index noise.
- B960 (SDK scaffold) and B961 (synthesis) are stronger P0/P1 anchors than the decompiled exemplars because
  they carry first-party, compile-ready source with signed-jar semantics intact.
- Distribution blocks (B1020–B1027) belong in a separate sub-heading so builders can opt in only when
  shipping or targeting a specific N4 OEM install.

---
**Status**: PENDING — INDEX row appended: `| 2026-09-18-corpus-index-refresh-b761-b1028.md | kit | 2026-09-18 | pending | 26 |`
