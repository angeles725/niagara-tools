<!-- review-status: folded -->
# 2026-09-18 · kit · palette-lexicon-assets-images-deltas

**Session**: corpus-mining for build-n4-module (operator: generate all possible kit deltas).
**Delta count**: 5

## What happened

Corpus-mined B746 (palette BOG XML authoring + pre-wired templates), B759 (lexicon/i18n + -doc/help
profile), B738 (icon/SVG placement), and B1015 (UmbrellaDashboard 3D SPA packaging) for kit gaps in
the palette, lexicon, icon, and asset-packaging veins. Cross-checked against existing kit toolbelt
(verify-module.sh, slot-coverage.sh, report-module.sh, lint-lexicon-ascii.sh), types docs, and
METHODOLOGY.md. Five distinct, non-duplicate deltas found.

## Evidence

- `check_palette` in `verify-module.sh:377` excludes `b:Folder` root entries from count but never
  WARNs that the root type is non-idiomatic — the idiomatic ungated root is `b:UnrestrictedFolder`
  per Tridium conventions; `b:Folder` is gated (requires permission to expand). `[ev: corpus B746 §746.1]`
- B746 §746.3 shows that palette `<p>` entries can nest child `<p>` elements with property overrides to
  form pre-wired commissioning-ready assembly templates; no kit recipe documents this technique, and the
  only palette guidance in `types/structure.md` (lines 34/38) is about non-empty entry count, not
  template authoring. `[ev: corpus B746 §746.3]`
- `METHODOLOGY.md:41` and `types/structure.md:9` both note `-doc` as a "SEPARATE `runtimeProfile=doc`
  module, NEVER a part of a code module" but give no authoring recipe — no `toc.xml` (JavaHelp 1.0 TOC
  DTD), no Guide-on-Target HTML naming (`doc/<mod>-<TypeName>.html`), no `help.guide.base` lexicon key.
  B759 §759.6 provides the complete recipe including two naming conventions and the two build plugins
  (`niagara-doc` + `bajadoc`). `[ev: corpus B759 §759.6]`
- `METHODOLOGY.md:16` has a partial icon recipe — "a 16×16 PNG in `src/rc/` + `getIcon()` returning
  `BIcon.make(BOrd.make("module://<mod>/rc/icon16.png"))`" — which is functional but non-idiomatic.
  B738 §738.4 shows the N4 convention is `icons/x16/` + `icons/x32/` resource dirs (not `rc/`), uses
  `BIcon.std("<file>")` (which resolves to `module://icons/x16/<file>` per `BIcon.java:69-71`), plus SVG
  via `BIcon.make(BOrd)` for a scalable single-file icon. Neither `static final` caching nor layered icons
  via `BIcon.make(BOrdList)` appear anywhere in the kit. `[ev: corpus B738 §738.4]`
- `report-module.sh:122-123` greps for `'^slot-coverage: WARN dup-keys:'` but `slot-coverage.sh:386`
  emits `'slot-coverage: FAIL dup-keys: ...'` (upgraded from WARN to FAIL in A1/B792). The consumer
  pattern was never updated, so `dup_ct` is always 0 — duplicate-key FAILs are silently swallowed and
  `report-module.sh` always emits `PASS dup-keys` even when duplicates are present. `[ev: corpus B759 §759.1]`
  `[ev: kit toolbelt/slot-coverage.sh:377-387 vs toolbelt/report-module.sh:122-123]`

## Proposed kit deltas (propose-never-apply)

