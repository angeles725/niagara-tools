### Fixed — ci-local --changed kit-links fails loudly (retro row T32 of the 2026-11-15 coordination stretch)

- The kit-links step of `scripts/ci-local.sh --changed` exits 3 when bats cannot be provisioned at the version
  ci.yml pins, instead of running whatever bats is on PATH: the download error, or one line when the provisioned
  bats is not the pinned version.
- A pinned-tarball download (shellcheck, bats) is bounded by curl `--connect-timeout 10 --max-time 120`, so a
  blackholed network exits 3 instead of hanging.
- A git error while detecting whether the change touches a kit doc is reported (exit 3), never read as no change.
- The generated state of the snapshot is written by the snapshot's own `toolbelt/kit-state.sh`.
