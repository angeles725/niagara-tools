### Fixed — mcp-n4: a remove's confirmation token survives runtime slots ticking (retro row M15)

- `n4_remove_component` no longer hashes the values of runtime slots into the plan its confirmation token
  binds: a slot whose declared flags hold readonly or transient (an alarm extension's `timeInCurrentState`,
  flags `rt`), or whose instance flags hold transient, is shown in the dry-run inverse without its value. Such a
  slot ticking between the dry run and the execute no longer refuses the execute with "confirmation_token does
  not match".
- The token still binds the remove op, the subtree's structure (every slot and child name and type) and every
  configuration value: a child added or a configured value changed after the dry run is still refused.
- The journal and `n4_rollback` keep the whole snapshot, runtime values included, as before.
