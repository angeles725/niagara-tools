### Fixed — ng-deploy: retro rows D1-D4 and D7-D9 of the 2026-11-15 coordination stretch

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
- Before the build, each station jar the run will replace is renamed to `<jar>.ng-probe` and straight back: a jar a
  running station holds is exit 40 with the line above and no gradle call, instead of after a minute of building. A
  jar without write permission still deploys (a rename needs the directory's permission). A jar that cannot be moved
  back is exit 40 naming both paths and the rename to make; a `.ng-probe` file already there is never written over.
- The backup and its purge run just before the copy, after the build and every check (with `--no-deploy`, after the
  build): a run that fails at the build or a check leaves `_backups/` as it was, instead of one more archive of the
  same jars per refused run. Exit 21 still means nothing was copied. When `STATION_MODULES_DIR` is
  `<NIAGARA_HOME>/modules` the backup holds the jars that were there before the build, under their jar names. The
  `--no-backup` warning is printed where it was, before the build.
