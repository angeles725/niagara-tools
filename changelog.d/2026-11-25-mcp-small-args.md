### Changed

- `mcp-n4` `n4_add_history` accepts `capacity_records` as an alias of `capacity` for a number of records
  (1 to 2147483647; `"unlimited"` stays `capacity` only). It plans, executes and reads back exactly what
  `capacity` does. Giving both is refused before the station is touched: `capacity and capacity_records are the
  same argument: give only one of them`. Any other unknown argument keeps its `unknown argument '<name>' ...; did you
  mean '<key>'?` refusal (`capacity_recs` still suggests `capacity`). The tool schema and description name the alias.
  Tests: `TestAddHistoryCapacityRecords*` (writeops), `TestAddHistoryDeclaresCapacityRecords` (registry). Issue #859.
- `mcp-n4` `n4_bql_query` description now says that BqlQuery decodes `$xx` inside the query text while slot names are
  stored escaped, so a `where name = 'Numeric$20Writable'` matches nothing: write `'Numeric$2420Writable'` (`$24` is a
  literal `$`) or use `like` with `*`, and that an empty result with an escape in a quoted literal carries that
  advice in `hint` (already done by `bql.EscapeHint`, #753; `TestEmptyResultWithAnEscapedLiteralHintsTheDollar24Form`).
  Decoded names are not accepted: the station decodes the query text before it compares, so the escaped form is the
  only one that reaches the name. Test: `TestBqlQueryDescriptionTellsTheEscapedNameForm` (registry). Issue #876.
