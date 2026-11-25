### Added — `sweep-fold-audit` reports a folded retro whose body Status still says PENDING

- For each folded INDEX row, `sweep-fold-audit` reads `<kit-root>/retros/<file>`; when its line-1 marker is
  `<!-- review-status: folded -->` and a `**Status**: PENDING` line remains, it prints
  `fold-audit: WARN <file>:<line> header says folded but body Status says PENDING: make it `**Status**: FOLDED …``
  and a `fold-audit: status drift: N retros (header folded, body Status PENDING)` summary. Like every WARN it fails
  `--strict` (CI's step). A retro file that is absent, or a pending retro, is not read, so the output is unchanged for them.
- The 80 retros that carried the drift now read `**Status**: FOLDED …` (the word, and `| pending |` in a quoted INDEX
  row, changed; nothing else in those files). Rule: `build-n4-module-kit/METHODOLOGY.md` ("A retro's header and body
  Status agree"). Pinned by `go/internal/foldaudit/statusdrift_test.go` and `tests/sweep-fold-audit.bats` F25/F26.
