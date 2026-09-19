<!-- review-status: folded -->
# 2026-09-18 · kit · ord-path-nav-deltas

**Session**: corpus-mining for build-n4-module (operator: generate all possible kit deltas — ORD/path/nav).
**Delta count**: 9

## What happened

Mined B5 (ORD + BOG + BQL/NEQL), B35 (nav tree + Workbench), B38 (BOrd resolution full scheme
catalog), B587 (`hierarchy:` ORD scheme), B757 (BAbstractService + BINavNode), and B778 (custom ORD
scheme authoring) for gaps in `types/logic-authoring.md`, `types/dashboard.md`, and `types/structure.md`.

**Already folded — skipped:**

- Custom ORD scheme: `extends BOrdScheme` + `@NiagaraType(ordScheme)` + `resolve()` → `types/logic-authoring.md §Author-side SPIs` [B778]
- `SERVICE_ORD` pinning + traversal reject (five write-surface gates) → `types/dashboard.md §Write-surface gates` [B763]
- `module://` ORD for rc resource serving → `types/dashboard.md §Web-tier exemplars` [B791]
- `BSimpleJob.submit()` returns a `BOrd` handle to poll job state → `types/logic-authoring.md §Background jobs` [B774]
- Cross-station boundary: `fox:` ORD hop = real JVM boundary → `types/logic-authoring.md §Cross-module comms` [B802]
- BQL/NEQL provider recipe (`BQuery` + `BIAgent`) → `types/logic-authoring.md §Query/search surface` [B782]
- `SlotPath.unescape` for username in RBAC check → `types/dashboard.md §RBAC` [B763]
- `BAbstractService` authoring recipe + `getServiceTypes()` → `types/logic-authoring.md §Author-side SPIs` [B778/B757]

**Gaps found — 9 proposed deltas:**

Five content areas are absent from kit guides:

1. **BOrd resolution API for rt/ux code** — the call site recipe (`BOrd.make` → `resolve()` →
   `OrdTarget.get()`) and error types (`UnresolvedException`, `UnknownSchemeException`,
   `InvalidOrdBaseException`) are nowhere in the kit; only the custom-scheme authoring recipe exists.

