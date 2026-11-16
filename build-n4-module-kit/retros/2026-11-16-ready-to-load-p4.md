<!-- review-status: pending -->
<!-- module: kit -->
<!-- deltas: 2 -->
# 2026-11-16 · kit · ready-to-load-p4

**Session**: row P4 of the 2026-11-16 client A over-amps retro: a "ready to load" report for a built client module.
**Delta count**: 2

## What happened
Before the operator loaded a client A module on its production station, the coordinator checked by hand the jar's
module and vendorVersion, the `baja` floor of `META-INF/module.xml` and the Niagara of the mirror it was compiled
against, the signer files, the verify-module result, schema-risk against the deployed version, the jar's sha256
and whether the mirror held the deployed jar again. Nothing printed them in one place, and nothing compared them
with the operator's target Niagara and signer. `report-module.sh --ready` now does, per built jar, with an explicit
`--expect-niagara` and `--expect-signer`, and prints the read-back to run after the load.

## Evidence
- RED before GREEN: the ten `TestReady*` cases failed (usage exit 2, no ready rows) before the mode existed; the
  old-behaviour pin `TestReadyLeavesTheFullReportAlone` and the extended `TestParseArgs` stayed green.
  `[ev: code go/internal/reportmodule/ready_test.go]`
- The kept-jar naming of `build.sh` and the schema-risk verdict were shared first, behaviour unchanged (the
  buildmodule and reportmodule suites green on the refactor commit alone). `[ev: code go/internal/reportmodule/mirror_kept.go]`
- Differential of the full report on `tests/fixtures/report-module` against origin/main: identical stdout, stderr
  and exit on every fixture; only the usage text gains the `--ready` paragraph. `[ev: code go/internal/reportmodule/args.go]`

## Proposed kit deltas (propose-never-apply)
| Δ | Delta | Target file / § | Token |
|---|---|---|---|
| Δ1 | APPLIED 2026-11-16. `report-module.sh --ready`: module, niagara-target, niagara-home, signer, verify-module, schema-risk, sha256 and mirror rows; READY TO LOAD only when no row FAILs or ERRORs, else NOT READY with the failing rows; `--json`; no station contact | `build-verify.md` § Ready to load · `toolbelt/report-module.sh` header | `[ev: code go/internal/reportmodule/ready.go]` |
| Δ2 | APPLIED 2026-11-16. The post-load read-back (joins D10/T33) is printed, never run: `n4_station_runtime` must list the part at the built version with no mismatches | `build-verify.md` § Ready to load | `[ev: code go/internal/reportmodule/ready_output.go]` |

## Lessons
- A load decision needs the operator's target stated: a check that reads only the jar can say what it requires,
  not whether the station meets it, so the target and signer flags turn assumptions into rows.
