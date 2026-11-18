### Added — mcp-n4: `n4_connect` `expected_version` (field retro delta U14)

- `n4_connect(expected_version=...)` refuses the session unless the detected version equals it or
  starts with it by whole dotted parts (`4.14` matches `4.14.0.162`, `4.1` does not); an undetected
  version never matches. Without the argument a connect is unchanged. `expected_station` already
  pinned the station name.
