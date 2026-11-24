### Changed — CI runs the bats suite in two duration-balanced shards

- The `bats` CI job is now `bats unit tests (shard 1/2)` and `(shard 2/2)`. New `scripts/bats-shard.sh <i> <n>`
  prints one shard's files (a weight table placed longest-first, unlisted files round-robin, heaviest first in
  each shard), and `--check <n>` fails unless every `tests/*.bats` file is in exactly one non-empty shard.
  Each shard keeps the Java 8 setup, the TAP plan-versus-results count and the skip allowlist gate.
- The shard reaches the steps through `$GITHUB_ENV` (a first step of the job), because `scripts/ci-local.sh` reads
  no job-level `env:` or step-level `if:`; a local replay leaves it unset and runs every `tests/*.bats` file.
