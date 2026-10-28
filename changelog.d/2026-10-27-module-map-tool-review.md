### Fixed — module-map tool: retryable add-concern, Tests cells inside the root, station tests by import line

- `module-map.sh add-concern`: a run that fails midway now also removes the directories it created (deepest first,
  never a pre-existing one) and its `MODULE-MAP.md.tmp`, so the same command can be retried; a module whose
  shallowest Java source is in the default package is refused (exit 3) instead of yielding a dot-prefixed package.
- `module-map.sh tests` (and its Tests-cell expansion) refuses an entry that is absolute or leaves `--root` (exit 3),
  as lint-module-map MM3 does; a station test is one with an `import javax.baja…` line, not any file naming
  `javax.baja`; a runner that cannot start or exits other than 0/1 is exit 3.
