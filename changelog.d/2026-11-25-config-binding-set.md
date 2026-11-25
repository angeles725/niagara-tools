### Changed

- mcp-n4: `n4_set_slot` on a template instance's `templateConfig` names the slot the station propagates the value to (`propagates_to`), refuses it outside the write scope or outside the instance, reads it back there, and refuses a `#Name`/`#DisplayName` binding (use `n4_rename_component`).
