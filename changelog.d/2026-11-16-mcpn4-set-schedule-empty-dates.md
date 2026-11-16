### Fixed — mcp-n4: n4_set_schedule with empty date lists
- `add_date: []` and `remove_date: []` on a calendar or trigger schedule are refused as "nothing to set" (when nothing else is given) instead of failing with an internal error.
