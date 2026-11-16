### Fixed — `report-module.sh --ready` review advisories (retro row P4b)

- A module root that cannot be listed is now `report-module: --ready: cannot list <root>: <error>` (exit 3),
  no longer hidden behind "no built jar under ...".
- An artifact of the module (`module-include.xml`, its own `<artifact>.gradle.kts` or a `build.gradle[.kts]`)
  with no built jar is one `FAIL  jar` row naming it, so the module reads NOT READY instead of leaving the
  artifact out; a `docs/` or `gradle/` directory beside the artifacts gets no row.
- A baja floor that cannot be compared with `--expect-niagara` (`4.14-rc` against `4.15`) FAILs the
  `niagara-target` row instead of WARNing, so the jar never reads READY with its target unchecked.
- The lint-ntpl-manifest-deps header and `types/logic-authoring.md` now name the 100,000 entries-read bound of a
  run and its FAIL row next to the 128 MiB bound (retro row T34 doc follow-up).
