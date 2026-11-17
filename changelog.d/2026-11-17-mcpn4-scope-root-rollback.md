### Changed — n4_rollback can undo a create of the write-scope root under the same scope

- A create of the scope root itself was allowed by that scope but its rollback was refused as outside it
  (the rollback checked the parent). The rollback is now checked on the created path, as the create was,
  while nothing was written inside the root later; `n4_remove_component` of the root stays refused.
