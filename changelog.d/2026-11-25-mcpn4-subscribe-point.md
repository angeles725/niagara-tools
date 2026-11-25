### Added

- mcp-n4: `n4_subscribe_point(ord, seconds)` subscribes one control point over BOX for at most 40 s so its driver polls it, re-reads it every 2 s until its stale flag clears, and always unsubscribes (also after a failed read; BOX subscriptions have no lease). A freshly created proxy point can now be checked against the field without writing to it; `ended` says `stale_cleared`, `not_polled` (point or device down, disabled or in fault), `read_failed` or `timeout`. (#875)
