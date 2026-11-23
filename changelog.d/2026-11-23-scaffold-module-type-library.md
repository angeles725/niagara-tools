### Added — scaffold-module --type library

- `toolbelt/scaffold-module.sh <ModuleName> <out-dir> --vendor <v> --type library` emits a library module of
  `types/utility-lib.md` Shape A (an `-rt` part with `<types/>`, no Java, no palette) from the new
  `fixtures/MinimalLibrary`; its `MODULE-MAP.md` declares `type: library`, so `lint-structure` passes it clean. The
  fixture has not been built with gradle. The usage line and the unknown `--type` message name the four types.
