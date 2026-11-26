### Added — lint-property-accessors

- New `toolbelt/lint-property-accessors.sh <src-root>` (issue #1159): FAILs a top-level `B*` class declaring a slot
  by hand (`public static final Property x = newProperty(...)`) without a public `get<X>()`/`is<X>()` or a public
  `set<X>(...)` in the same class. Such a type compiles, signs and passes `verify-module.sh`, but the station cannot
  load it (`GRAVE [sys.schema] ... No setter for property (x)` / `Cannot load type for ...`, then
  `NoClassDefFoundError` on every use; measured on Niagara 4.14.0.162). One row per missing or non-public accessor, at
  the `Property` declaration; Slot-o-Matic generated code always has both and is clean. Rule package
  `go/internal/lint/rules/propertyaccessors`; documented in `types/structure.md` § Property accessors; mutation tests
  PA1 (missing setter) and PA2 (non-public setter, missing getter) in `tests/lint-property-accessors.bats`.
- `report-module` runs `lint-property-accessors` (FAIL rows relayed, no verdict row) on every artifact's `src/`, after
  `lint-slot-default-type`.
