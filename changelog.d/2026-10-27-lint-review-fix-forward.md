### Fixed — rc-scan map anchors ignore the directories above the module; MM7 count and duplicate concerns

- rc-scan's `fetch-outside-services` exemption, its third-party skip and its vendored-JS skip read the path below
  the artifact dir: a module checked out under a directory named `rc/js/services`, `rc/vendor` or `rc/ext` is no
  longer exempted or skipped as a whole.
- lint-module-map MM7: the row message counts the concern-folder NAMES it decides on (`model/x` + `adapter/x` count
  once) and lists them; a second row with the same concern name (case-insensitive) is its own MM7 row.
