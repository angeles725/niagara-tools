<!-- review-status: pending -->
<!-- module: kit -->
<!-- deltas: 3 -->
# 2026-10-31 · kit · reference-port-parity

**Session**: niagara-tools kit, branch `feat/port-parity-gate` from c9164303 (new mechanical gate after a client A port)
**Delta count**: 3

## What happened
A client A dashboard module ported a reference viewer script. The port went from 646 to 220 lines and lost the
whole equipment view (`#equipmentView` and about 22 functions) on every level. The design table said "same", and
no check compared the reference with the port, so the drop reached hand-off unseen. The user's rule: "No puede
volver a pasar esto." A one-off audit script listed the absences (reference files with no same-named port file,
reference DOM ids and named functions absent from the port); this retro turns that audit into a kit gate in Go.

## Evidence
- Scan of reference and port trees: files by basename, DOM ids (also inside JS strings), named functions
  `[ev: 41627375]`
- Waivers judged per line: no `[decision: <ref>]` is a FAIL, a waiver matching nothing is a stale WARN `[ev: 049940df]`
- Command with module mode (`<module-root>/.port-parity`) and flag mode, text and JSON output, exits 0/1/3 `[ev: d50bc132]`
- Mutation proof: `tests/port-parity.bats` PP-drop-waiver (removing one waiver line flips exit 0 to 1) and
  PP-no-decision (a waiver without `decision:` FAILs) go RED when the waiver match or the decision check is dropped
  `[ev: d50bc132]`

## Proposed kit deltas (propose-never-apply)
| Δ | Delta | Target file / § | Token |
|---|---|---|---|
| Δ1 | A reference port runs `toolbelt/lint-port-parity.sh` before hand-off; every absence needs a waiver with a decision reference | `BUILD-LOOP.md` § 0. Orient | `[ev: 049940df]` |
| Δ2 | `report-module.sh` runs lint-port-parity on every module that has a `.port-parity` file (no row otherwise) | `go/internal/reportmodule/members.go` perModule | `[ev: d50bc132]` |
| Δ3 | Frontend doctrine: a port of a reference frontend declares it in `.port-parity` and waives each drop with `[decision: <ref>]` | `types/frontend-standard.md` § Prototype → module | `[ev: 41627375]` |

## Lessons
- A design table that says "same" is a claim, not a check: compare the reference and the port mechanically.
- Dropping a reference feature is a product decision; the waiver must name where the user decided it.
- A gate that reads an empty tree must fail as an environment error, never pass as "nothing missing".
- A new report-module member that applies to few modules adds no row elsewhere, so the golden reports stay unchanged.
