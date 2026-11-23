### Fixed — lint-servlet-write-guards no longer drops one-argument generated setters

- `lint-servlet-write-guards`: the "fewer than two arguments" exemption now applies to the bare `set(v)` only
  (`AtomicReference`, `ThreadLocal`). A generated property setter takes one argument and is a component slot write,
  so `comp.setDouble(42.0)` / `n.setBoolean(true)` again get `no-target-permission` and `no-csrf` (the narrowing of
  the previous fold had hidden them). A numeric first argument still drops a call only when a value follows it
  (JDBC `setInt(1, v)`, `List.set(0, x)`).
