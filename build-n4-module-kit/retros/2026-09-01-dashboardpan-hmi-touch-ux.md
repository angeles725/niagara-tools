<!-- review-status: folded v0.2 · 2026-09-01 -->

# Retro — DashboardPan HMI touch UX + chart (2026-09-01) — PROPOSED kit deltas

Status: **propose-never-apply**. A human folds these into `types/dashboard.md` / `METHODOLOGY.md`.
Context: second half of the DashboardPan session — after the 3→5-room extension (see `2026-09-01-dashboardpan-5rooms.md`), the operator drove a round of live UX fixes against the real deploy target, a **Honeywell WEB-HMI10/CF** (10.1" TFT, 1280×800, projected-capacitive multi-touch, i.MX8M Mini). All iterated on `dashboard-preview.py /hmi` and **approved by the operator before each compile**. Everything below is frontend (`-ux`, `src/rc/index.html`); the facade/control `-rt` were untouched this round.

## What this PROVED (seeds for `types/dashboard.md`)

1. **On a FIXED touch panel, a config panel that overflows the viewport must SPLIT INTO SUB-TABS — not scroll, not shrink to more columns.** The kiosk is 1280×800 with no ergonomic scroll (capacitive, wall-mounted). Cuarto 3 grew to 15 config fields (9 common + 6 defrost) = 8 rows in 2 columns → clipped by `overflow:hidden`. Three fixes were tried in order and only the last survived operator review:
   - `overflow-y:auto` (scroll) — REJECTED: "no es ergonómico" on a fixed touch panel.
   - 3-column grid (15 fields → 5 rows) — REJECTED: touch targets got too small for a 10" panel.
   - **Sub-tabs inside the panel (General / Defrost)** — ACCEPTED: keeps 2 big columns, each group fits with no scroll, one "Guardar cambios" saves both. RULE for `types/dashboard.md`: when a room's field count exceeds what fits in 2 columns at ≥44px targets, partition the fields into named sub-tab GROUPS, not more columns or scroll. Data-drive it: give the room a `groups:[{name,fields}]` and render a `.cfg-tabs` bar; keep ONE shared `items[]` + one save so the operator edits across tabs and saves once.

2. **Extract the per-field builder before you need two of it.** Splitting into groups meant building the same stepper/toggle row into N grids. Extracting `buildField(room, f, items) → row` (registers `spInputs[ord]`, pushes to `items`, returns the row) made the group loop trivial and kept save/prefill unchanged. RULE: a dashboard config panel's row-builder should be a standalone function from the start — grouping, reordering, and conditional fields all become cheap.

3. **A "missing" config control on the live HMI is almost always a STALE deployed `-ux`, not a code bug.** The operator reported `terminateOnResistanceTemp` (and earlier `coolOnSensorFault`) "not in the interface." Both render correctly in current source AND in the built jar (verified: `unzip -p …-ux.jar rc/index.html | grep -c`). The panel was just clipping them (lesson 1) AND the JACE ran an older `-ux`. RULE: before debugging a missing UI element, (a) confirm it renders in the CURRENT `src/rc` on `/hmi`, (b) `unzip -p` the deployable jar and grep for it, (c) THEN suspect the deployed build. Do not touch code until the live-vs-source gap is ruled out.

4. **Writable config RANGES live in the frontend clamp, not in Java.** "Solo puedo escribir hasta -40" was a `SP_COMMON` field `min:-40`, not a `@NiagaraProperty` facet or a servlet clamp. Verified the facade `setpoint` (facet = UNITS only) and control `BColdRoom.setpoint` (`facets = null`) impose no MIN → the link never re-clamps. RULE: setpoint/limit min·max·step·decimals for a dashboard live in the SPA's field descriptor array (`SP_COMMON`/`SP_DEFROST`), and that is the ONLY place to change an operator-writable range unless a Java facet also constrains it — so check the facade/control facets too before concluding.

## What this PROVED about charts on an HMI (seeds for `types/dashboard.md` — chart section)

5. **`preserveAspectRatio="none"` on a fixed-viewBox SVG chart DISTORTS on a real panel — measure the box and set the viewBox to real px (1 unit = 1 px).** The chart used `viewBox="0 0 900 340" preserveAspectRatio="none"` stretched into a responsive box → non-uniform scaling of strokes and axis text ("no se ve bien"). Fix: at the top of `renderChart`, read `chart.getBoundingClientRect()` and set `viewBox = "0 0 <w> <h>"` to the actual pixel size (guarded for width>60). Now aspect matches, `none` no longer distorts, and font-size px are true. RULE: an SVG chart that must fill a responsive/fixed container should size its viewBox to the measured box each render, not carry a fixed art-board viewBox.

6. **`stroke:currentColor` on every series = every line the same inherited color.** Nothing set per-series `color`, so all polylines drew in the inherited ink — indistinguishable when overlaid. Fix: a `SERIE_COL` palette assigned via `el.serie[i].style.color` (and the legend chip), and visible series get `.on` (opacity 1, wider stroke) — previously visible series stayed at the faint `.28` because `.on` was only toggled on hover, never on filter. RULE: a multi-series chart must assign a distinct color per series explicitly and mark the currently-shown series visually bold; do not rely on inherited `currentColor` or a hover-only emphasis class.

7. **On a CAPACITIVE HMI the chart read-out is a TOUCH crosshair (pointer events + `touch-action:none`), never hover.** Added a tap/drag crosshair: `pointerdown/move` map the touch X back to a reading index (inverse of the X scale stored in a module-level `CS`), draw a vertical line + per-series dots + a tooltip box with the reading's **time** and, per visible series, its **friendly name (`equipo`) + value**. `touch-action:none` on the SVG stops the panel from hijacking the drag as a scroll; double-tap clears. RULE: any hover-only affordance (tooltip, highlight) is dead on a touch HMI — implement it with pointer events and disable native touch gestures on the interactive element. And label points by the operator's name (`equipo`), never the raw tag (`TE-11`).

## Tooling lesson (seeds for `METHODOLOGY.md` — working with `-ux` sources)

8. **A dashboard `index.html` with embedded base64 art defeats the Read tool — navigate with `sed` ranges + a line-length filter.** DashboardPan's `index.html` embeds room photos/planos as base64 data URIs on single multi-MB lines; `Read` blows the token budget on the whole file, and a naive `grep -n` dumps the mega-lines. Reliable navigation: `sed -n 'A,Bp' file | cut -c1-160` for a known range, and to search skip the giant lines with `awk 'length<300'` or a tiny python `for i,l in enumerate(...): if len(l)<300 and pat.search(l): print(i,l)`. Always re-verify with `node --check` on the extracted `<script>` blocks after editing. RULE: for any asset-laden single-file artifact (dashboard SPA, minified bundle), reach for range-scoped `sed`/line-length-filtered search from the start — do not try to Read the whole file.

## Already covered (do NOT re-add)
- Preview-before-compile gate (`dashboard-preview.py /hmi`) — kit already mandates it; this session re-confirmed its value (it caught the 3-column-too-small BEFORE a wasted build). Keep as-is.
- STORED repackage for Workbench signing, Java-8 + slotomatic build, bytecode-52 verify — covered in `2026-09-01-dashboardpan-5rooms.md` and `build-verify.md`.
- Data-driven room count (splice DATA not CODE) — covered in the 5-rooms retro.

## Proposed delta table (machine-countable)

| # | Proposed change | Target (file · section) | Evidence | Priority |
|---|---|---|---|---|
| 1 | Overflowing touch-config → sub-tab GROUPS (not scroll/columns); `groups:[{name,fields}]` + shared items/one-save | `types/dashboard.md` (config panel) | index.html `SP_ROOMS` Cuarto3 groups; operator rejected scroll + 3-col | HIGH |
| 2 | Extract `buildField(room,f,items)` as the row-builder from the start | `types/dashboard.md` | refactor enabling groups | MEDIUM |
| 3 | "Missing UI element" triage: source `/hmi` → `unzip -p` jar → suspect deploy, before code | `types/dashboard.md` / `METHODOLOGY.md` | `terminateOnResistanceTemp`/`coolOnSensorFault` stale-`-ux` incident | HIGH |
| 4 | Writable ranges live in SPA field descriptors (min/max/step); check facade+control facets too | `types/dashboard.md` | setpoint `-40` clamp incident | MEDIUM |
| 5 | SVG chart: size viewBox to measured box px each render (kill `preserveAspectRatio` distortion) | `types/dashboard.md` (chart) | `renderChart` box-measure fix | HIGH |
| 6 | Multi-series chart: explicit per-series palette + bold the shown series (not inherited currentColor / hover-only) | `types/dashboard.md` (chart) | `SERIE_COL` + `.on`-on-visible fix | MEDIUM |
| 7 | Touch chart read-out = pointer-event crosshair + `touch-action:none`; label by `equipo` not tag | `types/dashboard.md` (chart) | touch tooltip on WEB-HMI10 | HIGH |
| 8 | Asset-laden single-file `-ux`: navigate with `sed` ranges + `awk 'length<N'`/python line-filter, not Read | `METHODOLOGY.md` | base64 index.html defeated Read | MEDIUM |
| 9 | Plano overlay alignment: the image, `IMG_W/IMG_H`, the zones `viewBox`, AND `.frame aspect-ratio` must ALL match the plano image — swapping the image without updating all four desfases the polygons | `types/dashboard.md` (plano) | 3→5 rooms swapped to a 1248×891 render but `.frame` stayed `1247/771` → offset | HIGH |
| 10 | `#plano` src can be an inline base64 (not `img/plano.png`); read the REAL src before trusting a file — the on-disk `plano.png` may be orphaned | `types/dashboard.md` / `METHODOLOGY.md` | grabbed the wrong 1247×771 file; real image was 1248×891 base64 | MEDIUM |
| 11 | When a prior version renders correctly on the panel, diff its plano config against the broken one — same technique, hunt the one inconsistent value | `METHODOLOGY.md` | REF `DashboardPan-Leon` (3 rooms, all 1247×771 consistent) isolated the stray `.frame` aspect | MEDIUM |

## Plano-alignment addendum (seeds for `types/dashboard.md` — plano section)

The zone-polygon overlay is drawn as **percentages** of `IMG_W/IMG_H` into an SVG whose `viewBox` equals the plano image, and `ajustarCaja()` sizes the overlay box to the image's real rectangle at runtime (`getBoundingClientRect` + `contain`). This is **resolution-independent** — it is NOT tuned to 1280×800; it self-fits any panel. It aligns **only when four values agree with the plano image**: (1) the `#plano` image itself, (2) `IMG_W/IMG_H`, (3) the zones `viewBox`, (4) `.frame aspect-ratio`. A proven-good reference version (`Downloads/DashboardPan-Leon`, 3 rooms) had all four at `1247×771`; the current 5-room build correctly moved image + `IMG_W/IMG_H` + `viewBox` to a new `1248×891` render but left `.frame aspect-ratio:1247/771` — that single stray value stretched the overlay off the image. **RULE:** treat the four values as one atomic unit; changing the plano image means updating all four together. Belt-and-suspenders: call `ajustarCaja` on every path that reveals the plano (init, `ResizeObserver`, sub-view show, page show, `window.load`, `img.onload`, back-to-plano) so an unknown WebView's layout timing (e.g. the HMI's i.MX8M Chromium) can never leave it unfitted.
