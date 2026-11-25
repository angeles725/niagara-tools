### Changed

- mcp-n4: `n4_invoke_action` sends the default `control:Override` with `active`/`inactive` only to a `control:BooleanWritable` or a subtype the type registry names; another component's call stays argument-less (an unreadable registry keeps the override). A test pins that the station's stated link refusal counts only on a batch's first op.
