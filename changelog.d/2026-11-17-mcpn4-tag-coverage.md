### Added — mcp-n4: `n4_tag_coverage`, how far a station's Haystack tagging reaches (station-audits S9)

- `n4_tag_coverage(base, dictionary?, max_points=2000, exclude_services=true)` lists, in one read-only call, the
  control points missing `hs:point` or an outbound `hs:equipRef` relation (`points_without_tags`, with what each
  misses), the `hs:equip` components no point references (`equips_without_points`) and the direct tags no installed
  dictionary defines (`unknown_tags`, with count and an example; a namespace with no dictionary installed is one
  `ns:*` entry, so a station without a Haystack dictionary says so once). Implied tags and relations count. Lists
  keep 100 items; `counts` holds the totals; a read cut at `max_points` sets `truncated`.
- Bulk tagging (S10) needs no new tool: `n4_set_tags` already adds and removes on up to 50 ords under one dry run and
  token; a new test pins N points with add and remove together.
