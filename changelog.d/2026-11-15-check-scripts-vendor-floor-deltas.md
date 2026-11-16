### Fixed — check scripts and lint-vendor-floor fail closed on the paths their ports kept open

- `scripts/check-installed-line.sh` reads git's file names NUL-separated: a group whose path git prints in quotes (a
  non-ASCII directory) is compared like any other. Before, such a group was skipped, so its downgrade was reported
  by no row and the exit stayed 0. Retro go-check-scripts-ports Δ1.
- `scripts/check-client-source.sh --build-state FILE`: a file that exists and cannot be read is an environment fault,
  exit 3 with `check-client-source: cannot read FILE: <reason>` and no row. Before, the same stderr line was followed
  by `WARN  source-of-truth  source_of_truth undeclared` and exit 0. Retro go-check-scripts-ports Δ2.
- `scripts/check-bats-skips.sh`: the header of `tests/bats-skip-allowlist.txt` states the pattern dialect (POSIX ERE as
  Go `regexp.CompilePOSIX` reads it), and a line Go compiles with another meaning than the author wrote is exit 3 with
  `check-bats-skips: allowlist line uses <construct>, outside the dialect (...): <line>`: a backslash before anything
  but an ERE operator (`\<`, `\>`, `\n`, `\t`, `\x41`), a backslash inside a bracket expression, `[[:word:]]`,
  `[[=a=]]`, `[[.a.]]`. Before, such a line was matched silently. A line Go does not compile keeps
  `allowlist line is not a valid ERE` and exit 3. Retro go-check-scripts-ports Δ3.
