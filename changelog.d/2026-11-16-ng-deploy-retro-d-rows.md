### Fixed — ng-deploy: a jar held open, a directory in the home, the home NOTICE, a module outside git

- A jar that cannot be replaced or moved with `Permission denied`, `Operation not permitted` or `Device or resource
  busy` (exit 40 in the station, exit 10 in `<NIAGARA_HOME>/modules`) is followed by the line `a running station or
  Workbench holds its module jars open; stop it and run again`: Windows keeps a running station's module jars open.
- A directory or a dangling link under the name of a jar of the mode in `<NIAGARA_HOME>/modules` is refused before the
  build (exit 40, `build refused for <jar>`) in every case, not only when the station is that directory: the put-back
  could not rename over it.
- After a deploy that stopped, the NOTICE about `<NIAGARA_HOME>/modules` says it holds the jars it held before the
  build only when every jar was put back or removed; a jar of another origin left in place gets a NOTICE that says so.
- A deploy from a working directory that is not a git repository ends 0 with `NOTICE: not a git repository; the
  deployed commit is not recorded in .last-deploy-sha: the next annotation check has no starting point` instead of an
  ERROR and exit 41. A repository whose HEAD cannot be read is still exit 41.
