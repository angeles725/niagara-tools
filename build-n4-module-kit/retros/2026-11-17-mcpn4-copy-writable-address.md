<!-- review-status: folded -->
<!-- module: kit -->
<!-- deltas: 1 -->
# 2026-11-17 · kit · mcpn4-copy-writable-address

**Session**: a field JACE (4.14, client A's site) where `n4_copy_component` of a writable Modbus point
landed every copy at `Hex:-1` (verdict `mismatch`), reproduced and fixed on the 4.15.6.20 lab station.
**Delta count**: 1

## What happened
Copies of a writable Modbus point (`control:NumericWritable`, `ModbusClientNumericProxyExt`) faulted
with "Illegal Modbus address Hex:-1", while a read-only `control:NumericPoint` copy kept its address.
The first hypothesis (the flags-only `regType` child in the copy body) was falsified on the lab: the
same body with `regType` first or dropped failed the same way.

## Evidence
- Lab station log, one line per copy: "Duplicate Writable point for register address: Modbus:40002".
- Decompiled `BModbusClientProxyExt.started()` / `changed(dataAddress)`: a second writable proxy at
  an address already used on the device is reset to `Hex:-1`.
- Controls: a distinct address (40010) stuck on the copy; the source's address stuck once the source
  was removed.

## Proposed kit deltas (propose-never-apply)
| Δ | Delta | Target file / § | Token |
|---|---|---|---|
| Δ1 | APPLIED 2026-11-17. `n4_copy_component` `copies[]` sets each copy's `proxyExt/dataAddress` in the same add; the plan notes copies that would share a register and the read-back names the cause | `mcp-n4-kit/README.md` (n4_copy_component), `mcp-n4-kit/METHODOLOGY.md` § 3 | `[ev: retro mcpn4-copy-writable-address Δ1]` |
