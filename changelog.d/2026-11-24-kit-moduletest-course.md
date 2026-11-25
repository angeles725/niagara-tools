### Changed — `types/moduleTest.md` teaches the 4.14+ test layout and no longer blames the plugin for 0 tests

- The lifecycle example, the `moduleTest-include.xml` example and checklist item 1 use `srcTest/test/com/<v>/<mod>/`
  with `package test.com.<v>.<mod>`. TEST-G8 states that path and package must agree: two of our module
  tests declare `package com.angeles.…` under `srcTest/test/`, and one built test module has no `<types>`
  (B1236 §1236.8). `Total tests run: 0` now lists the layout first, then the WSL route, then the signing; the
  "plugin 7.6.17 bug" is marked `[INFER]` with its live check (B1236-G1) open.

### Added — `types/moduleTest.md` TEST-G9 and TEST-G10

- TEST-G9: the RBAC harness `userPermissionsScenario(...).actAsUser(...)` of `BTestNgStation` with a worked example,
  its steps, its cleanup, and the two course tests that use it.
- TEST-G10: the `TestHelper` utilities `LatestLogHandler`, the private-field helpers (`getPrivateField`,
  `setPrivateField`, `runWithAlternateValue`), `invokePrivate`, `assertCanRunConcurrently` and `waitForClockChange`,
  each with when to use it and the trap read in the source (latest record only, no level change,
  `Exception`-only catch, exact-class argument match).
