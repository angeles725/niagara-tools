### Added — scaffold-module `--type service` and `--with-doc` (course-findings campaign, unit U11: S5, S7)

- `scaffold-module.sh --type service`: a station service plus its background export job (`BSimpleJob`) and a pure
  class with its JUnit test, in a strict module map. Job contract kept: slots snapshotted in the constructor,
  `isAlive()` then `JobCancelException`, an OPERATOR action that submits with a null Context, a plain-file-name
  guard, UTF-8 through `createFileToOverwrite`. It starts from the library fixture; `templates/service-job` replaces
  its plumbing.
- `scaffold-module.sh --with-doc`: a `<Module>-doc` help part from `templates/doc` (gradle file, TOC, index, one
  Guide-on-Target page `doc/<Module>-<TypeName>.html` named after the registered type, no `B`) and its map row. A
  library registers no type, so the flag is exit 2 there.
- `types/structure.md`: the `-doc` recipe's TOC example now matches the Tridium samples (`target` with `.html`, no
  DOCTYPE) and names the flag; `types/service.md` names the scaffold.
- Tests: `go/internal/scaffoldmodule` (`service_test.go`, `doc_test.go`) and `tests/scaffold-module-service.bats`.
- Not in this change: S6 (async-offload `Invocation` fixture) and the B3 `userPermissionsScenario` station-test case.
- `go/README.md` and `BUILD-LOOP.md` list the new `--type service` and `--with-doc`.
