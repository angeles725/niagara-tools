### Added

- build-n4-module: two persistence rules confirmed live on a 4.14.0.162 lab station (B1462 §1462.10): a module-written `AuditEvent` records the user name it passes (`types/logic.md`), and a custom `BHistoryRecord` layout change is never detected by the station, so the layout is versioned by the module (`types/logic-authoring.md`).
