### Fixed — run-2 review fix-forward: skip allowlist last line, CL19 node, build-verify mode text

- `scripts/check-bats-skips.sh` reads an allowlist whose last line has no trailing newline (that line was dropped,
  so its skip read UNLISTED).
- `tests/ci-local.bats` CL19 skips its lockfile pin check with "node not installed" on a host without node; CI's
  bats job sets up node, so the reason is deliberately not allowlisted there.
- `build-verify.md` § Known gap: mode B (the `-ux`-only build) and mode C (`-rt` only) never regenerate `-ux` slots;
  the sentences that called mode B silent and "the exception" are gone.
