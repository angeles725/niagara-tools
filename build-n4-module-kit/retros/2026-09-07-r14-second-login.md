<!-- review-status: pending -->
# 2026-09-07 · DashboardPan · r14-second-login

**Session**: PANCCADIA — second login (individual Niagara credentials) required before any dashboard write.
**Delta count**: 2

## What happened
Model: login 1 = shared station session to VIEW; login 2 = an individual local Niagara username/password INSIDE the dashboard to UNLOCK writes, attributed per user. Implemented real-credential re-auth (B830 `BUserService.getUser → auth.canLogin → BPasswordCache.validate → authenticateOk/Failed`, fail-closed on non-BPasswordCache), a write-session keyed by the existing JSESSIONID (sliding 5-min TTL, no self-issued cookie), a write gate, and audit persisted-not-shown. Found today's writes did `parent.set(prop,value,null)` — a NULL Context that bypasses BOTH native permission AND the AuditEvent.

## Evidence
- `handleSetpointWrite`: `DashboardConfigSession.requireActive` FIRST → 403 `config_login_required`; then `checkCanWrite(login2User)`; then `parent.set(prop, toSet, writeContext)` with the login-2 `BUser` as Context [ev: BDashboardServlet.java:211/313]
- `DashboardConfigAuth` uses `BPasswordCache.validate` (9 refs) — no fixed/config password [ev: DashboardConfigAuth.java]
- `lint-servlet` 0 NEW security FAILs; audit persisted to `auditLog`, no read endpoint, no `#__log`; `2.4.0` [ev: lint-servlet.sh]

## Proposed kit deltas (propose-never-apply)
| Δ | Delta | Target file / § | Token |
|---|---|---|---|
| 1 | `lint-servlet` security rule: a write handler that calls `set(prop,val,null)` (NULL Context) BYPASSES native BPermissions enforcement AND the AuditEvent — flag it and require the authenticated `BUser` be passed as the `set()` Context. | toolbelt/lint-servlet.sh | [ev: BDashboardServlet.java] |
| 2 | Doctrine for a "second login before write": re-auth via the certified B830 `BPasswordCache` path (fail-closed on LDAP/SAML), key the write-session to the existing HttpOnly JSESSIONID (avoid a self-issued cookie / CSRF-token surface). | types/dashboard.md §write-auth | [ev: DashboardConfigSession] |

## Lessons
- `canLogin()` is on `javax.baja.security.BAbstractAuthenticator` (not `BUser`); `BPasswordCache`/`BAbstractAuthenticator` live in `javax.baja.security` — verify package membership with javap before coding.
- `BUser implements Context`, so passing it as the `set()` Context is a direct cast — and it is what turns on native permission + audit attribution.
- Binding to the existing JSESSIONID removes a session-fixation/CSRF surface a custom cookie would add.

---
**Status**: PENDING — INDEX row appended: `| 2026-09-07-r14-second-login.md | DashboardPan | 2026-09-07 | pending | 2 |`
