### Changed — field commissioning notes

- `BUILD-LOOP.md` § 6: save the station and stop running write batches before a module install (the install
  restarts the station).
- `types/driver-authoring.md` § 11.2: the BACnet Discover Networks filter; a point's object instance is not the
  controller terminal. § 11.3: one writable Modbus proxy point per register per device (a duplicate is reset to
  `Hex:-1`); use the stock Modbus Async managers.