| Δ | Delta | Target file / § | Token |
|---|---|---|---|
| Δ1 | `check_palette` should WARN (or FAIL under `--strict`) when the `module.palette` root `<p>` carries `t="b:Folder"` instead of `t="b:UnrestrictedFolder"` — `b:Folder` is access-controlled (gated) while `b:UnrestrictedFolder` is the idiomatic ungated palette root; using `b:Folder` causes "access denied" palette expansion for engineers without the folder's category permission | `toolbelt/verify-module.sh` §`check_palette` | `[ev: corpus B746 §746.1]` |
| Δ2 | Add a §"Palette assembly templates" recipe: nested `<p>` children with property overrides in `module.palette` ship pre-configured commissioning-ready assemblies (drag one component = correct tree + flags already set); document the authoring workflow (build correct assembly in a scratch station, copy it into the palette view + Save → serialized BOG XML; or hand-edit); include the `hasDefrost=true` / `DefrostController`-as-child example to illustrate how the B731 trap (false → never defrosts) is baked out of the template | `types/structure.md` §palette authoring (new) | `[ev: corpus B746 §746.3]` |
| Δ3 | Add a §"-doc/help profile authoring recipe": `<module runtimeProfile="doc">` with empty `<types/>`, a `src/doc/toc.xml` (JavaHelp 1.0 TOC DTD, `<tocitem text= target= image=>`), Guide-on-Target HTML naming `doc/<mod>-<TypeName>.html`, On-View naming `doc/<mod>-<ViewTypeName>.html`, the `help.guide.base` lexicon key (`help.guide.base=module://docUser/doc`) linking a type's F1 help to its HTML; note the two build plugins (`com.tridium.niagara-doc` for help content, `com.tridium.bajadoc-module` for API Javadoc); note our modules ship none today | `types/structure.md` §`-doc` help profile (new) | `[ev: corpus B759 §759.6]` |
| Δ4 | Add a §"Adding a block icon" recipe to `logic-authoring.md`: (a) raster icons go in `icons/x16/` + `icons/x32/` module resource dirs (NOT `rc/`), bundled into the jar; (b) `private static final BIcon icon = BIcon.std("<file>");` resolves to `module://icons/x16/<file>` (must `static final` — never build per call); (c) `public BIcon getIcon(){ return icon; }` override; (d) SVG (one scalable file) via `BIcon.make("module://<mod>/icons/<file>.svg")` skipping the x16/x32 raster pair; (e) layered/badge icons via `BIcon.make(BOrdList)` (base + overlay); NOTE that `METHODOLOGY.md:16`'s `src/rc/icon16.png` recipe is functional (rc/ is a valid module resource path) but non-idiomatic — the N4 convention is `icons/x16/`; clarify distinction: `rc/` = servlet static web assets, `icons/x16/` = module-level component icons | `types/logic-authoring.md` §"Adding a block icon" (new) | `[ev: corpus B738 §738.4]` |
| Δ5 | Fix `report-module.sh` dup-key grep: change `grep -c '^slot-coverage: WARN dup-keys:'` to `grep -c '^slot-coverage: FAIL dup-keys:'` (or `grep -c 'dup-keys:'`) — `slot-coverage.sh` was promoted from WARN to FAIL in A1/B792 but the consumer in `report-module.sh` was never updated, so every module with duplicate lexicon keys silently emits `PASS dup-keys 0` in the aggregate report, defeating the B792 upgrade | `toolbelt/report-module.sh` line ~122 | `[ev: corpus B759 §759.1]` `[ev: kit retro campaign8-slot-per-slot (A1/B792)]` |

## Lessons

- `BIcon.std()` and `icons/x16/` are the N4 idiomatic icon convention; `rc/` is the servlet web-asset
  directory. Both paths are technically accessible via `module://` ORDs but they serve different
  purposes. The kit currently conflates them; Δ4 disambiguates.
- A WARN-to-FAIL upgrade in a lint tool must be followed by an update to every grep/parse consumer
  (report-module.sh, bats tests). Δ5 is a concrete example of the gap surviving undetected.
- Pre-wired palette assembly templates (Δ2) are pure resource additions — no code risk, additive —
  making them the lowest-cost way to bake correct structure and default flags into commissioning.
- The `-doc` profile (Δ3) is a self-contained shipping unit orthogonal to the code modules; it is
  already architecturally correct to ship none today, but the recipe should exist so any future
  help investment has a clear, kit-documented path.
- `b:Folder` vs `b:UnrestrictedFolder` (Δ1) is a silent commissioning friction: the palette appears
  in Workbench but expansion fails for engineers without the correct category permission — easy to
  miss in testing where the developer always has full access.

---
**Status**: PENDING — INDEX row appended: `| 2026-09-18-palette-lexicon-assets-images-deltas.md | kit | 2026-09-18 | pending | 5 |`
