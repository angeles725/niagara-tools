### Added — mcp-n4 finds components by tag and lists the tag dictionaries

- `n4_neql_query` runs one NEQL query (`<base_ord>|neql:<query>|view:file:ITableToCsv`) and returns the
  matching components' station ORDs, or with `bql` the columns and rows of a select run on the matches.
  `|` is refused, a station parse error is reported as HTTP 400 with the query, `max_rows` (default 200,
  at most 2000) bounds the reply.
- `n4_list_tag_dictionaries` lists the installed tag dictionaries (namespace, enabled, frozen, version,
  status) with their tag definitions, tag groups and relation ids, bounded by `max_items`.
