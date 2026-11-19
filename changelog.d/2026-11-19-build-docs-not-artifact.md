### Fixed — build.sh and report-module skip a module's docs/ folder; a FAILed report restores the mirror jar

- An artifact is a directory with `module-include.xml`, or a `<MOD>-rt/-ux/-wb/-se/-doc` part with its build file,
  for report-module (lint run and `--ready`) and build.sh (`--schema-baseline-ref`, the INCOMPLETE pointer) alike.
  A `docs/` folder beside the parts gets one INFO line instead of a schema-risk ERROR (build exit 10) or an
  INCOMPLETE verdict (build exit 53).
- When the report-module punch-list FAILs (build exit 50), build.sh puts the pre-build jar back into the mirror's
  `modules/`, as a failed verify gate does, so a fix rebuilt at the same version is not reported as drift (exit 51).
