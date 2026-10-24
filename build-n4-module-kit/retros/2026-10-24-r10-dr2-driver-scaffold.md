<!-- review-status: folded -->
<!-- module: kit -->
<!-- deltas: 3 -->
# 2026-10-24 · kit · r10-dr2-driver-scaffold

**Session**: niagara-research quality round 10, driver-authoring task DR2, branch `feat/r10-dr2-driver-scaffold`
from origin/main (DR1 landed as #498)
**Delta count**: 3

## What happened
`scaffold-module.sh --type driver` now emits an NDriver module from a new fixture, `fixtures/MinimalDriver`: an
`-rt` part (network, device folder, device, points extension, point folder, proxy extension, a `BUdpCommConfig`
subclass, request/response messages, a message factory, a Baja-free codec, device and point discovery
preferences and leaves) and a `-wb` part (a point manager that replaces the generic NDriver one). The bats suite
went RED first (unknown `--type`), then green. The emitted tree was built signed through `build.sh` into a
mirrored 4.14.0.162 home: `verify-module: 26 passed, 0 failed, 9 skipped, 1 warned -> ALL PASS` (the WARN is the
missing moduleTest, owed to DR5). The codec srcTest runs as pure JUnit (16 tests) and goes red when the checksum
check is dropped.

## Evidence
- First build: `BMinimalDriverDevice.java:47: error: cannot access javax.baja.alarm.BIAlarmSource` until the `-rt`
  gradle file declared `api(":alarm-rt")`; the `-wb` manager failed on `javax.baja.ui.BWidget` until
  `api(":bajaui-wb")` (local build 2026-10-24, mirror of OptimizerSupervisor-N4.14.0.162).
- A device slot named `host`: `getHost() in BMinimalDriverDevice cannot implement getHost() in BISpaceNode`
  (same build); `BComponent implements BISpaceNode`
  (`organized/docSource/docSource-doc/extracted/baja/javax/baja/sys/BComponent.java:84`), whose default
  `getHost()` returns `BHost` (`.../baja/javax/baja/space/BISpaceNode.java:44`).
- Action binding: `organized/baja/baja/vineflower/com/tridium/sys/schema/ComplexIntrospector.java:316-333` looks up
  `do<Action>(param)` and falls back to `do<Action>(param, Context)` only when it is missing;
  `BNPointDeviceExt` already declares the one-argument form
  (`organized/docSource/docSource-doc/extracted/ndriver-rt/com/tridium/ndriver/point/BNPointDeviceExt.java:177`).

## Proposed kit deltas (propose-never-apply)
| Δ | Delta | Target file / § | Token |
|---|---|---|---|
| Δ1 | Driver scaffold type + MinimalDriver fixture; the rt part needs alarm-rt, the wb part bajaui-wb | `toolbelt/scaffold-module.sh`; `fixtures/MinimalDriver`; `types/driver-authoring.md` § 12 | `[ev: corpus B1239 §1239.6]` |
| Δ2 | A Context overload of the framework's `doSubmitDiscoveryJob` on `BNPointDeviceExt` is dead code; the Context form works only on an action you declare | `types/driver-authoring.md` § 1.11 | `[ev: corpus B1239 §1239.11]` |
| Δ3 | A device address slot cannot be named `host` (`BISpaceNode.getHost()` clash); the § 5 ping example names placeholders | `types/driver-authoring.md` § 5 | `[ev: corpus B810 §810.2]` |

## Owed (later driver-authoring tasks)
- DR4: verify-module driver mode should also pass the scaffold (lint-driver-scaffold clean, no bare comm config
  slot, `isAlive()` in both discovery hosts); TC-DRV4 pins these structurally today.
- DR5: a `BTestNgStation` loopback test for the scaffold's ping, read, write and timeout paths (clears the
  moduletest WARN), and a live check that the network's Context-form `doSubmitDiscoveryJob` is the bound do
  method (only the binding code was read, no station ran it).
- The scaffold is UDP only. A TCP variant needs a `LinkMessage.receive` framer with bounds checks and keeps
  `serverEnabled` false (§ 1.6, § 1.10).

## Lessons
- A doctrine example that names a slot is not proof the slot can exist: build it once.
- Whether an override is called depends on how Baja binds the action, not on Java overloading.
- A gradle dependency list is only proven by javac; superclass interfaces pull in modules the code never names.

---
**Status**: FOLDED 2026-10-24 (branch `feat/r10-dr2-driver-scaffold`) — Δ1 `toolbelt/scaffold-module.sh`,
`fixtures/MinimalDriver`, `types/driver-authoring.md` § 12; Δ2 § 1.11; Δ3 § 5. INDEX row generated from the header.
