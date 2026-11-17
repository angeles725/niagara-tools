### Added — build.sh signal handler ordering test (waits for the chain's own restore)

- `TestASignalWaitsForTheChainsOwnRestore` pins that a signal arriving while the build chain restores `modules/`
  or cleans up waits for it before touching anything (a mutation that drops the handler's lock fails it).
