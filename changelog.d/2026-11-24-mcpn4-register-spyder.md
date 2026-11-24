### Added — n4_register_bacnet_spyder registers a Spyder in one planned batch (issue #932, retro R12 ΔM9; folds #884 and #816)

- New write tool `n4_register_bacnet_spyder(network_ord, name, network_number, mac, instance, discover?, wait_seconds?,
  expected_model?)`: create the device (name stored escaped, `2016` -> `$32016`), set its MS/TP address and `objectId`,
  `writeDeviceInstance`, and (default) the point discovery, as one dry run and one confirmation token. The add is
  `plan.ops`, the steps that need the new device's handle are `plan.follow_up` (hashed into the token and recorded in the
  journal intent). The inverse removes the created device; `writeDeviceInstance` has none and the plan says so.
- Refused before any token: a network that is not a `bacnet:BacnetNetwork`, is disabled or in fault; a licence whose
  `device.limit` the device would exceed; a name already under the network (the refusal gives the escaped BQL form,
  `name = '$2432016'`); an instance another BACnet device holds (one BQL); instance 9999, the factory instance.
- The run stops at the first step the station refuses and refuses `writeDeviceInstance` on a device in fault (the
  `n4_invoke_action` rule), with `observed.failed_step` and an `unverified` verdict. The read-back checks the address and
  `objectId`, the controller's own instance, and summarizes the discovery; `expected_model` compares the discovered hardware.
- `n4_rollback` of such a batch removes the device under the same later-work guard as a create. `plan.PlanKeys` and
  `IntentKeys` gain `follow_up`; the fake station's `Respond` hook answers actions the fake does not know.
