<!-- review-status: folded -->
<!-- module: kit -->
<!-- deltas: 7 -->
# 2026-10-24 · kit · r10-dr1-driver-doctrine

**Session**: niagara-research quality round 10, driver-authoring task DR1, branch `docs/r10-dr1-driver-doctrine`
from origin/main
**Delta count**: 7

## What happened
An audit of `/build-n4-module` for driver work found that the launcher never routed a driver build to
`types/driver-authoring.md`, and that the guide lacked the parts a first NDriver build needs: which framework to
pick, how the UDP/TCP/serial comm config is wired, a message-framing skeleton, the discovery job body, how
licence limits fault a point or device, and how to test a driver. DR1 closes those doctrine gaps from three
sources (docSource ndriver-rt / driver-rt / baja, the devkit New Driver Wizard templates, corpus blocks and the
devguide); the scaffold, lints and exemplar are later tasks.

## Evidence
- The Decision Gates table had no driver row `[ev: kit skill/SKILL.md]`.
- The wizard's UDP option declares a bare `BUdpCommConfig`, which keeps the base message factory whose
  `NMessage.fromInputStream` throws: `organized/devkit/devkit-wb/extracted/gradle/ndriver/BNfooNetwork.java.vm:67-72`,
  `organized/docSource/docSource-doc/extracted/ndriver-rt/com/tridium/ndriver/datatypes/BCommConfig.java:413-425`.
- The UDP receiver never calls `LinkMessage.receive`; TCP and serial do:
  `organized/docSource/docSource-doc/extracted/ndriver-rt/com/tridium/ndriver/comm/udp/UdpLinkLayer.java:193-208`.
- Driver licence limits are checked only when `getLicenseFeature()` is overridden:
  `organized/docSource/docSource-doc/extracted/driver-rt/javax/baja/driver/BDeviceNetwork.java:581-589`.
- `BIpNetwork`, named by the audit, does not exist in the corpus; the DDF class is
  `organized/devIpDriver/devIpDriver-rt/vineflower/com/tridium/ddfIp/udp/BDdfUdpNetwork.java`.
- The devkit ships no NDriver test template; its only driver test template is
  `organized/devkit/devkit-wb/extracted/gradle/videodriver/BNfooVideoDriverTest.java.vm`.

## Proposed kit deltas (propose-never-apply)
| Δ | Delta | Target file / § | Token |
|---|---|---|---|
| Δ1 | Decision Gates row for driver / device-network builds → `types/driver-authoring.md`; corpus-index row for B877 | `skill/SKILL.md` § Decision Gates; `corpus-index.md` | `[ev: corpus B877]` |
| Δ2 | Framework decision table: NDriver vs basicDriver vs DDF vs raw `javax.baja.driver` | `types/driver-authoring.md` § 1.8 | `[ev: corpus B877 §877.6]` |
| Δ3 | Comm config recipe: local-side address, peer on each message, UDP datagram framing, subclass the config, sender check, DNS per send | `types/driver-authoring.md` § 1.9 | `[ev: corpus B877 §877.3]` |
| Δ4 | NMessage / LinkMessage / factory skeleton, stream framing in `receive`, correlation, unsolicited listener | `types/driver-authoring.md` § 1.10 | `[ev: corpus B1239 §1239.6]` |
| Δ5 | Discovery job body over `BNDiscoveryJob`: cancel check, progress, `discoverFail`, leaves, multi-step | `types/driver-authoring.md` § 1.11 | `[ev: corpus B1239 §1239.11]` |
| Δ6 | Licensing facts: start-time counts, `getLicenseFeature`, `device.limit` / `point.limit`, fault causes, restart | `types/driver-authoring.md` § 5 | `[ev: corpus B14]` |
| Δ7 | Testing section: Baja-free codec in pure JUnit, comm stack against a loopback mock in `BTestNgStation`; two moduleTest tier rows | `types/driver-authoring.md` § 12; `types/moduleTest.md` | `[ev: corpus B1239 §1239.4]` |

## Owed (later driver-authoring tasks)
- DR2: `scaffold-module.sh --type driver` must generate a `BUdpCommConfig` / `BTcpCommConfig` SUBCLASS with
  its own message and link-message factories, a Baja-free codec class, a `srcTest` codec test, and a
  `doSubmitDiscoveryJob` that passes the user `Context`.
- DR4: verify-module driver mode should flag a bare `BUdpCommConfig` property and a `getDiscoveryObjects` with
  no `isAlive()` check, next to the planned `serverEnabled` and enum-range checks.
- DR5: the exemplar rebuild proves § 12 with a real `BTestNgStation` loopback test, and records whether an
  `NMessage` subclass loads in a WSL JUnit run (§ 12 marks it `[INFER]`).

## Lessons
- A wizard option is not a working configuration: the UDP choice compiles and fails on the first datagram.
- "Address" on an NDriver comm config is the local side; the peer belongs to the device and each message.
- A licence check that depends on an overridable hook is off until the hook is written; say which hook.

---
**Status**: FOLDED 2026-10-24 (branch `docs/r10-dr1-driver-doctrine`) — Δ1 `skill/SKILL.md` + `corpus-index.md`,
Δ2-Δ7 `types/driver-authoring.md` (§ 1.8-§ 1.11, § 5, § 12) and `types/moduleTest.md`. INDEX row generated from
the header.
