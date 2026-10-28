### Fixed — ng-deploy: every check runs before the copy

- `scripts/ng-deploy.sh` runs the type-count verify, the `BUILD_ID` cache-buster check and the verify-module gate
  on the jars in `build/libs` BEFORE copying them to `STATION_MODULES_DIR` (the prechecks already ran there). A
  failing check (exit 12, 50, 51, 52, 53 or 54) now leaves the station's modules dir untouched; before, the verify
  and the gate ran after the copy, so an exit 50 still left the rejected jar installed.
