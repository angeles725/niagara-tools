### Changed — LP1 lab restart proof traps

- `build-n4-module-kit/build-verify.md` § Lab restart proof (LP1): set two holds far apart so the proof can tell
  which one held an output; grep the module's `config.bog` type prefix in the cleanup check, not its name; a
  remove plus `n4_rollback` creates a new instance and is not a restart; touch a jar swapped under the same file
  name so the registry rebuilds.
