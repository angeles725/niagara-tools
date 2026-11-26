### Added

- `report-module.sh --feature-doc <file>` (#1127): reads the feature document of the change and prints one
  `(module)  ADVISORY  feature-doc` row per skill step it lacks, in this order: the tier line (P0-P3, before the first
  `## ` heading), the `## Behavior decisions` heading with content (a filled table row, or "none") and the
  `## Checks skipped` heading (its text may be "none"). A file that cannot be read is one row saying so. The rows are
  counted as ADVISORY and never change the exit code or the verdict; without the option nothing is read, and `--ready`
  refuses it. In `skill/SKILL.md` those three steps move from UNCHECKED to ADVISORY, and a short rule says to load the
  skill before running a `toolbelt/` script directly. Tests: `feature_doc_test.go`, `feature_doc_unit_test.go`,
  `tests/report-module.bats`. Follow-up: `build.sh` (`go/internal/buildmodule`) does not pass `--feature-doc` yet.
