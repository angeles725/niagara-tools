### Changed

- `build.sh` (#1114): the mirror (`<niagara_home>/modules`) is a build sandbox, not a station, and the
  `--schema-baseline-ref` version WARN no longer calls it "deployed". It now says "the mirror holds <MOD>-rt X (a
  build, not necessarily what a station runs); deployed version unknown" and asks to confirm the ref.
- New `build.sh --target <name>`: the deployed ref is read from `<module-root>/.deploy-targets`, one line
  `<target> <MOD>@<vendorVersion>=<commit>` per target and module. The commit is the schema-risk baseline (as
  `--schema-baseline-ref` would be) and its vendorVersion is checked against the defaultModuleVersion at that commit.
  A target with no line for the module is exit 2 before gradle; it excludes `--schema-baseline-ref` and
  `--first-deploy`. Writing `.deploy-targets` from `ng-deploy.sh` is a follow-up.
- `build.sh` drift check (#1114): a passing mirror build records `<jar file name> <sha256>` per installed jar in
  `<module-root>/.build/mirror-built` (never in `modules/`; gitignore `.build/`). A rebuild of an unreleased version
  whose previous mirror jar has the recorded digest is this checkout's own build: one INFO line and no exit 51 instead
  of `--no-drift-check`. No record, an unreadable one or a different digest keeps exit 51.
