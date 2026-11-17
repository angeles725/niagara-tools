### Added — mcp-n4 alarm health (station-audits S4)

- `n4_alarm_health(window, chatter_threshold, unacked_days, top)`: chattering sources, alarms still out of normal,
  alarms unacknowledged for more than N days and the busiest sources of a window, from two fixed reads of `alarm:`
  grouped by the server. Read-only.

### Added — mcp-n4 alarm routing audit (station-audits S5)

- `n4_list_alarm_classes(audit=true)`: the classes no recipient is linked to, and each recipient's settings and the
  problems that stop delivery (no or invalid email address, no remote station, empty transitions or days, an empty
  webhook/Telegram endpoint, a last delivery failure). Credential slots are named, never returned. Without `audit`
  the reply is unchanged.
