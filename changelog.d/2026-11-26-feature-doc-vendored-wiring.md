### Added
- `build.sh --feature-doc <file>` forwards the option to `report-module`, so the ADVISORY rows for the skill steps the
  feature doc lacks (tier line, `## Behavior decisions`, `## Checks skipped`) show in a build's punch-list; without
  it nothing changes, and a file that cannot be read stays report-module's one ADVISORY row. [ev: issue #1145]
- `report-module` runs `lint-vendored` on a `-ux` part's `src/rc` next to `vendor-floor`: a vendored file at a
  version the kit's known-defects table lists is a WARN row (the verdict changes only as any WARN row does), and an
  rc dir without `vendor/` gets no row. [ev: issue #1145]
