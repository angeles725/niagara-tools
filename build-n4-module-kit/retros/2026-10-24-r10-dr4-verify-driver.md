<!-- review-status: pending -->
<!-- module: kit -->
<!-- deltas: 1 -->
# 2026-10-24 · kit · r10-dr4-verify-driver

**Session**: niagara-research quality round 10, driver-authoring task DR4, branch `r10/dr4-verify-driver` from
origin/main (DR2 #502; fast-forwarded to 83ffead5 with verify-module phase B #505 and DR3 #503 before the first commit)
**Delta count**: 1

## What happened
The Go `verify-module` gained a driver mode. With `--src`, a profile dir whose sources hold a class extending a
network or device base (directly or through its own classes, `javasrc.Classes` + `Descendants`) gets five rows,
appended after the existing checks: `driver-scaffold`, `driver-comm-config`, `driver-discovery-alive`,
`driver-server-enabled` and `driver-manager`. Every other module prints no driver row, so the recorded goldens are
unchanged. Each check went RED first on a negative copy of `fixtures/MinimalDriver` generated in the test, and the
fixture itself passes clean (4 PASS, 1 SKIP: it has no TCP config).

## Evidence
- Doctrine per row: lint-driver-scaffold § 5 / § 1.6; bare comm config § 1.9 ("Subclass the config; never ship a
  bare `BUdpCommConfig`"); `isAlive()` § 1.11 step 1; `serverEnabled` § 1.6 ("Client drivers keep `serverEnabled`
  false"); point manager `@AgentOn` § 1.6; PD-01 § 1.3 (`types/driver-authoring.md`).
- PD-01 names `BBasicNetwork` drivers only. The MinimalDriver NDriver network ships a point manager and no device
  manager and relies on the framework's device manager (`getDeviceManagerSubscribeDepth`, § 8), so the gate binds
  PD-01 to `BBasicNetwork`/`BSerialNetwork` and reports it as not applicable on an NDriver network
  (`go/internal/verifymodule/driver.go` driverManager).
- The enum-proxy range rule (§ 2) asks commissioning to set the point's `range` facet in the station, so no source
  rule exists to check; the gate leaves it out instead of inventing one.
- Fixture run: `verify-module.sh --src fixtures/MinimalDriver/MinimalDriver <stub MinimalDriver-rt.jar>` prints
  PASS driver-scaffold, driver-comm-config, driver-discovery-alive (2 hosts), driver-manager and SKIP
  driver-server-enabled (the fixture ships no module jar, so a stub jar stands in; its other rows are not about
  the driver).

## Proposed kit deltas (propose-never-apply)
| Δ | Delta | Target file / § | Token |
|---|---|---|---|
| Δ1 | verify-module driver mode: detection rule + five driver rows, each pinned on MinimalDriver and a negative copy; the enum range stays a commissioning check | `go/internal/verifymodule/driver.go`; `types/driver-authoring.md` § 12; `build-verify.md`; `skill/SKILL.md` | `[ev: code go/internal/verifymodule/driver_test.go]` |

## Lessons
- A doctrine rule scoped to one framework (PD-01 on `BBasicNetwork`) must keep that scope in the gate: applied to
  an NDriver network it would fail the kit's own scaffold.
- A rule that lives in station configuration (a point's facet) has no source pin; say so in the doctrine instead
  of a heuristic check.

## Owed
- The `toolbelt/verify-module.sh` header (the Go usage text is its lines 2-20) does not list the driver rows; it
  was outside this task's edit surface.
- A source-side enum-range rule needs doctrine first (for example, a discovery leaf that proposes an
  `EnumPoint` setting a frozen-enum range in `updateTarget`).
