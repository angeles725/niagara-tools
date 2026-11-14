### Added — mcp-n4 create_component with a driver proxy extension
- `n4_create_component` can create a point together with its driver proxy extension.

### Fixed — mcp-n4 copy, remove and simple encoding
- A `baja:Simple` takes a whole number as its decimal string; the Modbus enum tag is `signedInteger`.
- `n4_copy_component` and `n4_remove_component` accept station-escaped names for an existing copy or source.
- A copy leaves out name-only children, the derived Modbus address and alarm extension state.
