### Fixed — `types/actions.md` no longer says a missing do-method can be a silent no-op, and names the BOX `sys/rpc` wire

- § 1: a frozen action with no `do` method is refused at type load (`ComplexIntrospector.java:316-343`) and
  `lint-action-handler.sh` FAILs it; the old "a silent no-op is possible" sentence is replaced by what a DYNAMIC action
  does (it runs its `BAction` handler value, `ComponentSlotMap.java:1340-1346`).
- § 2: "never run a handler through `this::doX`, a `new Thread` or a pool task" is now a rule that points at the
  `lint-action-handler.sh` WARN (course F3); the stale "owed Go lint G28" note is gone (`lint-async-blocking.sh` ships).
- § 6: `@NiagaraRpc` has two wires, `/rpc` and BOX `sys/rpc` (`baja.rpc` uses BOX, `BSysChannel.java:200-201`); the
  table lives in `types/data-access.md` § 8 and is linked, not repeated.
