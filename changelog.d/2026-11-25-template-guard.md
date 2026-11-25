### Changed

- mcp-n4 `n4_make_station_template` (#1057): the dry run now refuses by default. Confirmed on a 4.14.0.162
  station, the `MakeStationTemplateJob` zips the station home `^` including `^temp`, where it writes its own
  output, so the `.ntpl` grows without end (66.9 GB after 95 min, about 12 MB/s, from a 68 KB config) and fills
  the host disk; a job cancel only interrupts the thread and the zip copy does not check it. The refusal states
  that cause, the disk risk and the recovery (`n4_save_station`, stop the station process, delete the partial
  `^temp/*.ntpl`, restart the station). New boolean argument `accept_runaway_risk` (default `false`): `true`
  plans the job anyway, and the plan then carries a `RUNAWAY RISK accepted` note with the same warning.
- Read-back of a job still running: besides `unverified`/`running` it reports `temp_output`, the size of
  `^temp/<station>.ntpl` from two reads of the file servlet 5 s apart (`first_bytes`, `second_bytes`,
  `growth_bytes`, `bytes_per_second`, `growing`, and a `RUNAWAY` warning with the recovery when it grows). The
  size is the `Content-Length`; a read that hits its 1 MiB cap without one says "at least N bytes" and claims no
  growth. A file that cannot be read (the station may make the name unique) says where to look in `^temp`.
  A finished job is not sampled.
- The tool description in `registry.json` says all this in one paragraph. Tests: `template_make_guard_test.go`,
  `template_make_readback_test.go`.
