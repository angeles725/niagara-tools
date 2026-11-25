### Added — n4_invoke_action loadControlProgram loads a Spyder's ControlProgram (issue #819)

- New action `loadControlProgram` (only on a `honeywellBacnetSpyder:BacnetSpyder`, no arg), the load step for the
  `ControlProgram` the station unloads on its own after a download; a write below it fails with only the path until it is
  loaded again. Dry run notes: no inverse, may fail with `HeapLimitReached` (heap past the configured limit), an already
  loaded program returns no job.
- The read-back waits for the `LoadJob` when the station returns one (the compile/download job wait) and reports
  `logic_loaded` (the `logicLoaded` slot) and `control_program` (the child): `verified` when loaded, `mismatch` when the
  job failed or ended without the program, `unverified` when the job is still running or no job came back with nothing
  loaded.

### Changed

- The error naming an unloaded `ControlProgram` now says to run `n4_invoke_action loadControlProgram` on the Spyder
  instead of opening the program in Workbench.
