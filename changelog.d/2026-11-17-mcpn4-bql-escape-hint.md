### Added — mcp-n4: n4_bql_query hints the $24 form on an empty escaped-literal result

- An empty `n4_bql_query` result whose quoted literal holds a `$xx` escape carries a `hint`: BQL decodes the
  escape (`$20` is a space) while slot names are stored escaped, so `name = 'A$2420B'` matches `A$20B`.
