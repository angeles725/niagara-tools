### Fixed — n4_rollback of a create no longer silently removes later work

- The rollback of an `n4_create_component` batch is refused when a later batch (not itself rolled back) recorded
  a target equal to or under the created component, or when the component now holds a link or a child component
  its create did not make (a Workbench edit no journal records; a frozen slot its type declares does not count).
  Before, the plan was one remove op and the verdict `verified`: on the bench, rolling back the create of a folder
  removed it with 10 blocks, their history extensions and 6 links later batches had added.
- The refusal names the created ORD, each later batch as `batch_id (tool)` newest first and each extra child or
  link, and says to roll the later batches back first (newest first) or remove that work by hand. Nothing is
  removed and no rollback batch is journaled. Rollbacks of other batches are unchanged. Bench retro 2026-11-15 Δ1.
