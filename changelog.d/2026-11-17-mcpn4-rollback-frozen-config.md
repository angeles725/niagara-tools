### Fixed — mcp-n4: n4_rollback restores the configuration of frozen children

- The rollback of an `n4_remove_component` now writes back the configuration of a frozen child (a
  component the station creates with its parent, e.g. an alarm extension's `offnormalAlgorithm` or a
  writable's `proxyExt`): the dry run plans `frozen_config` on the component spec from the station's type
  contracts (type swap plus set ops for the captured configuration slots, never readonly, transient or
  status slots), so the token covers it; a value the station refuses stays in
  `frozen_config_not_restored` with the station's reason in `refused`. The first remove + rollback of a
  freshly alarmed point now brings back the algorithm type and its `highLimit`/`lowLimit` (retro
  2026-11-16 row M18); what the depth-3 snapshot did not capture (`limitEnable` bits) still cannot.
