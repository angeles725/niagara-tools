### Changed — the template lint stays inside the module and runs in report-module

- `toolbelt/lint-ntpl-manifest-deps.sh`: a symlink is followed only when its real target is inside the module
  root. A link to an ancestor (a sibling module), to `/` or to a file of another module is not followed and is one
  `WARN  ntpl-link  <link>: a symlink to <target>, outside the module root -- not followed, nothing behind it is
  checked` row, exit 0; a module whose only finding is such a link prints no "nothing to check" row. A link inside
  the module is pruned by its target too, so `out -> build` no longer re-exposes a part's build output, and a file
  reached under two spellings is read once.
- `--niagara-home ""` is a usage error (exit 3, the lint's usage exit) instead of "no installation" (which turned
  every `ntpl-deps` row into WARN). A `modules/<name>.jar` ships its module only when it is a regular file the lint
  can open: a dangling link, a directory or an unreadable file under that name ships nothing. A `.ntpl` that is
  not a regular file (a FIFO, or a link to one) is exit 3 naming it, never "nothing to check".
- `toolbelt/report-module.sh` runs the template lint once per module root (a new member) and takes
  `--niagara-home <dir>`, passed on to it; without the option the lint reads `NIAGARA_HOME` from the environment
  report-module inherits. Its rows are relayed whole (`(module)  FAIL  ntpl-deps  <file.ntpl>: <reason>`), a
  `FAIL` row fails the report, and a module with neither a build file nor a template is a SKIP row. Every report
  gains the member's row (`PASS  lint-ntpl-manifest-deps  clean` or SKIP).
- `verify-module --src` reads the declared dependencies through the shared `lint.DeclaredDeps` (no behaviour
  change).
