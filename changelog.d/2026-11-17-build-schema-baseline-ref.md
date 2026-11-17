### Added — build.sh takes the schema-risk baseline from a git ref

- `build.sh --schema-baseline-ref <deployed-ref>` runs the archive command the schema-risk SKIP row prints, for the
  run: before gradle, every artifact under `<module-root>/<MOD>` is extracted as it is at the ref into a temporary
  directory, `<artifact>/.deploy-baseline` links to it, and both are removed when the run ends. An already-deployed
  module whose deployed jars are not at hand no longer ends at exit 53 after a manual SAFE run.
- Exit 2 before gradle for a ref that is not a commit, a module directory absent at the ref, an artifact that already
  has a `.deploy-baseline` (never moved or overwritten; a dangling link a killed run left is removed with an INFO line), or `--first-deploy`; an artifact new since the ref gets an
  INFO line and keeps its SKIP row. The INCOMPLETE line now names the option.
