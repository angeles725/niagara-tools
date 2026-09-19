<!-- review-status: folded -->
# 2026-09-19 · kit · security-model

**Session**: 2026-09-19 kit-improvement campaign (branch odd/apply-kit-candidates-2026-09-19) — WU folding the security cluster (user-mandated first-class).
**Delta count**: 11

## What happened
The user made audit, RBAC+CSRF+secrets, licensing, and anti-injection first-class. Security guidance
was scattered across the kit; consolidated into one doc. Cluster SEC-01..SEC-11 in the master register.

## Evidence
- null-Context write skips audit AND grants BPermissions.all: `[ev: code ComplexSlotMap.java]`
- CsrfProtectedFilter + x-niagara-csrfToken: `[ev: devguide csrfProtection]`; BPassword doPrivileged `[ev: code BPassword.java]`
- BqlQuery.toBqlLiteral + SlotPath.escape; Reflow anti-patterns (HTTP phone-home, GET-destructive, open BQL/BOrd, ^-enumeration): `[ev: corpus B507]`

## Proposed kit deltas (folded in this PR)
| Δ | Delta | Target | Token |
|---|---|---|---|
| Δ1 | null-Context audit+permission double hazard | `types/security.md` §1 | `[ev: code ComplexSlotMap.java]` |
| Δ2 | BasicContext(user) chain for attributable servlet writes | `types/security.md` §1 | `[ev: code ComplexSlotMap.java]` |
| Δ3 | oBIX PUT shared-user attribution gap | `types/security.md` §1 | `[ev: corpus B507]` |
| Δ4 | BPermissions 6-bit table + Flags.isOperator tier | `types/security.md` §2 | `[ev: corpus B507]` |
| Δ5 | CsrfProtectedFilter + x-niagara-csrfToken (vs X-Requested-With) | `types/security.md` §3 | `[ev: devguide csrfProtection]` |
| Δ6 | BPassword doPrivileged/SecretChars, never log getValue | `types/security.md` §3 | `[ev: code BPassword.java]` |
| Δ7 | BQL toBqlLiteral + SlotPath.escape (no parameterized API) | `types/security.md` §4 | `[ev: corpus B507]` |
| Δ8 | BOrd.make(clientInput) allowlist requirement | `types/security.md` §4 | `[ev: corpus B507]` |
| Δ9 | ^-root enumeration permission check | `types/security.md` §4 | `[ev: corpus B507]` |
| Δ10 | licensing BILicensed/getLicenseFeature/serviceStarted | `types/security.md` §licensing | `[ev: code BAaPhpNetwork.java]` |
| Δ11 | anti-patterns table (5 Reflow findings) | `types/security.md` §anti-patterns | `[ev: corpus B507]` |

## Lessons
- A Context-less write is a double footgun: no audit record AND full permissions.
- BQL has no parameterized query — `toBqlLiteral()`+`SlotPath.escape()` is the only safe embedding.

---
**Status**: FOLDED into `types/security.md` (2026-09-19).
