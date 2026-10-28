### Fixed — pre-push gates every pushed ref; CI runs the npm-tool suites and asserts a SKIP allowlist

- `.githooks/pre-push` gated only the FIRST ref line of a push: `git push origin a b`, `--all` or a branch deletion
  listed first let every later ref through unchecked. Every ref line now runs the gate; the first failing ref stops
  the push with its status.
- The CI bats job installs the pinned npm tools of `build-n4-module-kit/toolbelt/eslint/package.json` (eslint
  10.11.0, acorn 8.18.0; installed versions asserted, npm cache keyed on that file), so the report-module ESLint and
  lint-vendor-floor acorn tests run in CI instead of skipping.
- New `scripts/check-bats-skips.sh <bats.tap> <allowlist>`: the CI bats step fails when a test skips for a reason no
  line of `tests/bats-skip-allowlist.txt` matches (one whole-reason ERE per line; exit 1 unlisted, 3 usage or no plan).
