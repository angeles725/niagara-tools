### Fixed — ng-deploy: review advisories T15 and T18 of the 2026-11-15 coordination stretch

- A SIGINT, SIGTERM or SIGHUP that arrives while the probe before the build looks at a station jar it does not
  move (one the station does not have yet) ends the run with 128 + the signal before the build, instead of being
  dropped when the probe stops holding signals.
