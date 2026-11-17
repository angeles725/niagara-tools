### Fixed — build and deploy safety gates

- **Behaviour change.** `toolbelt/build.sh`: a SIGINT, SIGTERM or SIGHUP between the baseline snapshot and a passing
  chain puts the pre-build `modules/` jars back (and removes a first-deploy jar) before the temporary directory is
  removed, with `build.sh: interrupted (<signal>) before the gate passed; …`; the exit stays 128 + the signal.
- **Behaviour change.** `scripts/ng-deploy.sh`: `--no-deploy` (or `NO_DEPLOY=1`) when `STATION_MODULES_DIR` is
  `<NIAGARA_HOME>/modules` is exit 10 before the backup and before gradle (`--no-deploy refused: …`).
- **Behaviour change.** `scripts/ng-deploy.sh`: `MODE=B` in the env file is exit 10 with file and line unless
  `--mode` is given, and `VERIFY_MODULE_BIN` unless `--no-gate` is given, as `NO_GATE=1` already was; the
  environment of the command still sets `VERIFY_MODULE_BIN`.
