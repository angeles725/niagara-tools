<!-- review-status: folded -->
<!-- module: kit -->
<!-- deltas: 3 -->
# 2026-11-17 · kit · field-commissioning-notes

**Session**: deep retro of a client B field job (a JACE with DIP-addressed MS/TP controllers and Modbus RTU
split units) plus a lab reproduction of a copy defect: three commissioning rules the kit did not carry.
**Delta count**: 3

## What happened
- A module install restarted the field JACE while an agent was creating 105 links; the station went away at
  link 77 and the unsaved part had to be redone.
- Discover on the controllers' MS/TP trunk found nothing for 24 minutes because only the first network was
  selected; a terminal was mapped from the BACnet object instance number.
- Copies of a writable Modbus point landed at `Hex:-1`. A lab reproduction on Niagara 4.15.6.20 showed the
  station resets any second writable point on the same register of a device ("Duplicate Writable point for
  register address"); read-only copies kept their address. A vendor point manager threw on a Modbus device
  without the vendor's device config.

## Evidence
- Field retro of 2026-11-16 (client B), rows U19, U21 and the copy finding of row U3; lab batches of the
  reproduction are recorded in the research repository.
- `modbusCore` `BModbusClientProxyExt` `started()` / `changed()` → `writablePointAlreadyExists` (decompiled source).

## Proposed kit deltas (propose-never-apply)
| Δ | Delta | Target file / § | Token |
|---|---|---|---|
| Δ1 | APPLIED 2026-11-17. Save and stop running write batches before a module install (it restarts the station) | `BUILD-LOOP.md` § 6 | `[ev: retro field-commissioning-notes Δ1]` |
| Δ2 | APPLIED 2026-11-17. BACnet Discover Networks filter; object instance is not the terminal number | `types/driver-authoring.md` § 11.2 | `[ev: retro field-commissioning-notes Δ2]` |
| Δ3 | APPLIED 2026-11-17. One writable Modbus proxy per register per device (duplicate → `Hex:-1`); use the stock Modbus Async managers | `types/driver-authoring.md` § 11.3 | `[ev: retro field-commissioning-notes Δ3]` |
