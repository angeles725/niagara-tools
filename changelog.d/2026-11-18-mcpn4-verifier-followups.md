### Fixed — mcp-n4 link batch recovery and licence check follow-ups

- `n4_create_link` batch form: when the station's replies cannot be trusted and one entry's lookup
  fails, that entry carries `lookup_error` and the other entries keep their inverse, so
  `n4_rollback` still removes the links that were found (it used to journal an empty inverse).
- `n4_create_component`: a device create under a licence with no `device.limit` no longer runs the
  station-wide device count in its dry run.
