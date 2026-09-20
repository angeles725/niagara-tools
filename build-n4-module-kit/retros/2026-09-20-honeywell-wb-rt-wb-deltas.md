<!-- review-status: pending -->
# 2026-09-20 · kit · honeywell-wb-rt-wb-deltas

**Session**: niagara-research focus `honeywell-wb` (B1077–B1083) — how Honeywell's own N4 modules (Spyder/Centraline/Galileo/EagleHawk/PlantController/cloudLinkSbp) build RT+WB. Synthesis B1083 recorded 20 proposed deltas; the 8 highest-value are folded here for build-n4-module, plus 5 anti-patterns to avoid.

**Delta count**: 8

## What happened
Surveyed 6 Honeywell WB module families and classified them into 3 tiers (manager-extension / HMI-operator / PX-binding). Extracted the reusable RT↔WB patterns Honeywell uses on top of Tridium's base, and the recurring structural anti-patterns. Proposed as kit deltas — this retro does not edit the kit.

## Evidence
- B1077 (honeywellDeviceManager-wb SPI + per-user prefs), B1078 (BACnet/Modbus managers + static State), B1079 (galileoKitPx PIN/pin-slots), B1080 (PlantControllerHMI chunked-OTA), B1081 (EagleHawkHMI), B1082 (cloudLinkHonSbp RT-only), B1083 (synthesis). `[ev: corpus B1077-B1083]`

## Proposed kit deltas (propose-never-apply) — full text in corpus B1083 §5
| Δ | Delta | Target file / § | Token |
|---|---|---|---|
| Δ1 | Plugin-extensible multi-device-type manager via an SPI (`BIHonDeviceModel` + `NiagaraRegistryUtil.getImplementersOfTypeSpec()`) — one manager base, many device types register | `types/driver-authoring.md` / `types/wb-widgets.md` | `[ev: corpus B1077]` |
| Δ2 | Persist WB user preferences as per-user JSON under `userHome/` (no rt slots needed) — e.g. remembered columns/commands per `<username>` | `types/wb-widgets.md` | `[ev: corpus B1077]` |
| Δ3 | Fail-closed authorization default: a missing authorization mixin/PIN → the control is DISABLED + HIDDEN (never open-by-default) | `types/security.md` / `types/wb-widgets.md` | `[ev: corpus B1079]` |
| Δ4 | Multi-level user authorization on UI bindings via integer `visibilityPin`/`actionPin` slots (default -1) — reusable for PX widgets and manager rows | `types/wb-widgets.md` | `[ev: corpus B1079]` |
| Δ5 | Chunked Base64 transfer (≈5000-byte segments) for firmware/file OTA when rt actions accept only small strings | `types/logic-authoring.md` / `types/distribution.md` | `[ev: corpus B1080]` |
| Δ6 | `AtomicBoolean` single-run guard on async background jobs triggered from the UI — prevents duplicate submissions | `types/logic-authoring.md` / `types/observability.md` | `[ev: corpus B1082]` |
| Δ7 | Redact `Authorization`/credential headers in structured logging (even at FINEST) — cloud/http connector hardening | `types/security.md` / `types/observability.md` | `[ev: corpus B1082]` |
| Δ8 | `static State extends DeviceState` (class-scoped) to survive a manager view close/reopen without losing subscription/discovery state | `types/wb-widgets.md` | `[ev: corpus B1078]` |

## Lessons
- Honeywell's WB is Tridium's base + a thin extension layer; the reusable wins are the SPI-extensible manager (Δ1), fail-closed auth (Δ3), and per-user prefs (Δ2).
- **Anti-patterns to AVOID (B1083 §4):** static class-level widget singletons (not thread-safe across two open managers); unconditional `(BBacnetDevice)` cast (ClassCastException on a Modbus device); dual-namespace in one module; Modbus inner-class manager without justification; mixed pre-annotation + `@AgentOn` registration eras. Candidate lints.
- These pair with the wb-vendor-ux deltas (PD-01..PD-25) — same RT↔WB vocabulary, Honeywell-specific refinements.

---
**Status**: PENDING — 8 folded of 20 recorded; full set + 5 anti-patterns in corpus B1083.
