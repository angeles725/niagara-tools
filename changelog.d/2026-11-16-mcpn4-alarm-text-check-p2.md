### Added — mcp-n4: n4_alarm_text_check, the live LC9 alarm-text check (retro row P2)

- `n4_alarm_text_check` reads every alarm source extension below `base` (one BQL select, subtypes included) and
  flags `parent-path` (a `%...%` token rooted at no alarm record field, such as `%parent.out.value%`: the texts
  resolve against the record, never the source) and `empty-tonormal` (a `toOffnormalText` with an empty
  `toNormalText`, which blanks `msgText` on every return to normal). Rule names, slots and rows match the offline
  `bog-nav alarm-text`, so the same extension gives the same finding from a bog or a live station. METHODOLOGY's
  LC9 row is now mechanical. The 2026-11-13 client A over-amps text failed silently; it is now one read-only call.
