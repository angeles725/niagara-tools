### Added — `types/hx-px.md` (Hx and Px in a module) and `lint-wb-usability`

- `build-n4-module-kit/types/hx-px.md`: which of Hx, Px and bajaux to use; the file layout of a Px graphic, an Hx
  view and a Px widget in a module; binding ORDs and the `n4_px_binding_audit` check (what its `broken`, `unknown`,
  `counts` and `truncated` mean and what it does not prove); Hx theming, events and the 5 s poll; permissions and
  CSRF; and the pitfalls the research blocks found (a fixed `actionArg` on a typed action fails silently in Hx, a
  widget with no Hx agent vanishes, a missing `<import>` module). Every rule carries its source. Linked from
  `types/wb-widgets.md`, `build-n4-module-kit/README.md` and the decision table of `skill/SKILL.md`.
- `toolbelt/lint-wb-usability.sh` (rule `go/internal/lint/rules/wbusability`, `n4lint lint-wb-usability`): the
  narrow, mechanical part of the lint candidate in `types/wb-widgets.md` (UX rule). WARN, one per class, at the class
  name, when a class extends `BWbComponentView`, holds a `BOrd` member (a non-static field, or a class-level
  `@NiagaraProperty` of type `BOrd`) and its file has no `BComponentChooser` in code (an import is not an invocation)
  and no `targetType` facet. `--strict` makes a WARN exit 1; exit 3 for usage, a root with no `*.java` or an
  unreadable sub-directory. A comment `lint-wb-usability: waived <reason>` on the class line or the five above it
  waives the class. Left out: whether the user edits the ORD, a chooser in another class or file, drag-drop, the
  status table, and views on a subclass or a manager base. Not a report-module member (`notInReport` in
  `go/internal/reportmodule/completeness_test.go`). Pinned by `tests/lint-wb-usability.bats` (WBU-import,
  WBU-static, WBU-waiver-string, WBU-base) and the rule's golden cases.
