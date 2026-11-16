### Added — mcp-n4: n4_set_schedule writes the weekly schedule
- `n4_set_schedule(ord, weekly)` replaces only the days given (`{"sunday": []}` clears one), names every fault of the request before anything is sent, and reads the schedule back to compare each day written. Each day is one set whose inverse restores it exactly through `n4_rollback`.
