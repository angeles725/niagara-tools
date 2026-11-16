### Added — mcp-n4: n4_set_schedule writes the weekly schedule
- `n4_set_schedule(ord, weekly)` replaces only the days given (`{"sunday": []}` clears one), names every fault of the request before anything is sent, and reads the schedule back to compare each day written. Each day is one set whose inverse restores it exactly through `n4_rollback`.
- Every overlapping pair of ranges in a day is named (a long range against each range inside it), and a schedule that is not there is named with the request faults that need no station.
