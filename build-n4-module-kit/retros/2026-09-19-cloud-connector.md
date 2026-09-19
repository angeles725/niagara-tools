<!-- review-status: folded -->
# 2026-09-19 · kit · cloud-connector

**Session**: 2026-09-19 kit-improvement campaign (branch odd/apply-kit-candidates-2026-09-19) — WU folding the cloud-connector cluster (vendor mining: cloudLink*).
**Delta count**: 4

## What happened
Cloud/IoT connectivity is a module class the kit did not cover, found in the net-new PowerB
`cloudLink*` family. Cluster CL-01..CL-04 in the master register. SPI details are [INFER] from
class/field signatures (javap + vineflower), not prose comments.

## Evidence
- BCloudConnectionService + 3 static SPI factory maps: `[ev: code cloudLink-rt BCloudConnectionService]`
- transport dual model (okhttp platform vs Qpid fat-jar): `[ev: code cloudLink-rt BAmqpTransport]`
- KeyRing credentials + per-backend KeyRingPermission: `[ev: code cloudLinkAzure-rt module.xml]`

## Proposed kit deltas (folded in this PR)
| Δ | Delta | Target | Token |
|---|---|---|---|
| Δ1 | BCloudConnectionService core + 3 SPI factory maps | `types/cloud-connector.md` §1 | `[ev: code cloudLink-rt module.xml]` |
| Δ2 | transport dual model + store-and-forward | `types/cloud-connector.md` §2 | `[ev: code cloudLink-rt BAmqpTransport]` |
| Δ3 | auth SPI + KeyRing credential pattern | `types/cloud-connector.md` §3 | `[ev: code cloudLinkAzure-rt module.xml]` |
| Δ4 | channel SPI + backend plugin chain + nc SmartTagDictionary | `types/cloud-connector.md` §4-6 | `[ev: code cloudLinkForge-rt module.xml]` |

## Lessons
- Multiple cloud backends plug into one service via static factory-map registries (the extension seam).
- Cloud credentials live in the platform KeyRing (KeyRingPermission), never in component properties.

---
**Status**: FOLDED into `types/cloud-connector.md` (2026-09-19).
