### Changed — report-module runs the course-coverage run B lints

- `report-module` runs `lint-lexicon-apostrophe` (per artifact, after `lint-lexicon-ascii`), `lint-lifecycle-guard`
  and `lint-intrinsic-lock` (per artifact `src/`, after `lint-slot-default-type`); WARN rows relayed, no verdict row.
