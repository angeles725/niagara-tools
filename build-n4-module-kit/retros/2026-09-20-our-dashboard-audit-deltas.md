<!-- review-status: pending -->
# 2026-09-20 · kit · our-dashboard-audit-deltas

**Session**: niagara-research focus `our-dashboard-audit` (B1062–B1065) — a rigorous audit of OUR OWN `DashboardPan` module (rt+ux+wb) against the kit documentation and the full corpus. Subject: `Cliente/Leon-Guanjuato/Dashboard/DashboardPan/`, main branch, build date 2026-08-31.

**Delta count**: 5

## What happened
Audited DashboardPan across four dimensions: (ODA1) coverage gate + structural map; (ODA2) servlet security audit: RBAC, CSRF/XHR guard, path traversal, and response headers; (ODA3) data path + frontend quality: REST-POLL model, `{v,st}` JSON contract, library bundling, preview server, and portability; (ODA4) rt model + wb profile + test coverage + synthesis verdict. Remittances to chihuahua (B648–B655), module-best-practices (B705–B710), write-surface seam (B796), and audit trail (B829) declared before opening gaps.

**Already covered by the kit (no delta needed):** RBAC fail-closed pattern (`BPermissions.OPERATOR_WRITE`, exception-path deny), CSRF/XHR guard on all `/api/*`, double path-traversal guard, zero CDN dependencies, fault-aware `{v,st}` JSON contract, and the preview-server ergonomics. These confirm existing kit rules (MBP2, types/security.md) hold. Proposed deltas target only the **gaps and risks** the audit surfaced.

## Evidence
- B1062 (structural map: `rt`=2 classes, `ux`=5 classes, `wb`=0 Java; empty palette confirmed; audit dimensions seeded). `[ev: corpus B1062]`
- B1063 (ODA2 security audit: RBAC=GOOD; CSRF=GOOD; traversal=GOOD; **RISK**: `setApiHeaders()` at `BDashboardServlet.java:540–543` omits `X-Content-Type-Options`; no `Content-Security-Policy` anywhere; ODA2-G1/G2 registered). `[ev: corpus B1063]`
- B1064 (ODA3 data path: REST-POLL 5 s explicit design choice; `DashboardReader.java:75` hardcoded `SERVICE_ORD`; `DashboardReader.java:78` hardcoded `ROOMS[]`; zero CDN JS+CSS+fonts confirmed; preview server XHR guard reproduced + config-login mock; ODA3-G2 registered). `[ev: corpus B1064]`
- B1065 (ODA4 synthesis: `module.palette` empty — MBP3 violation; `module.lexicon` empty; `_trimAuditRing` is a pure-String static with a "WSL unit testing" comment but no test exists in srcTest; ODA4-G1 = missing test; ODA4-G2 = live deploy unprobed; synthesis verdict: production-ready for its design scope, risks are friction not correctness). `[ev: corpus B1065]`

## Proposed kit deltas (propose-never-apply)
| Δ | Delta | Target file / § | Token |
|---|---|---|---|
| Δ1 | PD-ODA1 add a servlet security-header checklist to the guide: every `BWebServlet` API response path must call a `setApiHeaders()` helper that includes `X-Content-Type-Options: nosniff`; HTML responses must additionally set `Content-Security-Policy: default-src 'self'; script-src 'self' 'unsafe-inline'` — DashboardPan's `setApiHeaders()` omits both (ODA2-G1/G2); propose as a lint check in `lint-servlet.sh` | `docs/module-best-practices.md` MBP2 / `types/security.md` / `lint/wire-lints` | `[ev: corpus B1063]` |
| Δ2 | PD-ODA2 add config-driven layout guidance to the dashboard how-to: `SERVICE_ORD` and the room/slot array must be resolvable at runtime (e.g. walk `BDashboardService` children via BQL introspection, or expose them as configurable `@NiagaraProperty` slots) — hardcoding them in Java source means every new site requires a recompile; document the portability tradeoff and the mitigation pattern | `docs/how-to-create-an-n4-module.md` §dashboard / `types/dashboard.md` | `[ev: corpus B1064]` |
| Δ3 | PD-ODA3 strengthen the empty-palette rule: `module.palette` for a dashboard module MUST include at least the primary rt service type (`BDashboardService`) and the facade slot container (`BRoomPanel`) as draggable entries — the empty palette is an MBP3 violation that forces the integrator to use Config > New Component on every new deployment; document the minimum palette contract and cross-reference [B788] | `docs/module-best-practices.md` MBP3 / `types/dashboard.md` | `[ev: corpus B1065]` |
| Δ4 | PD-ODA4 add a test-coverage delta for ring-buffer and pure-string audit helpers: any rt class with a pure-String static helper annotated "for WSL unit testing" (e.g. `_trimAuditRing`) MUST have a corresponding JUnit test in the rt-test profile — the comment is the author's intent contract; the test's absence is a coverage gap detectable by a lint sweep of `srcTest` | `docs/module-best-practices.md` MBP2 / `types/dashboard.md` | `[ev: corpus B1065]` |
| Δ5 | PD-ODA5 add a commissioning-verify follow-up note to `commissioning-verify.sh`: after deploying a `BWebServlet`-based module, probe the live endpoint with a `curl` that checks the response headers for `X-Content-Type-Options` and `X-Frame-Options`; cross-reference the existing `commissioning-verify.sh` post-deploy checklist (ODA4-G2: DashboardPan runtime behavior was not probed in this focus — static audit only) | `docs/module-best-practices.md` / `commissioning-verify.sh` | `[ev: corpus B1065]` |

## Lessons
- DashboardPan's security model is **correct at the gate level** (RBAC, CSRF, traversal) but misses **header hygiene** on the API paths. The two-line fix (add `X-Content-Type-Options` to `setApiHeaders()`) is lower-effort than any functional security gap — documenting the full required set as a checklist prevents the same omission in future modules.
- The `SERVICE_ORD`/`ROOMS[]` hardcoding is an intentional project-scope decision, not a bug. The kit's job is to surface the tradeoff and provide the BQL-introspection alternative so future modules don't repeat it unknowingly.
- Empty `-wb` palette + lexicon are confirmed friction per [B788]; the delta proposes making the minimum palette contract explicit in MBP3 so it is enforceable, not just advisable.
- The `_trimAuditRing` no-test gap shows the pattern: a "testable by design" comment is only valuable when a test actually exists. Propose a lint rule that cross-checks `srcTest` for each pure-static helper that carries such a comment.

---
**Status**: PENDING — 5 proposed deltas; full detail in corpus B1062–B1065.