2. **`h:` handle scheme semantics in BOG** — handles are BOG-document-local (not globally unique),
   path-independent (survive moves), and intra-document `BOrd` properties serialize as `v="h:N"`.
   This affects palette template authoring (kit's `types/logic-authoring.md §Palette`).

3. **Cross-station ORD pipe recipe** — `ip:host|fox:|station:|slot:/path` and the normalization rule
   (host query trims prior fragments) are missing from the kit. The `fox:` boundary is noted as a
   boundary concept but the resolution composition recipe is absent.

4. **`service:` lookup scheme** — `service:baja:AlarmManager` resolves the running service instance
   by type; an alternative to `Sys.getService(Type)` when storing a configurable ORD reference in a
   property. Neither the scheme nor the alternative use-case is documented.

5. **Nav-tree ordering and visibility levers** — the two levers (`isNavChild()→false` hides from nav
   while keeping the slot; slot order = nav order) and the virtual nav node recipe are not in the kit.
   B757 is cited in the corpus-index at P2 but no kit section reflects these authoring rules.

6. **`.nav` file XML schema + static-cache gotcha** — the nav-file node/ord/icon XML schema for
   packaging a custom nav tree in a module JAR is absent; the `NavFileDecoder.cache` static-map
   gotcha (edits on disk while WB is open → sidebar does NOT refresh) is unrecorded.

7. **`hierarchy:` ORD stability property** — the kit does not mention that a `hierarchy:` ORD
   survives component-tree reorganization (resolved through tags/relations) or the escaped
   `station:|` leaf seam for cross-station entity dereference.

8. **ORD scheme typology reference** — the 5-type classification (host/session/space/lookup/query)
   and the full use-case-aligned scheme table are absent; builders currently re-derive the catalogue
   each time from B5 rather than having a compact kit reference.

9. **`nav:` ORD scheme mechanics** — how `BNavScheme` resolves `nav:Station/Floor1/Zone` by walking
   `getNavChild()` segment by segment with inline permission enforcement (`checkPermissions` raises
   `UnresolvedException`, not a visible error) is undocumented and affects any module that references
   nav nodes in ORD properties or graphics bindings.

## Evidence

- B5 §5.1.4: `BOrd.resolve(base,cx)` pipeline — parse → normalize → iterate queries → `target.get()` → BObject; errors: `NullOrdException` / `UnknownSchemeException` / `SyntaxException` / `UnresolvedException` / `InvalidOrdBaseException` `[ev: corpus B5 §5.1.4]`
- B5 §5.1.5: absolute ORD (host-anchored, no base needed) vs relative (requires non-null base); normalization trims prior fragments when a host query is encountered `[ev: corpus B5 §5.1.5]`
- B5 §5.1.1: 29 registered schemes; scheme typology (host/session/space/lookup/query); `service:baja:AlarmManager` example `[ev: corpus B5 §5.1.1]`
- B5 §5.2.4: handle = opaque string, unique per BOG DOCUMENT (not global); BOrd property serializes as `v="h:N"` for intra-document reference; `h:` ORD stable across renames/moves unlike `slot:` path `[ev: corpus B5 §5.2.4]`
- B5 §5.1.1 / B38 §38.1: `service:` scheme resolves running service instance; `BServiceScheme` + `ServiceQuery` + `ServiceSession` inner classes carry lateral context `[ev: corpus B5 §5.1.1; B38 §38.1]`
- B35 §35.5.2: `BNavScheme extends BOrdScheme`; `nav:Seg1/Seg2/...` walks `getNavChild()` per segment; `checkPermissions` raises `UnresolvedException` on denied node — UI silently fails to expand `[ev: corpus B35 §35.5.2]`
- B35 §35.5.3: `.nav` file XML schema — `<nav><node name="..." ord="..." icon="...">…</node></nav>`; `NavFileDecoder.cache` is `static Map`; edits on disk while WB is running do NOT propagate — sidebar shows stale tree until reload/Ctrl+R `[ev: corpus B35 §35.5.3]`
- B757 §757.3: the two nav levers — override `isNavChild()→false` to hide from nav only (vs `Flags.HIDDEN` = remove from ALL UI); slot order = nav order (reorder slots → reorder nav) `[ev: corpus B757 §757.3]`
- B757 §757.4: virtual nav node recipe — implement `BINavNode` methods manually (no backing slot); supply stable `getNavOrd`; parent's `getNavChildren` returns the synthetic instance; canonical: `BModulePaletteNode` `[ev: corpus B757 §757.4]`
- B587 §587.1: `BHierarchyScheme extends BSpaceScheme`, `ordScheme="hierarchy"`; body parses as `HierarchyQuery extends SlotPath`; last segment starting `station$3a$7c` (escaped `station:|`) = ENTITY leaf → dereferences to real component with permission check; a `hierarchy:` ORD is stable across component-tree reorganisation `[ev: corpus B587 §587.1]`

## Proposed kit deltas (propose-never-apply)

| Δ | Delta | Target file / § | Token |
|---|---|---|---|
| Δ1 | Add `## ORD resolution — calling BOrd from rt/ux code` section: (1) `BOrd.make(string)` + `resolve(base, cx)` → `OrdTarget.get()` = the call-site recipe; (2) absolute ORD (host-anchored, base may be null) vs relative (requires base); (3) five error types and what each means in practice (`UnresolvedException` = path not found; `UnknownSchemeException` = module not loaded; `InvalidOrdBaseException` = wrong context; `SyntaxException` = malformed; `NullOrdException` = empty ord); (4) normalization note: a host-query fragment trims all prior fragments — `slot:/a|ip:host|slot:/b` resolves to `ip:host|slot:/b` | `types/logic-authoring.md` new §ORD resolution | `[ev: corpus B5 §5.1.4–5.1.5]` |
| Δ2 | Add ORD scheme typology table to the new §ORD resolution: 5 types (host / session / space / lookup / query) with one-line use case each + notable examples (`local:`, `ip:`, `fox:`, `station:`, `slot:`, `h:`, `module:`, `service:`, `type:`, `bql:`, `neql:`, `hierarchy:`, `nav:`); note that custom schemes register via `@NiagaraType(ordScheme="id")` (already in §Author-side SPIs) | `types/logic-authoring.md §ORD resolution` sub-table | `[ev: corpus B5 §5.1.1; B38 §38.1]` |
| Δ3 | Add `service:` lookup scheme to §ORD resolution or §Author-side SPIs: `service:mymod:FooService` resolves the running service instance by type — an alternative to `Sys.getService(Type)` when you need to store a configurable ORD pointer in a property; warn that it resolves the FIRST-registered instance (same semantics as `Sys.getService`) | `types/logic-authoring.md §ORD resolution` | `[ev: corpus B5 §5.1.1; B38 §38.1]` |
| Δ4 | Add `h:` handle scheme semantics to §ORD resolution: handle is BOG-document-local (NOT globally unique — same handle id in two BOG files = two distinct entities); `BOrd` properties in BOG serialize as `v="h:N"` for intra-document links; `h:` ORD is path-independent (component can be renamed/moved without breaking a `h:` reference) — contrasts with `slot:` path which breaks on rename | `types/logic-authoring.md §ORD resolution` | `[ev: corpus B5 §5.2.4]` |
| Δ5 | Add cross-station ORD composition recipe to §ORD resolution: `ip:host|fox:|station:|slot:/path` chains fragments left-to-right; use `BOrd.make(base, suffix)` to compose; the `fox:` hop is the real JVM boundary (already noted in §Cross-module comms) — extending that note with the full pipe form and an example of how a servlet resolves a stored-ORD property pointing to a remote station | `types/logic-authoring.md §ORD resolution` | `[ev: corpus B5 §5.1.1, §5.1.5]` |
| Δ6 | Add `## Nav-tree ordering and visibility` subsection under the BAbstractService/BINavNode entry in §Author-side SPIs: (1) `isNavChild()→false` hides from the nav sidebar ONLY (use this when a child should exist as a slot but not clutter the nav tree); `Flags.HIDDEN` removes from ALL UI (wireable property still present, but invisible everywhere); (2) nav order = slot declaration order — reorder/rename `@NiagaraProperty` fields to reorder; (3) virtual nav node recipe: implement `BINavNode` methods manually, return the synthetic instance from the parent's `getNavChildren()`, supply a stable `getNavOrd()` | `types/logic-authoring.md §Author-side SPIs` new subsection | `[ev: corpus B757 §757.3–757.4]` |
| Δ7 | Add `## nav: ORD scheme` note in §ORD resolution or as a corpus-index P2 pointer: `BNavScheme` resolves `nav:Seg1/Seg2` by walking `getNavChild()` per segment; permission enforcement is INLINE (denied node → `UnresolvedException`, UI silently fails to expand — no error shown to the user); a nav: ORD in a PX graphics binding targets a nav-file node, not a station component directly | `types/logic-authoring.md §ORD resolution` | `[ev: corpus B35 §35.5.2]` |
| Δ8 | Add `.nav` file packaging note to `types/structure.md` (or `METHODOLOGY.md §Package`): `.nav` files live in `rc/nav/` inside the module JAR (or `~/stations/<s>/nav/` at runtime); XML schema is `<nav><node name="..." ord="..." icon="...">…</node></nav>`; STATIC-CACHE GOTCHA: `NavFileDecoder.cache` is a JVM-static Map keyed by ord — editing a `.nav` file on disk while Workbench is open does NOT refresh the sidebar until the user does Ctrl+R (reload) or reconnects; design around this for dev: ship the `.nav` as a resource in `rc/`, not an editable file | `types/structure.md` new §Nav file packaging | `[ev: corpus B35 §35.5.3]` |
| Δ9 | Add `hierarchy:` ORD stability note to corpus-index P2 or `types/logic-authoring.md §ORD resolution`: a `hierarchy:` ORD survives component-tree reorganisation (it resolves through tags/relations to the real component, not through slot paths); the last path segment encoded as `station$3a$7c` (URL-escaped `station:|`) is the ENTITY seam that dereferences to the real component with permission check; cite this when choosing between a `slot:` path and a `hierarchy:` path in a module that must reference stable cross-tree addresses | `types/logic-authoring.md §ORD resolution` | `[ev: corpus B587 §587.1]` |

## Lessons

- **BOrd resolution API is a conspicuous gap.** The kit covers custom scheme AUTHORING (B778) and ORD-as-a-gate (SERVICE_ORD, B763) but nowhere explains how to CALL `resolve()` from application code. Every servlet or reader that resolves a stored ORD property re-derives this from first principles.
- **`service:` scheme is hidden.** It appears in the B5 scheme table and B38 catalog but is never mentioned in the kit. Builders default to `Sys.getService(Type)` and miss the option to store a configurable ORD pointer for a service reference.
- **Nav-tree visibility vs HIDDEN is a non-obvious distinction.** B757 §757.3 makes it explicit: `isNavChild()→false` = nav only; `HIDDEN` = everywhere. Without this, builders either over-hide (remove from wiring) or under-hide (clutter the nav tree).
- **NavFileDecoder static cache is a dev-time footgun.** A builder who packages a `.nav` file and tests in a live Workbench session will see stale nav content with no error. The workaround is easy but the symptom ("why doesn't my nav change?") is confusing without prior context.
- **B757 is indexed at P2 but not folded.** The corpus-index entry points to B757 for "service + nav tree" but the kit types files contain only the service recipe; none of B757 §757.3–757.5 has a corresponding kit section. Cited-but-not-folded is a recurring gap pattern (also seen in B777 in the previous retro).

---

**Status**: PENDING — INDEX row appended:
`| 2026-09-18-ord-path-nav-deltas.md | kit | 2026-09-18 | pending | 9 |`
