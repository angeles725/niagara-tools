<!-- review-status: pending -->
<!-- module: kit -->
<!-- deltas: 2 -->
# 2026-11-17 · kit · mcpn4-replicate-points

**Session**: a client B field job that replicated the same Modbus points onto many devices with
`n4_copy_component` and `n4_set_slot`, one confirmation token per write.
**Delta count**: 2

## What happened
The job needed the same points on every device of a trunk. `n4_copy_component` copies one child
under one parent, so each device and each address change was its own dry run and token: 630 calls
in 39 minutes for 112 points, and the process write budget ran out before the save. Two of the
replicated writable points carried a non-null fallback (a power point at 0 = OFF, a setpoint at
200): every copy would have commanded its device with it on first communication, and nothing in the
dry run said so.

## Evidence
- Field retro of 2026-11-16 (client B), rows U10 (630 calls / 39 min / 112 points, budget exhausted)
  and U18 (fallback values pending a decision).
- Lab proof on the 4.15.6.20 lab station: see the PR description (batch ids).

## Proposed kit deltas (propose-never-apply)
| Δ | Delta | Target file / § | Token |
|---|---|---|---|
| Δ1 | APPLIED 2026-11-17. `n4_replicate_points`: N x M adds under one token, per-target `copies[]` name/address overrides, one budget write per add, refused up front when over `write_budget.remaining` | `mcp-n4-kit/README.md` write tools; `mcp-n4-kit/skill/SKILL.md` bulk | `[ev: retro mcpn4-replicate-points Δ1]` |
| Δ2 | APPLIED 2026-11-17. Copy/replicate dry runs note writable points with a non-null fallback | `mcp-n4-kit/README.md` `n4_copy_component`; `mcp-n4-kit/skill/SKILL.md` | `[ev: retro mcpn4-replicate-points Δ2]` |
