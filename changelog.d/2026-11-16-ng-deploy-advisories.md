### Fixed — ng-deploy: review advisories T15 and T18 of the 2026-11-15 coordination stretch

- A SIGINT, SIGTERM or SIGHUP that arrives while the probe before the build looks at a station jar it does not
  move (one the station does not have yet) ends the run with 128 + the signal before the build, instead of being
  dropped when the probe stops holding signals.
- A `<jar>.ng-probe` in the station that is not a regular file (a directory, a symbolic link) is no longer moved
  to the jar's name as a probe file a killed run left: the run stops with exit 40 naming what it is, nothing changed.
