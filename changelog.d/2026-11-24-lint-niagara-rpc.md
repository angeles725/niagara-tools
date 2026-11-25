### Added — lint-niagara-rpc

- New `n4lint lint-niagara-rpc <src-root>...` rule and `toolbelt/lint-niagara-rpc.sh` wrapper: FAIL an `@NiagaraRpc`
  method declaring `permissions = "unrestricted"` (any case) or `""`, FAIL one that writes (`.set(`, `.invoke(`, a
  typed setter with two or more arguments, a `doXxx(` call) with no `OPERATOR_WRITE`, `checkCanWrite` or similar
  check before the first write and no `w`/`W` in a literal `permissions`, and FAIL a bare waiver. Waive one RPC with
  `// lint-niagara-rpc: reviewed <reason>`. Doctrine in `types/security.md` §2.7; the kit already forbade the pattern
  in prose (`types/dashboard.md` RBAC, `types/wb-widgets.md`) with no lint.
