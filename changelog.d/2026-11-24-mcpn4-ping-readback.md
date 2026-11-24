### Changed — n4_invoke_action ping reads the health back (issue #931, retro R12 ΔM8)

- `n4_invoke_action` `ping` no longer ends `unverified` with empty `accepted`/`observed`. Before the ping it reads the
  status and `health` (`lastOkTime`, `lastFailTime`, `lastFailCause`) of the target device (a point device ext: its
  device; a network: every device under it, one BQL), and again after it. `observed.devices[]` holds each device's
  `before`, `after` and `changed` (the changed devices first), with `devices_total` and `devices_changed`.
- Verdict: `verified` when a device's status or health moved; `mismatch` when none did, with `observed.note` naming why
  (the network is disabled, the device is disabled or down, or the check may still be running: read the health again);
  `unverified` only when the health could not be read (`observed.hint` says which read failed). A ping on a device
  under a disabled network leaves the health as it was, so it now reports `mismatch` and names the network.
- The plan, its notes and the confirmation token are unchanged; the dry run and the executing call each read the
  health once, so a ping costs one extra read per call (a network: one BQL). The fake station accepts `ping`.
