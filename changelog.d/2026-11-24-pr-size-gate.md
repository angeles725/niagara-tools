### Added — a pull request over 400 changed lines fails CI unless it carries `size:exception`

- `.github/workflows/pr-size.yml` (on `pull_request`: opened, synchronize, reopened, labeled, unlabeled; separate from
  `ci.yml`) runs `scripts/pr-size.sh <base> <head>`: additions + deletions of the merge-base diff, printed per file with
  the total, exit 1 over the limit (`PR_SIZE_LIMIT`, default 400; exit 3 for a bad limit, an unknown ref or no
  repository). `go.sum`, `*.lock`, `package-lock.json`, `testdata/**` and `vendor/**` are listed as excluded and not
  counted; a `changelog.d/` fragment counts; a pure rename counts 0.
- A PR carrying the `size:exception` label passes the check with a warning; adding or removing the label re-runs it.
  Documented in `build-n4-module-kit/METHODOLOGY.md`. Pinned by `tests/pr-size.bats`. Before, the 400-line guidance was
  advice only.
