### Added — a module is installed into the Niagara version it was compiled for

- `ng-deploy.sh` exit 55: after the build and the version-bump precheck, a target installation (the `modules/baja.jar`
  of `STATION_MODULES_DIR`) whose Niagara major.minor differs from the one a built jar was compiled against (the highest
  Tridium dependency floor in its `module.xml`) is refused before the schema-risk precheck, the gate and the copy; the
  line names both versions and the installation. A jar with no recorded floor and a `baja.jar` that cannot be read are
  exit 55 too. A station directory with no `baja.jar`, `--no-deploy` and a station that is `<NIAGARA_HOME>/modules` are
  not checked.
- `toolbelt/check-target-version.sh <jar> <niagara_home>` (`go/internal/niagaraversion`): the same comparison for a copy
  made by hand; one PASS or FAIL row with both versions, exit 0 / 1, exit 2 when a version cannot be read.
- `build.sh` against a mirror ends with one line naming the installation the mirror was made from, its Niagara version,
  that the jars built were NOT installed there, and the command that installs them (`modjar.MirrorSource` reads the
  marker's `source=`). Nothing is printed for a real `niagara_home`, a marker without a source or a failed build.

### Changed — a held jar says which station to stop

- The hint after a copy or move a running station refused (`ng-deploy.sh` exit 40 or 10) and the line after `build.sh`
  exit 32 now say to stop the station and close Workbench of that installation first, and that on Windows the station is
  a service or `station.exe`. The deploy text of `build.sh` and `ng-deploy.sh --help` list the new exit and line.
