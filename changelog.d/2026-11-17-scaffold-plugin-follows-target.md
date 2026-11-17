### Fixed — scaffold-module picks the gradle plugin of the target release

- Without `--plugin-version`, `scaffold-module.sh` now bakes the Niagara gradle plugin that the `--target-version`
  release ships (4.13 7.3.40, 4.14 7.6.17, 4.15 7.6.22). A 4.15 scaffold used to pin 7.6.17 and failed preflight
  `plugin-pin` on a 4.15 niagara_home.
