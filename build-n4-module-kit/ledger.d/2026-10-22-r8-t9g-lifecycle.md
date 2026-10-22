<!-- ledger: kit -->
<!-- date: 2026-10-22 -->
<!-- last-change-note: quality round 8 T9g unit 3: component lifecycle from the course recordings -->
log: 2026-10-22 · quality round 8 T9g unit 3 (course recordings B1416/B1417) → logic-authoring lifecycle: `stopped()`/`started()` follow running state, not enable (services: `disabled()`/`enabled()`), links activate and propagate before `started()`, `started()` not `stationStarted()`, revert-and-log in `changed()`, `isParentLegal`/`isChildLegal`, frozen-slot facts (HIDDEN not a lock, frozen order, missing `do<Action>`, compare with `getDefaultValue`, default/type mismatch); `pollEnabled` on the poll service. No register C-row, no lint. Retro retros/2026-10-22-r8-t9g-lifecycle.md 2Δ folded. [ev: retro r8-t9g-lifecycle]
