<!-- review-status: folded -->
# 2026-09-19 · kit · se-profile

**Session**: 2026-09-19 kit-improvement campaign (branch odd/apply-kit-candidates-2026-09-19) — WU folding the -se profile cluster (deploy scope now includes JACE).
**Delta count**: 7

## What happened
structure.md said only "-se server-only"; no Java-SE definition, JACE load behavior, or install
mechanics. Cluster SE-C1..SE-C7 in the master register.

## Evidence
- -se = Java SE dependency; daemon loads {rt,se} via -rp:se: `[ev: devguide modules.txt]` `[ev: corpus B630]`
- -wb-deps-in-se ClassNotFound trap; headless AWT trap: `[ev: code alarm-se module.xml]` `[ev: code obix-se module.xml]`
- JACE install no-rollback: `[ev: corpus B633]`; resource limits `[ev: corpus B473]`

## Proposed kit deltas (folded in this PR)
| Δ | Delta | Target | Token |
|---|---|---|---|
| Δ1 | -se = Java SE dep (awt/swing/jdbc), not JACE-exclusive | `types/structure.md` §-se | `[ev: devguide modules.txt]` |
| Δ2 | daemon loads {rt,se} on JACE AND Supervisor; -wb invisible | `types/structure.md` §-se | `[ev: corpus B630]` |
| Δ3 | -wb-deps-in-se ClassNotFoundException trap | `types/structure.md` §-se | `[ev: code alarm-se module.xml]` |
| Δ4 | JACE headless AWT trap (display classes fail) | `types/structure.md` §-se | `[ev: code obix-se module.xml]` |
| Δ5 | JACE install: stop-all, overwrite, no rollback, USER_HOME fallback | `types/structure.md` §-se | `[ev: corpus B633]` |
| Δ6 | multi-part -se sibling recipe | `types/structure.md` §-se | `[ev: corpus B784]` |
| Δ7 | JACE resource limits + theme/-doc pointer | `types/structure.md` §-se | `[ev: corpus B473]` |

## Lessons
- -se loads on JACE; display-dependent Swing code in -se fails at runtime with no build-time signal.
- JACE module install overwrites in place with no backup/rollback — back up modules/ before shipping.

---
**Status**: FOLDED into `types/structure.md` §-se (2026-09-19).
