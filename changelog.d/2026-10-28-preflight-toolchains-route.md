### Fixed — `preflight.sh --toolchains` says which route it checked

- The `toolchains` row checks the WSL/POSIX `gradlew` route only; a root with a `gradlew.bat` gets a SKIP row saying
  the Windows route is not checked, instead of a PASS read as covering it.
- A `gradle.properties` pin of a full JDK 8 passes even when the root has no executable `gradlew`.
