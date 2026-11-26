### Added — ng-deploy writes `.deploy-targets`; `build.sh --target` feeds the drift gate

- `ng-deploy.sh --target <name>` (or `DEPLOY_TARGET` in the env file; issue #1139, follow-up of #1114): after a
  deploy that went through, it upserts the line `<name> <MODULE_NAME>@<vendorVersion>=<commit>` in
  `<module-root>/.deploy-targets`, the file `build.sh --target` reads. The version is the -rt jar's, the commit the
  one written to `.last-deploy-sha`, the module root the directory of `GRADLEW_PATH`. The line of the same target and
  module is replaced in place; comments, blank lines and every other line are kept, and the file is replaced
  atomically. There is no default name: without `--target` nothing is written and a closing `record:` line says how
  to. `--no-deploy`, a deploy that stopped, and mode B (no -rt jar) leave the file alone; a line that cannot be
  written (no commit, no readable vendorVersion, an unwritable file) is exit 41 after the copy, and a target name
  with a blank, `#`, `@` or `=` is exit 10 before anything runs. The new `record:` line ends every deploy that
  went through. Version-control access stays in the Go package (`go/internal/ngdeploy`); the script is unchanged.
- `build.sh --target <name>` also feeds the drift gate: changed bytes at the version the record gives for the module
  are exit 51 even when the mirror's jar is this checkout's own build, and at another version one INFO line skips
  the check (no station runs it). The record holds a version, not bytes, so the bytes compared stay the mirror's.
- The scaffold's `.gitignore` files (all four fixtures) list `.build/`, where `build.sh` keeps `mirror-built`, so a
  scaffolded module no longer commits it; `TC-GI1` pins it (and now covers the library type too).
