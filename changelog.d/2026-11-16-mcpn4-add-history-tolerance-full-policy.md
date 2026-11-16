### Added — n4_add_history sets a numeric COV change tolerance and the full policy

- `change_tolerance` (a number of at least 0, in the point's units) sets `changeTolerance` of each
  `history:NumericCovHistoryExt` in its add body, for every point of the batch: a change smaller than it records
  nothing. It is refused for mode interval, for an `ext_type` other than the numeric COV one and for a point that is
  not numeric, naming the point. The read-back checks it.
- `full_policy` (`roll`, the extension default, or `stop`) is set by a set op on
  `<ext_name>/historyConfig/fullPolicy` (`history:FullPolicy`) right after each add in the same batch; the capacity
  note names the policy and the read-back checks it.
- Three COV histories with a 0.5 A tolerance, 50000 records and a full policy are now one dry run and one token
  (proven live on a 4.15 lab station: 9 ops, `verified`, a 0.3 change recorded nothing). Before, the tolerance took a
  `n4_set_slot` dry run and token per point. `capacity`, `mode=cov` and N points under one token were already there
  (#670). Client-A overamps retro 2026-11-16, row M4.
