### Added — lint-instance-type-order

- New `n4lint lint-instance-type-order [--strict] <src-root>...` rule (issue #847): FAIL a Java class annotated
  `@NiagaraType` (or holding a Slot-o-Matic `TYPE = Sys.loadType(...)` field) that declares a static `INSTANCE` after
  its static `TYPE`. A station refuses such a type with "INSTANCE field is null; insure getType() comes last" while
  tests, build, verify and report pass; the row says to declare `INSTANCE` before `TYPE`. A class with no `INSTANCE`,
  or with `INSTANCE` first, is clean, and a comment or string literal is never read. Gotcha D6 in
  `types/issues-and-gotchas.md`.
