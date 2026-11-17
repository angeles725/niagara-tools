### Fixed — report-module --ready: non-part gradle directories and the unbuilt artifact line

- `report-module --ready` no longer reports a gradle directory beside the module parts (`buildSrc/`, a shared
  conventions build) as an unbuilt artifact: an artifact is a directory with `module-include.xml`, or a
  `<MOD>-rt/-ux/-wb/-se/-doc` directory with its gradle build file.
- The NOT READY line of an artifact with no built jar says `(not built)` instead of a `?` version.
