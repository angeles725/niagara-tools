<!-- ledger: kit -->
<!-- date: 2026-10-27 -->
<!-- last-change-note: kit-backlog B5.3: lint-structural-tests (G34) warns a srcTest read of a .java source in a part with no station test -->
log: 2026-10-27 · kit-backlog B5.3 (G34, C41 advisory) → new lint `lint-structural-tests.sh <module-root>...` (`go/internal/lint/rules/structuraltests`): WARN (FAIL under `--strict`) each `.java` literal read through `Files.readAllBytes`/`readAllLines`/`readString`/`lines` or built with `Paths.get`/`Path.of` in a `srcTest` file of a part whose `srcTest` has no `BTestNg`/`BTestNgStation` class. `types/moduleTest.md` § Client parity doctrine points to it; `tests/lint-structural-tests.bats` ST1-ST2. [ev: retro testing-parity-btestng Δ5]
