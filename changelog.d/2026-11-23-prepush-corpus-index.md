### Changed — the pre-push replay runs the corpus-index check on a kit doc change

- `scripts/ci-local.sh --changed` (which `.githooks/pre-push` runs) now also runs `tests/corpus-index.bats` on the
  snapshot of a kit doc change, before kit-links. A guide citing `[ev: corpus B<n>]` for a block the corpus index does
  not list is refused locally instead of failing the CI bats job (#948 failed there on B1453).
