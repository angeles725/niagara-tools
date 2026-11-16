### Fixed — mcp-n4: n4_alarm_text_check splits tokens as BFormat does (retro row P2b)

- `n4_alarm_text_check` now splits an alarm text into `%...%` tokens with a port of `BFormat.doParse`: `%%` is
  a literal percent, a token ends at the next `%` outside parentheses, and a `%` that never closes makes
  BFormat show the whole text as `ERROR <text>`, so no token of it is reported. Before, a lone `%` paired
  with the next one: `Load at 90% of %parent.out.value%` reported `% of %` as a parent-path token, and a
  `%` inside `%lexicon(...)%` split the call. The offline `bog-nav alarm-text` (niagara-go) has the same
  tokenizer and the same test table. The select's columns are now built from the text slot list, so the
  two cannot drift apart.
- New rule `bad-format`: a text BFormat cannot parse (a `%` that never closes) shows on the station as
  `ERROR <text>` and none of its tokens is evaluated. It is one WARN row naming the slot (token empty);
  `bad_format` counts the rows, `hint` advises `%%` for a literal percent, and the summary names
  bad-format only when a text has it. bog-nav alarm-text reports the same row.
