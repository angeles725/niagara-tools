### Changed

- `types/dashboard.md`: the in-station operator audit trail is a `BStringTrendRecord` history created on the first
  write, with strictly increasing timestamps and the old string ring read as a fallback (issue #912).
- `types/dashboard.md` § Write path: rapid operator steps send one command after the last click and confirm only the
  final value against the device read-back; the receipt timeout is a new behavior question in the
  `METHODOLOGY.md` § Domain correctness catalog (issue #898).
- `types/dashboard.md`: a write handler is a pure apply plan plus a thin executor, so the order and the rollback are
  JUnit-testable with `run-pure-test.sh` (issue #924).
- `types/dashboard.md` § Charts on an HMI and `types/frontend-standard.md` § 4: a history chart holds a value until the
  next record and the last value to the chart end, and reads every series with `valueAt`, never by index; exemplar
  `held()` / `valueAt()` (issue #822, K1).
- `skill/SKILL.md` step 4: on WSL, sign with the module's alias through the explicit profile; never report signing as
  unavailable before running it (issue #917).
