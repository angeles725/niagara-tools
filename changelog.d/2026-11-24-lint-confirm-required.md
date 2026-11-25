### Added — lint-confirm-required

- New `n4lint lint-confirm-required [--strict] <src-root>...` rule and `toolbelt/lint-confirm-required.sh`: FAIL a
  `@NiagaraAction` whose name holds a destructive word (`reset`, `clear`, `delete`, `remove`, `purge`, `wipe`,
  `factory`, `restart`, `reboot`, `overwrite`, whole camelCase words) and whose `flags` lack
  `Flags.CONFIRM_REQUIRED`, so Workbench and the bajaux `invokeAction` ask before running it. An annotation split over
  several lines is one annotation, `|`-combined and numeric flags are read, and comment, Javadoc and string text is
  never code. A `// lint-confirm-required: waived <reason>` comment on the annotation line or on a `//` line right
  above waives it; a marker with no reason is a FAIL of its own.
- `types/actions.md` § 3 gains the rule: which actions count as destructive, that they combine the flag with `|`
  (`Flags.OPERATOR` only when operators may run them), and that the flag is a client prompt, not authorization
  (pair it with `types/security.md` § 2.2, § 2.2a and § 2.4).
