### Added

- mcp-n4: `n4_discover_points` rows of a proxied object carry `name_differs` (the point is named unlike the BACnet `object_name`, which lives only on the device) and the read-back counts them in `names_differ`. (#820)
