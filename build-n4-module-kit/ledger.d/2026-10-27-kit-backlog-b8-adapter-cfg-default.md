<!-- ledger: kit -->
<!-- date: 2026-10-27 -->
<!-- last-change-note: kit-backlog B8: lint-adapter-cfg-default (audit A7-D1) warns a Cfg default that differs from its slot default -->
log: 2026-10-27 · kit-backlog B8 (audit 2026-10-03 A7-D1) → new lint `lint-adapter-cfg-default.sh <module-root>...` (`go/internal/lint/rules/adaptercfgdefault`): WARN a pure-model `*Cfg` field whose default (literal initializer, or 0/false) differs from the `@NiagaraProperty` defaultValue of the slot the adapter fills it from (`x.f = getSlot();`, `.getMillis()`/`.getSeconds()` for a BRelTime). `types/logic.md` § stateful controller points to it; `tests/lint-adapter-cfg-default.bats` ACD1-ACD2. Audit A7 closed.
