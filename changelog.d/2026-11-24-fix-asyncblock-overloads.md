### Fixed — `lint-async-blocking` judges the do overload the introspector runs, not the last one declared

- `go/internal/lint/rules/asyncblock` keyed a class's do methods by name only, so with overloads the last
  declaration won: `doX()` blocking beside a clean `doX(Context)` could be missed, and a clean `doX()` beside a
  blocking `doX(Context)` was flagged although `doX(Context)` never runs. The rule now keeps every overload and
  picks the one the action binds to, with the same shapes as `lint-action-handler` (`doX()`, `doX(P)`,
  `doX(Context)`, `doX(P, Context)`; the no-Context shapes are found first, `ComplexIntrospector` G1). A private
  method, or an overload whose parameters do not fit the action, is never judged (`lint-action-handler` reports it).
- Pinned by the `overloads` golden case and `AB5` in `tests/lint-async-blocking.bats`. Closes #1013.
