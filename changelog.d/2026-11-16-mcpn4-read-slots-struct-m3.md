### Fixed — mcp-n4: `n4_read_slots` reads a struct slot's own fields and decodes AlarmTimestamps (retro row M3)

- A BOX load answers with the nearest component, so `n4_read_slots` on a property path such as
  `.../OutOfRangeAlarmExt/toOffnormalTimes` listed the extension's own slots (live on a lab station). An ORD
  whose trailing segments name a property of the loaded component (a run of loaded or declared-default
  nodes without a handle, confirmed by one extra load of the component that must answer the same handle)
  now lists that property's fields: loaded ones, then its declared defaults from the type contract
  (every field is marked `default` when the property itself is at its default). A property with no fields
  (a simple) is listed itself. The reply adds `component_ord`, `property` and `property_type`.
- An `alarm:AlarmTimestamps` (`BAlarmTimestamps`: `alarmTime`, `ackTime`, `normalTime`, `count`) adds
  `alarm_timestamps {alarm_time, ack_time, normal_time, count}`: times as the station's ISO text with its
  offset, `null` when unset (`BAbsTime.NULL`, epoch 0), `count` as an integer. Component reads are unchanged.
- The test station now loads the nearest component for a property ORD, as N4 does, and can hold a frozen
  property at its default that resolves but is never sent.
