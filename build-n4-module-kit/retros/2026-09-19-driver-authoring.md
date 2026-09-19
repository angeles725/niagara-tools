<!-- review-status: folded -->
# 2026-09-19 · kit · driver-authoring

**Session**: 2026-09-19 kit-improvement campaign (branch odd/apply-kit-candidates-2026-09-19) — WU folding the driver/network cluster.
**Delta count**: 9

## What happened
Driver/network is a whole module class the kit had no recipe for (only folded factory/discovery
notes). Cluster D1-D8 + code-anatomy G3 (BProxyExt SPI) in the master register, reinforced by the
net-new KNX/Z-Wave vendor drivers (DR-01/DR-02).

## Evidence
- BProxyExt SPI + silent-DROP when device down: `[ev: code BProxyExt.java]` `[ev: corpus B810]`
- comm layer / serial: `[ev: corpus B517]`; tuning: `[ev: corpus B872]`; poll buckets: `[ev: corpus B872]`
- virtual points: `[ev: corpus B28]`; KNX group-address + Z-Wave inclusion: `[ev: code BEibnetIpNetwork.java]` `[ev: code BInclusionMonitor.java]`

## Proposed kit deltas (folded in this PR)
| Δ | Delta | Target | Token |
|---|---|---|---|
| Δ1 | Class ladder BDeviceNetwork/BBasicNetwork/BLoadableNetwork/BSerialNetwork | `types/driver-authoring.md` §1 | `[ev: corpus B517]` |
| Δ2 | BProxyExt readSubscribed/readUnsubscribed/write SPI | `types/driver-authoring.md` §2 | `[ev: code BProxyExt.java]` |
| Δ3 | comm layer makeComm + 3 workers | `types/driver-authoring.md` §3 | `[ev: corpus B517]` |
| Δ4 | BTuningPolicy map | `types/driver-authoring.md` §4 | `[ev: corpus B872]` |
| Δ5 | doPing + configFail/configFatal | `types/driver-authoring.md` §5 | `[ev: corpus B810]` |
| Δ6 | BIPollable poll buckets + dibs | `types/driver-authoring.md` §6 | `[ev: corpus B872]` |
| Δ7 | history/schedule device ext, virtual, serial | `types/driver-authoring.md` §7 | `[ev: corpus B28]` |
| Δ8 | KNX group-address model | `types/driver-authoring.md` §8 | `[ev: code BEibnetIpNetwork.java]` |
| Δ9 | Z-Wave inclusion lifecycle + firmware-in-jar | `types/driver-authoring.md` §8 | `[ev: code BInclusionMonitor.java]` |

## Lessons
- The point-level SPI (BProxyExt) is separate from the network/device level; the kit only had the latter.
- Transport shapes the point model: KNX addresses by group-address, Z-Wave by command-class capability.

---
**Status**: FOLDED into `types/driver-authoring.md` (2026-09-19).
