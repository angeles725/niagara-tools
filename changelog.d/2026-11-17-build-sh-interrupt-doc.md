### Changed — build.sh help names what an interrupt does

- `build.sh --help` says that SIGINT/SIGTERM/SIGHUP before a passing chain restore the pre-build `modules/` jars
  (exit 128 + signal) and that the process group must be interrupted, not build.sh's pid alone.
