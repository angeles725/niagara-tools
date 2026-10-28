### Fixed — lint rare cases found by new tests

- `lint-lifecycle-guard` LG2: two anonymous bodies of the same type no longer share one finding key.
- `lint-intrinsic-lock`: IL2 judges `<Outer>.this` inside an anonymous class; IL3 no longer counts a lambda or
  anonymous-class method body inside `synchronized` as a nested lock.
- `lint-charset-default`: a keyword before `(` (`return (s + "!").getBytes()`) is no longer read as a receiver.
- `lint-structure` L17: a commented-out `moduleName` in a gradle script no longer wins.
- The part-output prune list shared by three lint rules now lives in one helper.
