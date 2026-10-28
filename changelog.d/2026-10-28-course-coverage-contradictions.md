### Fixed — kit lines the N4 course evidence contradicts

- `types/logic-authoring.md`: a station restart at about 5000 subscribed points is reported, its cause unconfirmed;
  a bounded driver worker queue throws `QueueFullException` to the poster, it does not restart the station.
- `types/driver-authoring.md`: COV sensitivity is driver-specific (no COV slot on the base `BTuningPolicy`; BACnet's
  `covPropertyIncrement`, default 1.0, applies to COV-Property subscriptions only).
- `types/logic.md`: `stopped()` runs on `Nre:Console` only for a console quit or an Application Director stop; a
  platform reboot or rename uses the platform service thread.
- `types/structure.md`: `niagara.security.manager.disable` switches to a logging developer mode when licensed and only
  logs a WARNING otherwise; it never turns the security manager off.
