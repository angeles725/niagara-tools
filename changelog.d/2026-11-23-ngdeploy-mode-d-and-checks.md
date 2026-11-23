### Added — ng-deploy mode D and --plugin-version

- `scripts/ng-deploy.sh --mode D` deploys the `-rt` and `-wb` jars of a module with a workbench part (the `driver` scaffold's pair): the same build, backup, type-count verify, gate and all-or-nothing copy as mode A, with `EXPECTED_WB_TYPES` in the env file. Mode C still copies `-rt` only.
- `scripts/ng-deploy.sh --plugin-version <major>.<minor>.<patch>` forwards `-PniagaraPluginVersion` to every gradle call, as `build.sh` does; another shape is exit 10.

### Changed — four tools say more or refuse more

- `scripts/ng-deploy.sh`: the exit list of `--help` now names code 1 (the backup directory could not be made); the invalid-mode messages read "A, B, C, D".
- `toolbelt/preflight.sh`: a `plugin-pin` FAIL names the plugin versions the installation ships and the option that passes one; the version-drift walk stops at the filesystem root, the temporary directories and the home directory, and skips a directory it cannot read without a `find:` line (it took about 28 s under a temporary directory).
- `toolbelt/report-module.sh`: given a gradle root (a directory whose children hold the module parts) it exits 3 naming `<root>/<MOD>` instead of reporting INCOMPLETE; a verify-module row whose jar path holds a space keeps it out of the detail.
- `toolbelt/sweep-fold-audit.sh --strict`: a pending retro dated since `--deltas-since` whose every delta is cited now fails with `pending, but all N deltas are cited`.
