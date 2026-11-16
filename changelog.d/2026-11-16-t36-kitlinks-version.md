### Fixed — ci-local --changed kit-links runs the bats it checked (retro row T36 of the 2026-11-15 coordination stretch)

- The kit-links step of `scripts/ci-local.sh --changed` runs the bats binary whose version it compared with the
  ci.yml pin, by its absolute path, so a relative PATH entry cannot swap in another bats inside the snapshot.
- A bats version mismatch names the pinned version, the version found and the binary's path.
- A pinned-tarball download (shellcheck, bats) starts no retry after 60 s (`--retry-max-time 60`), so a blackholed
  network exits 3 within about 3 minutes instead of up to ~8.
