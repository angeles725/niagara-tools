### Fixed — weekly sweep of small review advisories (#968 #971 #973 #979 #981 #983)

- `n4_histories_audit`: a panic in a pooled history read is that history's failed read, noted in order, instead of
  ending the server (#968). `n4_rollback chain=true` on a create of several components says it follows nothing (#971).
- `ci-local --changed`: deleting a Go file runs the file-size gate (its debt line stayed until CI); the file-size
  messages name the debt file one way (#973, #983).
- `scripts/bats-shard.sh --check` fails on an empty shard (#981). `scripts/pr-size.sh` prints paths shell-quoted, so a
  file name with a newline cannot inject a workflow command; the `size:exception` waiver is the tested
  `scripts/pr-size-waiver.sh` (#979).
