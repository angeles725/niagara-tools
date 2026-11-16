### Fixed — n4_rollback restores the outgoing links of a removed component

- An acknowledged `n4_remove_component` records in its inverse each link that leaves the removed
  component (stored on a target outside it) as an `outgoing_link`; `n4_rollback` re-creates those
  links after the component, through the same link writer and journaled intent as the links inside
  it, and refuses when an end is outside today's write scope.
- The rollback read-back looks for each re-created link on its target and reports
  `outgoing_links: {restored, missing}`; a missing one makes the verdict `mismatch`. Before, a
  remove then rollback of a constant block lost its links into other blocks and read `verified`.
- A link stored at the remove snapshot's last level now keeps its slot names: its plan note names
  them (it printed `(None <- None)`), `snapshot_truncated` no longer lists the link's own slots,
  and such an internal link is recorded as a `relink` that `n4_rollback` restores, where the
  rollback refused it as a malformed relink before.
