### Added — mcp-n4: BACnet discovery and Honeywell Spyder commissioning over BOX

- `n4_rename_component` renames up to 50 dynamic slots under one dry run and token (RenameOp), with an
  inverse and a read-back.
- `n4_discover_devices` runs the network's Device Manager discovery (Who-Is, "Send Global") and lists each
  answer with `mac_display` exactly as Workbench's MAC Addr column prints it (`BBacnetAddress.bytesToString`:
  MS/TP in decimal, BACnet/IP port in hex) and `objects` (`listSize`), marked `registered`, `new` or
  `address_differs`; registered devices that stayed silent go under `not_answering`. Default wait 20 s
  (Workbench's 10 s missed MS/TP Spyders).
- `n4_discover_points` runs a device's point discovery job and lists every object with name, value and
  description, the station point that already proxies it, and for a Spyder a `hardware` block (hw id from
  `analogValue:0`, model, I/O).
- `n4_invoke_action` accepts, on a `honeywellBacnetSpyder:BacnetSpyder` only: `writeDeviceInstance` (read-back
  asks the controller its instance), `modelChange` (arg = model display name; inverse back to the previous
  model; read-back checks the model and `compiledFwModel`), `compile` and `spyderDownload` (arg `full` or
  `quick`; the read-back waits for the job and returns its state and log, and flags a download that skipped
  the compile step).
- The server reconnects once when the station drops the BOX session ("Could not find Server Session"): a
  read or dry run is retried, an execution is not (its token died with the session) and says to redo the
  dry run.

### Fixed — mcp-n4: BACnet octet strings are read back by their octets

- A `BacnetOctetString` slot (a MAC) is compared by its octets, so a write of the same address no longer
  reads back as a mismatch.
