<!-- review-status: pending -->
# 2026-09-17 · UmbrellaDashboard · dashboard-status-state-modeling

**Session**: `/build-n4-module` — after deploy, the operator asked how the dashboard knows normal/paro/reserva/alarma and whether fault/override on a link-mark is reflected.
**Delta count**: 4

## What happened
UmbrellaDashboard deployed to station PRUEBAS showed DEMO data (client-side sim) by default and had no real
notion of equipment operating state — only a coarse normal/offline/alarma. The operator's questions exposed
two distinct, unmodeled concepts: equipment OPERATING STATE (normal/paro/reserva/alarma, from control logic)
vs point HEALTH (fault/down/stale/overridden, the BStatus on each linked value). Fixed: default→live, added a
`unitState` rt slot for operating state, carried per-point `st` into the SPA, and added a B229-style per-value
health badge. Also fixed a lexicon mojibake (accents). All rebuilt green and verified live with a mock.

## Evidence
- Operating-state vs point-health model + fix `[ev: corpus B1017]`; UmbrellaDashboard build `[ev: corpus B1015]`.
- Reader `{v,st}` precedence `fault>down>stale>disabled>null>alarm>overridden>ok` `[ev: corpus B752 DashboardReader.java:321-332]`.
- BStatus 8-bit model (badge colors) `[ev: corpus B755 §755.2]`; facade slots are BStatusNumeric properties `[ev: corpus B732]`; status overlays `[ev: corpus B229]`.
- Lexicon mojibake `PresiÃ³n`/`SecciÃ³n` in Workbench property sheet → Niagara reads the lexicon as Latin-1 `[ev: corpus B1015 §1015.5]`.
- Rebuild `verify-module: 17 passed / 0 failed → ALL PASS`, baja 4.14; live mock preview showed states + badges.

## Proposed kit deltas (propose-never-apply)
| Δ | Delta | Target file / § | Token |
|---|---|---|---|
| Δ1 | document the OPERATING-STATE vs POINT-HEALTH split: add an operating-state slot (`BStatusEnum`/ordinal `unitState`, linked to the control module's state point) — a sensor's BStatus can NEVER yield normal/paro/reserva; point health rides free on each `BStatusNumeric` slot | `types/dashboard.md` § facade contract | `[ev: corpus B1017]` |
| Δ2 | add the SPA per-value HEALTH badge recipe: carry `st` per point in the live reading, render a status dot colored by BStatus (fault=amber, override=teal, down/stale=grey, alarm=red; B229/B755/B218), tooltip; elevate unit to alarma on any fault/down/stale/overridden | `types/dashboard.md` § "self-contained 3D SPA live-wiring" | `[ev: corpus B1017]` |
| Δ3 | **module.lexicon must be ASCII (no accents)** — Niagara reads the lexicon as Latin-1, so UTF-8 accents render as mojibake (`Presión`→`PresiÃ³n`); use plain ASCII (`Presion`) or `\uXXXX` escapes | `types/structure.md` § lexicon + `lint-structure.sh` (add a non-ASCII-in-lexicon check) | `[ev: corpus B1015 §1015.5]` |
| Δ4 | add a `lint-lexicon-ascii` check (or extend lint-structure L4) that FAILs on any non-ASCII byte in a `.lexicon` file, since it silently mojibakes on-station | `toolbelt/lint-structure.sh` | `[ev: corpus B1017]` |

## Lessons
- Operating state and point health are two different data with two different sources — model each explicitly; never fake operating state from a sensor's BStatus.
- A facade `BStatusNumeric` PROPERTY carries the linked point's BStatus for free (fault/override/stale) — the reader's `st` tag is all you need to reflect health; don't drop it in the SPA live path.
- `module.lexicon` is Latin-1 on-station: ASCII only, or accents mojibake in the property sheet.
- A deployed module should default to LIVE, not demo — demo data on a real station reads as fake data to the operator.
- The source (demo/live) toggle belongs as an rt config slot, not a browser control on a nav-hidden page (operator's point; open follow-up).

---
**Status**: PENDING — INDEX row appended: `| 2026-09-17-dashboard-status-state-modeling.md | UmbrellaDashboard | 2026-09-17 | pending | 4 |`
