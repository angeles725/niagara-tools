### Fixed — mcp-n4: a remove's confirmation token survives runtime slots ticking (retro row M15)

- `n4_remove_component` no longer hashes the values of runtime slots into the plan its confirmation token
  binds. A runtime slot is transient (never persisted): its declared or its instance flags hold `t`, such as an
  alarm extension's `timeInCurrentState` (flags `rt`). The dry-run inverse marks it `"runtime": true` and shows it
  without its value, so its ticking between the dry run and the execute no longer refuses the execute with
  "confirmation_token does not match".
- Readonly alone is not runtime: a readonly struct's children (a schedule's `yearSchedule/year`) and persisted
  readonly state (`alarmState`) still bind the token.
- The token still binds the remove op, the subtree's structure (every slot and child name and type) and every
  configuration value: a child added or a configured value changed after the dry run is still refused.
- The journal and `n4_rollback` keep the whole snapshot, runtime values included, as before.
