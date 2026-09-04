# Retro (PROPOSED delta — propose-never-apply) — editing the base64-heavy dashboard SPA, and the preview → screenshot → PDF report pipeline

- **Date**: 2026-09-04
- **Origin**: added a new "Condensadoras" tab + per-compressor HOA control to `DashboardPan-ux/src/rc/index.html`, previewed it in the HMI simulator, screenshotted it for a remote operator, and then produced a client report (web artifact + PDF). Several tooling frictions worth pinning so the next `-ux` change doesn't rediscover them.
- **Status**: PROPOSED. Tooling/workflow notes for dashboard (`-ux`) work and deliverables. Adds no rule.

---

## Finding 1 — the SPA `index.html` is too big for Read/Edit; edit it with anchored Python

`DashboardPan-ux/src/rc/index.html` embeds ~1 MB of base64 images, so the `Read` tool fails ("File content exceeds maximum tokens") **even with `offset`/`limit`** (a single line can be huge), and `Edit` refuses because the file was never "read". Two things that work:
- **Read a region** with `sed -n 'A,Bp' index.html` (justified fallback — the dedicated tool cannot).
- **Edit** with a small Python `str.replace()`, asserting `s.count(anchor) == 1` before each replace and writing the whole file back. All-or-nothing: assert first, write last. After JS edits, validate the whole `<script>` with `node --check` on the extracted script (`node --check` passed = syntax OK; it does not run the DOM).

## Finding 2 — the tab/data patterns to reuse (verified this session)

- **Add a top-level tab**: one `<div class="nav-item" data-page="X">` + one `<section id="page-X" class="page">`. `showPage()` toggles by `id === "page-"+name` — no JS change. Moving a tab = move only the nav-item (page order is by id, not DOM order).
- **Read-only data**: render from `lastEq["<ord>"]` (the last `/api/equipment` JSON, points shaped `{v, st, nm}`); the Reader emits `Cuarto<N>/<slot>`-style keys.
- **HOA control**: `hoaRow("Label", "<panel>/<slot>Mode", null, {stateOrd:"<panel>/<slot>State"})` inside `buildHoa()`; the segment selector `ctrlSeg` is built from `CUARTOS` — append a non-room segment (e.g. `dataset.room="COND"`) and branch on it in `buildHoa()`.

## Finding 3 — preview → screenshot → PDF, for a remote operator

The operator is often off-site, so a `localhost` URL alone is useless — deliver a screenshot / PDF.
- **Preview**: `tools/dashboard-preview.py --rc src/rc --prefix /dashboardpan --mock <file.json> --port <p>` serves `rc/` live (no build/sign/deploy) and mocks `/api/*` with the JSON. `/hmi` is the 1280×800 Honeywell-panel simulator. The mock JSON uses the same `{v, st, nm}` point shape — seed it with realistic values (from the live oBIX read) so the design shows populated.
- **Screenshot with navigation**: a headless Chromium binary ships in the Playwright cache (`~/.cache/ms-playwright/chromium-*/chrome-linux64/chrome`); the `playwright` npm module lives under `~/.npm/_npx/*/node_modules` (drive it with `NODE_PATH=<that> node -e '...'`). Chromium's bare `--screenshot` can't click a tab, so a tiny Playwright script (`page.click('[data-page="X"]')` then `screenshot`) is the only way to capture a non-default tab. Send with SendUserFile.
- **Client report → PDF**: author one HTML (ground it in the dashboard's real identity — sage/paper palette `--sage #6c715d`, `Prata`+`Roboto`), embed images as base64 data URIs (PIL: downscale big phone photos to ~780 px JPEG q85; keep UI screenshots PNG ≤1120 px) so it's a single self-contained file. Publish as an Artifact for the web copy; for the PDF, add a `@media print`/`@page A4` block (force white ground, `break-inside:avoid` on figures/cards, `print-color-adjust:exact`) then `chrome --headless --print-to-pdf --no-pdf-header-footer`. ~19 A4 pages / ~3 MB for this report.

## Why it matters

`-ux` iteration is UI-first (operator's rule: UI before compile), so a fast preview/screenshot loop with no build is the whole point — and the deliverable a client actually keeps is a PDF, not a localhost link. Pin the binary/module locations so the next session doesn't re-hunt them.
