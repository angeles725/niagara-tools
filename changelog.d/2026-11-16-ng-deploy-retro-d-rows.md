### Fixed — ng-deploy: retro rows D1-D4 and D7-D9 of the 2026-11-15 coordination stretch

- A jar that cannot be replaced or moved (exit 40 in the station, exit 10 in `<NIAGARA_HOME>/modules`) with `Device
  or resource busy` or `Text file busy`, or with `Permission denied` or `Operation not permitted` in a directory this
  user may write (how a WSL mount of a Windows drive answers for a jar a running station holds open), is followed by
  `a running station or Workbench holds its module jars open; stop it and run again`. The same errors in a directory
  this user may not write get a line about its permissions instead.
- A directory or a dangling link under the name of a jar of the mode in `<NIAGARA_HOME>/modules` is refused before the
  build (exit 40, `build refused for <jar>`) in every case, not only when the station is that directory: the put-back
  could not rename over it.
- After a deploy that stopped, the NOTICE about `<NIAGARA_HOME>/modules` says it holds the jars it held before the
  build only when every jar was put back or removed; a jar of another origin left in place gets a NOTICE that says so.
- A deploy from a working directory that is not a git repository ends 0 with `NOTICE: not a git repository; the
  deployed commit is not recorded in .last-deploy-sha: the next annotation check has no starting point` instead of an
  ERROR and exit 41. Only git's "not a git repository" (read in the C locale) with no `.git` there or above counts;
  a repository git refuses to open (dubious ownership, a corrupt `.git`) or whose HEAD cannot be read is still exit 41.
- Before the build, each station jar the run will replace is renamed to `<jar>.ng-probe` and straight back: a jar a
  running station holds is exit 40 with the line above and no gradle call, instead of after a minute of building. A
  jar without write permission still deploys (a rename needs the directory's permission). A jar that cannot be moved
  back is exit 40 naming both paths and the rename to make. SIGINT, SIGTERM and SIGHUP are held across the two
  moves. A `.ng-probe` file a killed run left is moved back when its jar is missing (NOTICE), and refused with the
  manual fix (exit 40) when the jar is beside it; it is never written over.
- The backup and its purge run just before the copy, after the build and every check (with `--no-deploy`, after the
  build): a run that fails at the build or a check leaves `_backups/` as it was, instead of one more archive of the
  same jars per refused run. Exit 21 still means nothing was copied. When `STATION_MODULES_DIR` is
  `<NIAGARA_HOME>/modules` the backup holds the jars that were there before the build, under their jar names; only this
  module's own jars are renamed or left out (another module's file aside is archived as it is). The
  `--no-backup` warning is printed where it was, before the build.
- A mirror `NIAGARA_HOME` gets the isolation `build.sh` applies, from one shared rule (`go/internal/gradleenv`, which
  `build.sh`'s `isolateMirror` now calls too): `JAVA_TOOL_OPTIONS += -Duser.home=<mirror>/.mirror-user-home` (and the
  real maven repository), `GRADLE_USER_HOME` pinned and `--no-daemon` on every gradlew call, so the signing plugin
  cannot read the real `~/.tridium` keystore. A mirror path or `$HOME` with whitespace is exit 10 before gradle.
- `ng-deploy --gradle-prop k=v` (repeatable) is forwarded to gradlew as `-Pk=v`, as `build.sh --gradle-prop` is, so
  `niagaraSigningAlias` and `niagaraSigningProfile` no longer need `ORG_GRADLE_PROJECT_*` set by hand; the gate expects
  the last `niagaraSigningAlias` given. `build.sh` is unchanged: same gradle arguments and environment.
