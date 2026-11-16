### Added — lint-structure accepts a declared theme or library module

- A module declares its kind with an optional line in the header of `<module-root>/MODULE-MAP.md`: `type: theme` or
  `type: library`. A declared theme's zero-Java `-ux` part passes L9 when it holds at least one `.nss` file, and
  FAILs without one; a declared library's `-rt` part passes L5 with no `module.palette` (a palette it ships is still
  judged). Before, both shapes FAILed exactly as `types/theme.md` and `types/utility-lib.md` prescribe them.
- The kind is read from the map only, never inferred from missing files: an undeclared part FAILs as before. The
  undeclared `-ux` L9 row now ends with how to declare a theme; the L5 text and the `-wb` L9 text are unchanged.
- `types/structure.md` lists every rule the lint runs (L12 to L16 added, L8 marked as `verify-module.sh`'s, L10 as a
  WARN or a FAIL by key and tree), and the header of `toolbelt/verify-module.sh` every check the gate runs, in run
  order. Two Go tests fail when either list and the code differ.
- Corrected against the code: where the station test runner runs (`METHODOLOGY.md`, `BUILD-LOOP.md`,
  `types/moduleTest.md`), the seven Spanish level names the console triage matches (`types/diagnostics.md`), and the
  corpus-index range in `skill/SKILL.md`.
