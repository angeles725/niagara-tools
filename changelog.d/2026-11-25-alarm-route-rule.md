### Changed

- mcp-n4: `n4_commissioning_report`'s `classes_without_recipient` counts only links from a class's alarm topic into a `routeAlarm` action, the rule `n4_list_alarm_classes` uses (one shared function); a link into another slot no longer hides a class without a recipient. Still one link scan.
