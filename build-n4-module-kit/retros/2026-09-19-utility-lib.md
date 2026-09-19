<!-- review-status: folded -->
# 2026-09-19 · kit · utility-lib

**Session**: 2026-09-19 kit-improvement campaign (branch odd/apply-kit-candidates-2026-09-19) — WU folding the utility-lib cluster (vendor mining: clUtils*, jtds).
**Delta count**: 3

## What happened
The kit only covered modules with a component palette; it had no guidance on reusable library
modules (a wrapped OSS jar, or shared helper types). Cluster UL-01/UL-02/DB-01 in the master register.

## Evidence
- pure lib bundle (empty <types/>, autoload, getClassLoader): `[ev: code jtds-rt module.xml]`
- helper-type lib (agent/abstract, no palette, registerHelper): `[ev: code clUtils-rt module.xml]`
- protocol-specialization split + narrowed <on type>: `[ev: code clUtilsBacnet-rt module.xml]`

## Proposed kit deltas (folded in this PR)
| Δ | Delta | Target | Token |
|---|---|---|---|
| Δ1 | pure 3rd-party lib bundle shape (jtds): empty types, autoload, permissions | `types/utility-lib.md` §1 | `[ev: code jtds-rt module.xml]` |
| Δ2 | helper-type lib shape (clUtils): agent/abstract helpers, no palette, self-register | `types/utility-lib.md` §1 | `[ev: code clUtils-rt module.xml]` |
| Δ3 | protocol-specialization split `<base>-rt`→`<base><Protocol>-rt` | `types/utility-lib.md` §2 | `[ev: code clUtilsBacnet-rt module.xml]` |

## Lessons
- A library module can register NO types (pure classpath bundle) or agent/abstract helper types — neither needs a palette.
- Split protocol specializations so the base carries no driver dep; each specialization narrows the agent `<on type>`.

---
**Status**: FOLDED into `types/utility-lib.md` (2026-09-19).
