### Added — mcp-n4 alarm health (station-audits S4)

- `n4_alarm_health(window, chatter_threshold, unacked_days, top)`: chattering sources, alarms still out of normal,
  alarms unacknowledged for more than N days and the busiest sources of a window, from two fixed reads of `alarm:`
  grouped by the server. Read-only.
