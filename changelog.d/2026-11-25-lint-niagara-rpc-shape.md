### Added — `lint-niagara-rpc` checks the shape of an `@NiagaraRpc` method: `no-context` and `no-box-transport` (finding K3)

- `no-context` FAIL: an `@NiagaraRpc` method whose last parameter is not typed `Context` (a qualified `....Context` and
  `final` or annotated parameters pass). The station appends the caller's Context as the last argument and finds the
  method by those argument types (`NiagaraRpcUtil.java:91-123`, `:189-192`), so a method with no parameter, a
  `SecurableContext`, a `Context[]` or a Context that is not last is never found (`NoSuchMethodException`). The row is on
  the line of the last parameter.
- `no-box-transport` WARN: an `@NiagaraRpc` whose literal `transports` lists no `box` (`TransportType` is `web`, `box`,
  `fox`) while a `*.js` file under the same root calls `baja.rpc(` (a comment, string or template is no call).
  `baja.rpc` sends `sys/rpc` over BOX (`bajaScript-ux rc/comm.js:729`, `box-rt BSysChannel.java:200-201`), and a method
  without box fails the transport match (`NiagaraRpcUtil.java:128-130`, `:181-187`). `transports` has no default
  (`NiagaraRpc.java:16`): an absent one is a compile error and is not a row; a constant is not judged;
  `transports = {}` warns. The row is on the `transports` line.
- A `// lint-niagara-rpc: reviewed <reason>` marker waives the new rows too (a bare marker is still its own FAIL).
- The rule now takes `--strict` (a WARN is exit 1) and the usage line says so; a root with `*.js` and no `*.java` is
  still the empty-root error (exit 3). The rule reads `*.js` only for the `baja.rpc(` call.
- `types/actions.md` § 6 points at the two checks. Goldens in `go/internal/lint/rules/niagararpc/testdata`, five
  new Mutation pins and bats tests in `tests/lint-niagara-rpc.bats`.
