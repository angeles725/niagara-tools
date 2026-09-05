<!-- review-status: pending -->
<!-- Marker lifecycle: the maintainer flips 'pending' to 'applied <date> · kit <sha>' (or 'dismissed') once these deltas are reviewed/folded into the build-n4-module kit; sweep-retros.sh reads this marker (METHODOLOGY §18). -->
# Retro — niagara-research · research-sdd · 2026-09-05 · focus `module-ux-testing-and-write-surface` (B762–B763) — kit deltas for /build-n4-module

> Run reviewed: the merged research lane (campaign6) — angle-2 residue (`-ux`/`-wb` off-station testing) + angle-3
> (`-ux` servlet write-surface authorization), merged because the thing you TEST is the thing you SECURE (the same
> servlet write endpoint). Two blocks: B762 (the test-seam taxonomy) + B763 (the write-surface playbook). Method:
> AUDIT-FIRST (most of the authz half was REMITTANCE → cited, not re-derived); the genuine research was the single
> uncovered testing gap. READ-ONLY on the build kit — PROPOSES only (§18); I do NOT edit `$KIT`.

## Proposed kit deltas (for `/build-n4-module`)

| # | Proposed change | Target (file · §) | Evidence (block · file:line) | Type | Priority |
|---|---|---|---|---|---|
| DUX1 | Add the `-ux` servlet **route() → RouteAction** seam as the authoring pattern: extract routing into a package-private `final` class taking `Function` header/param lookups, returning a sealed `RouteAction`; the servlet is a thin `instanceof` adapter over the **`final` WebOp**. Test with plain JUnit + `HashMap` (no station). | `types/dashboard.md` (new §"-ux testable seam") | B762 §762.2 · DashboardDispatch.java:30,11,38-43; DashboardDispatchTest 14 @Test | new | HIGH |
| DUX2 | Add the **purity gradient** rule: keep data-shapers pure by injecting the Baja touch as a `Function`/`Predicate`; `DashboardReader.buildEquipmentResponse(BComponent)` is the anti-pattern (takes a live component). | `types/dashboard.md` | B762 §762.3 · JsonUtil (0 baja imports) vs DashboardReader.java:12,143 | new | MED |
| DWB1 | Upgrade `-wb` doctrine from rung-0 seed to exemplar-backed: the **`wb/model/` lambda-injection seam** — a Baja-free `model/` package with the slot check injected as `Predicate<String>`; the `BWidget` view stays the adapter. **`-wb` IS off-station testable** (the discovery). | `types/wb-widgets.md` | B762 §762.4 · LinkSlotNameUtil.java:3,9,30; chihuahua-wb model 33 @Test | new | HIGH |
| DJS1 | JS residue note: for a testable SPA, split inline JS into files + add a dual-export shim (`if(typeof module!=='undefined')module.exports=…`) + a node harness; `node --check` proves SYNTAX only. DashboardPan's 2300-line inline `index.html` is the anti-pattern. | `types/dashboard.md` or `build-verify.md` | B762 §762.5 · Router.js:30,68,162-168 (no module.exports); DashboardPan index.html monolith | new | MED |
| DWS1 | Add **"The write-surface: five gates"** checklist for a mutating `-ux` endpoint: (1) `checkCanWrite` first line = `OPERATOR_WRITE` fail-closed (deny on no-user / no-service / `catch(Exception)`); (2) hand-rolled `X-Requested-With` guard IN the pure `route()` (framework CSRF filter covers `/rpc/*` only); (3) pin the write ORD under a `SERVICE_ORD` facade + traversal reject (or an explicit slot allowlist); (4) mutate under a per-Ord lock → HTTP **423** on contention; (5) audit every mutation (who/what/when/old→new), fire-and-forget, audit-failure-never-fails-the-write. | `types/dashboard.md` (new §"write-surface") | B763 §763.1-763.5 · DashboardRbacHelper.java:98,100; DashboardDispatch.java:123; BDashboardServlet.java:198,241,287-297; ChiRbacHelper ADR D1 :15-17 | new | HIGH |
| DWS2 | Add the **pure RBAC test seam**: extract the auth DECISION as a Baja-free `canWrite(boolean)`/`getRole`/`buildForbiddenJson` seam (chihuahua ADR D1) so write-auth is unit-testable off-station; `DashboardRbacHelper`'s collapsed Baja-only form is the anti-pattern. | `types/dashboard.md` | B763 §763.4 · ChiRbacHelper.java:26-28,54 (TESTABILITY SEAM) vs DashboardRbacHelper (no pure canWrite) | new | HIGH |

Rationale, one line each:
- **DUX1/DWB1** — WHY: `types/dashboard.md`/`types/wb-widgets.md` had no OFF-STATION test guidance for the web tier; these are the proven seams (14 + 33 pure tests in our own modules). COST: two doc sections. IMPACT: the web tier gets the same testable-core discipline `types/logic.md` has for control logic.
- **DUX2/DJS1** — cheap refinements that name the anti-patterns already in our tree.
- **DWS1/DWS2** — WHY: the write-surface security rule was scattered across blocks and unenforced in the kit; DWS1 is the checklist, DWS2 makes the security DECISION testable. IMPACT: a write endpoint is built, tested, and secured to one rule instead of per-module reinvention.

## Client-side punch-list (NOT a kit delta — for the module owner)
- DashboardPan: port chihuahua's pure `canWrite(boolean)` RBAC seam (so the decision is unit-testable); add a per-Ord write lock + 423 (concurrency); **re-grade BUILD-STATE issue U5** — its "OPERATOR-flag missing" claim is FALSE (DashboardPan enforces `OPERATOR_WRITE` fail-closed at `BDashboardServlet.java:198`, verified B763 §763.6). The genuine U5 residue is the pure-seam/lock/allowlist, not a missing authz check.

## Already covered (dedupe — proof the retro read the kit + corpus first)
- Control-logic testing (math seam / scheduler seam / live smoke) → [B743] + `build-verify.md` 4-layer stack; B762 is the WEB-TIER sibling, cites B743.
- The write-auth = `OPERATOR_WRITE` fail-closed FACT → already in [B648]/[B655] (chihuahua-source CS3); B763 APPLIES it into the kit checklist, does not re-derive.
- `-ux` RBAC OPERATOR_WRITE contrast → [B752]; template-method pure split → [B730 §730.7] + [B741 §741.2]; framework CSRF `/rpc/*` → [B58]/[B507].

## What went well (keep)
- AUDIT-FIRST kept the focus HONEST and SMALL: the authz half was REMITTANCE, so only 2 blocks (1 genuine gap + 1 playbook) instead of a padded campaign.
- Grep-verifying the delegated sweep's cites at the REAL client-repo paths caught that the sweep's abbreviated paths resolved correctly — and turned every load-bearing claim into [CERT] (B762 8/8, B763 6+1). (Reinforces the mega-campaign lesson: numeric/path claims are [CERT] only with a fresh grep.)
- B763 CORRECTED a BUILD-STATE open issue (U5) with code evidence rather than accepting it — a research lane feeding a truth back into the kit, not just adding docs.
