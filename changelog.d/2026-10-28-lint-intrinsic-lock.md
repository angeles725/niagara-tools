### Added — lint-intrinsic-lock

- New `toolbelt/lint-intrinsic-lock.sh [--strict] <src-root>` (WARN, advisory): IL1 a `synchronized` instance method
  and IL2 a `synchronized (this)` block in a Niagara `B*` class (the framework locks the same monitor), IL3 a
  `synchronized` block nested in another lock in any class. Not yet run by `report-module`.
