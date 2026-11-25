### Fixed — n4_invoke_action active/inactive send the default override; a stated link refusal is journaled as refused (issue #985 Δ4, Δ5)

- `n4_invoke_action` `active` and `inactive` on a `control:BooleanWritable` now send the argument the station's
  action takes: a `control:Override` with `duration` 0 (`BBooleanWritable` `active`/`inactive` have
  `parameterType BOverride`; `doActive`/`doInactive` read `override.getDuration()`, so a call without one failed on the
  station as a bare "station error" and left the batch in-doubt). A duration of 0 is the station's own default: the
  point applies its `maxOverrideDuration`, and the override has no expiry when that is 0 too. `auto` takes no argument
  and still sends none; `arg`/`arg_type` on the three state actions are still refused. The tool description says so.
- A first op the station refuses with its own `checkLinks` text that states nothing was applied (`Could not create
  link: …`, e.g. a readonly target property such as `in8`; `Could not find Source Slot: …`, `Could not find Target Slot:
  …`, `Could not find Source: …`, `Could not find Target: …`) is journaled as refused and answered as a tool error with
  the station's text, even when the read-back of the target cannot be made (before, only a successful read-back showing
  no change gave that; otherwise the batch stayed in-doubt). The station throws these while it only checks, before it
  adds the first link. A bare "station error", a transport failure and the same text on another op stay in-doubt.
- Tests: `TestActiveAndInactiveSendTheDefaultOverrideArgument`, `TestAutoStaysArgumentLessAndAnArgIsStillRefused`
  (`writeops`), `TestAStatedLinkRefusalIsRefusedEvenWhenTheReadbackFails`,
  `TestOnlyTheStationsStatedRefusalSkipsTheReadback` (`plan`); the plan differential declares the added argument.
