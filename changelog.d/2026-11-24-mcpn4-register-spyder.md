### Added — n4_register_bacnet_spyder registers a Spyder in one planned batch (issue #932, retro R12 ΔM9; folds #884 and #816)

- New write tool `n4_register_bacnet_spyder(network_ord, name, network_number, mac, instance, discover?, wait_seconds?,
  expected_model?)`: create the device (name stored escaped, `2016` -> `$32016`), set its MS/TP address and `objectId`,
  `writeDeviceInstance`, and (default) the point discovery, as one dry run and one confirmation token. The add is
  `plan.ops`, the steps that need the new device's handle are `plan.follow_up` (hashed into the token and recorded in the
  journal intent). The inverse removes the created device; `writeDeviceInstance` has none and the plan says so.
- Refused before any token: a network that is not a `bacnet:BacnetNetwork`, is disabled or in fault; a licence whose
  `device.limit` the device would exceed; a name already under the network (the refusal gives the escaped BQL form,
  `name = '$2432016'`); an instance another BACnet device holds (one BQL; a BQL that cannot be read, or is cut or past the row cap, refuses with the
  reason: the field write has no inverse); instance 9999, the factory instance.
- The run stops at the first step the station refuses and refuses `writeDeviceInstance` on a device in fault (the
  `n4_invoke_action` rule), with `observed.failed_step` and an `unverified` verdict (never a `mismatch`). The fault rule fails closed: a device status
  that cannot be read is reported as `failed_step` and the field write is not sent. A step that failed with no station
  answer (timeout, dropped connection) is told from a refusal: it may have been applied, the controller may already hold
  the new instance after `writeDeviceInstance`, and the batch is `in_doubt`. The read-back checks the address and
  `objectId`, the controller's own instance, and summarizes the discovery; `expected_model` compares the discovered hardware.
- `n4_rollback` of such a batch removes the device under the same later-work guard as a create. `plan.PlanKeys` and
  `IntentKeys` gain `follow_up`; the fake station's `Respond` hook answers actions the fake does not know.
