### Changed — n4call executes a write only with --execute, a destructive one only on its own line (retro row M17, opportunity O9)

- **Breaking for n4call scripts**: without `--execute`, every write in the calls file is a dry run
  only and the stderr mode line says so; a run that executed writes before needs `--execute` now.
  `--dry-run-only` stays and excludes `--execute`. A flag lost on the way to n4call can no longer
  execute a write: a wrapper's dry-run-only flag did not reach n4call and an `n4_remove_component`
  executed on the lab station.
- A destructive write (`destructiveHint`: remove, rollback, save, ack_alarm, set_tags) executes only
  with `--execute`, `--allow-destructive` and its own line's `"execute": true`; a run with
  `--execute` that misses either is refused with exit 2 before the server starts, naming the line.
  `"execute"` on a line without `"write": true` is a usage error.
- The server already executes only with a token a dry run issued in the same process and connect
  session; two new black-box tests pin a restart and a token no dry run issued.
