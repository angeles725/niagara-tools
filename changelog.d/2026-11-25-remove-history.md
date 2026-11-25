### Added

- mcp-n4 `n4_remove_history` (#992 T19): a guarded write tool that deletes one history and all its records
  through the station's history channel (`deleteHistories`, the call the station's own history client makes),
  so a lab test can clean up the histories it created. It has **no inverse**: the records are gone and
  `n4_rollback` cannot restore them; the plan says so. Planning needs `acknowledge_data_loss: true`.
  The dry run (`history_removal`) lists the history id, its record count and the extensions that record into it.
  It refuses a history no extension records into (its slot path is unknown, so it is outside every write
  scope), an extension outside the write scope, an enabled extension unless `allow_enabled_extension: true`, and
  an extension read cut at its cap. The read-back is `verified` when the history can no longer be read.
  The channel allow-list gains exactly `history: deleteHistories` (not `clearAllRecords` or `clearOldRecords`).
  Tests: `history_remove_test.go`, `plan/channel_test.go`.
