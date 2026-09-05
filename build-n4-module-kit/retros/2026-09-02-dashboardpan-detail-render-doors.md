<!-- review-status: folded -->

# Retro — DashboardPan detail-view render + door alarm — PROPOSED kit deltas

Status: **propose-never-apply**. A human folds these into `types/dashboard.md`.
Context: after the main-plano SVG fix (see `2026-09-01-dashboardpan-ux-direct-build.md`), the per-room DETAIL view still rendered wrong on the panel (fans off the evaporator, points/labels displaced, zone labels clipped), and a per-room door alarm was added. Frontend-only iterations verified in the `/hmi` simulator via headless Chrome, then Java-8 + slotomatic + jar, gate ALL PASS.

## Panel WebView reality (seeds for `types/dashboard.md`)

1. **The Honeywell WEB-HMI10 (i.MX8M) WebView does NOT honor CSS `aspect-ratio`, and letterboxes `<img object-fit:contain>` differently than desktop Chromium.** A layout whose height depends on `aspect-ratio`, or whose vector overlay is JS-fitted to an `object-fit:contain` rect, diverges on the panel. Diagnostic tell: it looks right in the desktop `/hmi` simulator and wrong only on the panel. Design so nothing critical depends on `aspect-ratio` or `object-fit` — use SVG intrinsic sizing + `preserveAspectRatio` instead (both universally supported, including old WebKit that renders SVG shapes but may not do CSS `aspect-ratio`).

2. **Overlay raster + vectors as ONE inline SVG, never a raster `<img>` plus a separately-sized overlay.** The bug: the room photo was an `<img>` (or a first-cut `<svg><image>` in a *different* svg than the fans/leaders), and the fan/leader vectors lived in separate SVGs sized by JS — the two layers diverge on the panel, worst for near-square room images (they letterbox most). Fix: put `<image xlink:href>` + the fan `<g>` + the leader `<g>` in ONE `<svg viewBox="0 0 w h" preserveAspectRatio="xMidYMid meet">`. Now image and vectors share one coordinate system and letterbox as a unit — impossible to misalign. This is the same fix as the main plano; the detail view is just the plano pattern applied per room.

3. **`<image>` inside SVG needs `xlink:href` (+ `xmlns:xlink`), not SVG2 `href`, on old WebKit.** SVG2 `href` renders blank on the i.MX8M; `xlink:href` works on old and modern. Set it in JS with `setAttributeNS("http://www.w3.org/1999/xlink","href",…)`.

4. **HTML overlay elements (point markers, label cards) must be sized to the SVG image's REAL rendered rect, not to the frame.** When the labels/points stay as HTML `<div>`s (richer than SVG text) over the one-SVG raster, size their container with `getBoundingClientRect()` of the SVG `<image>` element (offset relative to the frame), NOT the frame's contain-rect — otherwise the HTML layer diverges from the in-SVG image+vectors. One `ajustarBoxDetalle()` that reads the image rect fixed points+leaders alignment that a frame-based `ajustarCaja` broke.

5. **Contain the detail image with `height:100%` + SVG `meet`, not `height:auto`, or overflow clips bottom labels.** A near-square room image sized `width:100%;height:auto` overflows a shorter frame; `overflow:hidden` then clips the zone label that sits near the bottom. Give the frame a definite height (`height:100%` from the stage) and let the SVG `meet` letterbox inside it → image fully visible, no clip, all labels shown. Verified across wide (C3) and square (C2/C4) rooms.

6. **Verify panel-path rendering in the simulator by forcing the intrinsic path.** Setting `#frame{aspect-ratio:auto}` makes desktop Chromium size by the SVG's intrinsic viewBox — the SAME path the panel uses — so a headless `/hmi` screenshot then reflects panel behavior, not a desktop-only render. Pixel-analysis of the raster (e.g. red-fan centroid) confirms a data coordinate matches the render, separating "data wrong" from "render wrong".

## Feature pattern — per-room boolean alarm (seeds for `types/dashboard.md`)

7. **A per-room link-in boolean (e.g. door-open) is a small full-stack slice, done generically:** one `boolean` slot on the facade `BRoomPanel` (`SUMMARY`, link-in from the physical contact — NOT `OPERATOR`, it is an input not config), one entry in the reader's boolean-slot loop (emits `CuartoN/<slot>`), and a frontend read keyed generically as `"Cuarto"+id+"/"+slot`. The zone polygon force-reds via a CSS class that overrides the temperature-state color; the room card shows the alert text. Start scoped to the room(s) that need it, but keep the slot on every room (cheap, uniform) so the frontend stays generic.

## Field / ops (seeds for BUILD-LOOP)

8. **After deploying a new module to the panel, power-cycle it — the WebView caches the old page.** A "blank after redeploy" on the panel is often stale WebView state, not a code bug. Operator field note (verified): disconnect/reconnect the panel when loading a new screen. Check the power-cycle before diagnosing code.

9. **Keep `src/rc/` free of editor/backup files.** `*.bak`/`.orig`/scratch under `rc/` get packaged by `processResources` (they ship + are servable + bloat the jar ~2-5MB). Remove them; likewise an orphan `img/plano.png` when the real image is base64-embedded. A `verify-module.sh` WARN on non-asset files under `rc/` would catch this.
