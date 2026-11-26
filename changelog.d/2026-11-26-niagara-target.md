### Added — the Niagara version of a module's station is recorded with the module

- A module's `build-state.v1` envelope may carry optional `niagara_target: <version>` (2 to 4 dotted numbers, e.g.
  `4.15.6.20`) and `plugin_version: <version>` (e.g. `7.6.22`) fields (issue #1158), so nothing ties a module to
  the wrong Niagara version unnoticed: a client module was built against a 4.14 mirror, signed and released as
  ready to install while its station ran 4.15.6.20. `unknown` or no field records nothing; `sweep-build-state.sh`
  fails (exit 1) an invalid value, the field twice, a mistyped line or two envelopes of one module.
- The record lives with the module: `build.sh` and `ng-deploy.sh` read `<module-root>/BUILD-STATE.md` when that
  file exists (it is then the only file read), else the kit's `BUILD-STATE.src.md` as before; `--build-state <file>`
  still wins. `expected_signer` uses the same lookup.
- `build.sh` exits 10 before gradle when `niagara_home` (for a mirror, also the installation it was made from) is
  another major.minor than `niagara_target`, or when an explicit `--target-version` or `--plugin-version` (or
  `$NIAGARA_PLUGIN_VERSION`) differs from the record. Left out, `--target-version` defaults to the major.minor of
  `niagara_target` (so `verify-module.sh` checks the jars' baja `vendorVersion` floor against the station's line)
  and `--plugin-version` to `plugin_version`. One `INFO` line names the record. A home whose version cannot be read
  is a `WARN`.
- `ng-deploy.sh` exits 55 before the build when the version of `STATION_MODULES_DIR`'s installation is another
  major.minor than `niagara_target`, naming both; an unreadable version is a `WARN` and the existing jar-vs-station
  check still runs. Documented in `build.sh`'s header, `ng-deploy`'s usage, `BUILD-LOOP.md` §4, `BUILD-STATE.src.md`
  and `skill/SKILL.md`.
