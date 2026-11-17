### Changed — mcp-n4: n4_set_schedule and n4_invoke_action are destructive (M19)
- Both are now listed with `destructiveHint: true`: `n4_set_schedule` replaces every event of the schedule (`n4_rollback` still restores it exactly), and `n4_invoke_action`'s `active`, `inactive` and `auto` have no inverse. `n4call` therefore executes them only with `--execute`, `--allow-destructive` and the line's own `"execute": true`.
- benchcall's runner field `dryRunOnly` is now `writesExecute` (it held the effective mode, not the flag) (M17b).
