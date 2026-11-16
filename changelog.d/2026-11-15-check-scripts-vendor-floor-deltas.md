### Fixed — check scripts and lint-vendor-floor fail closed on the paths their ports kept open

- `scripts/check-installed-line.sh` reads git's file names NUL-separated: a group whose path git prints in quotes (a
  non-ASCII directory) is compared like any other. Before, such a group was skipped, so its downgrade was reported
  by no row and the exit stayed 0. Retro go-check-scripts-ports Δ1.
- `scripts/check-client-source.sh --build-state FILE`: a file that exists and cannot be read is an environment fault,
  exit 3 with `check-client-source: cannot read FILE: <reason>` and no row. Before, the same stderr line was followed
  by `WARN  source-of-truth  source_of_truth undeclared` and exit 0. Retro go-check-scripts-ports Δ2.
