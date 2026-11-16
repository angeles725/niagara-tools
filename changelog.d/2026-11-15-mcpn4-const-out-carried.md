### Fixed — copy, remove and rollback carry a constant block's configured out

- `n4_copy_component` and the `n4_remove_component` snapshot used to drop every Status `out`, so a copy of a
  `kitControl:NumericConst` with `out=10.0` held 0 yet read `verified`, and remove then rollback brought it
  back at 0. An `out` is now kept when the parent's type contract declares it without `r` or `t` (the
  constant blocks declare it summary-only); a computed `out` (a writable's, declared `rtso`) is still left out,
  as is any `out` whose type the station does not describe.
- The copy read-back compares the carried `out` with the source value, and the rollback read-back of a remove
  compares each carried `out` with the live slot, so a dropped value reads `mismatch`, never `verified`.
- The remove plan now reads the subtree's type definitions through the registry channel (one call).
