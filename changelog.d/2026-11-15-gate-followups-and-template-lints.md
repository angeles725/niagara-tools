### Added — lint for the template archives a module ships

- `toolbelt/lint-ntpl-manifest-deps.sh [--niagara-home <dir>] <module-root>` (Go lint G92): every `*.ntpl` under a
  module root (the extension in any letter case) must hold `template.bog` and `template-manifest.xml`, a well-formed
  `<template>` manifest with every `<dependency>` named and inside `<dependencies>`, and only dependencies that a
  `*.gradle.kts` of the module declares (`api`, `nre`, `implementation`) or that the Niagara installation ships; a
  template under `newStations/` must carry `isStation="true"` on its manifest root. Rows
  `FAIL  ntpl-entries|ntpl-manifest|ntpl-deps|ntpl-station`; exit 0 no FAIL (one PASS row when the module ships no
  `.ntpl`), 1 any FAIL, 3 usage or environment.
- The installation is `--niagara-home <dir>` or `NIAGARA_HOME` (the option wins): each `modules/<name>.jar` is a
  module it ships. Without one the lint cannot tell a stock module from a missing one: it prints one WARN row
  saying so, reports undeclared dependencies as `WARN  ntpl-deps`, and exits 0. A path that does not exist or has
  no `modules/` is exit 3.
- The archive is read as untrusted input: 32 MiB, 10000 entries and a 4 MiB manifest at most, and an unsafe entry
  name refuses it. A template behind a symlink (a linked file, or a file under a linked directory) is read
  through the link; a `.ntpl` link that leads nowhere is exit 3. A path with a control character is printed
  quoted, one row per line. `report-module` does not run the lint yet.
- `types/logic-authoring.md` § What a template upgrade keeps and what it wipes; `types/distribution.md` § 8 Check.

### Changed — four gate tools refuse inputs they used to pass

- `scripts/check-bats-skips.sh`: an allowlist line using `{,2}`, `{}`, an interval bound with a leading zero
  (`{02}`, `{1,02}`), any other unescaped brace that opens no `{n}`, `{n,}` or `{n,m}` interval, a star right after
  the start anchor (`^*`), or a class name in single brackets (`[:alpha:]`) is exit 3 naming the construct. Such a
  line was matched as Go reads it, which is not what GNU `grep -E` reads. Write `\{` for a literal brace, `\*` for
  a literal star and `[[:alpha:]]` for a class.
- `toolbelt/lint-vendor-floor.sh`: an rc directory the process may not search is exit 3
  (`env fault: cannot look for vendor/ under <rc-dir>: <reason>`). It was "no vendor/ under <rc-dir>", exit 0.
- `scripts/check-client-source.sh`: a BUILD-STATE file with CRLF or CR-only line endings, a doubled CR, a UTF-8
  byte-order mark, or blanks after the `module:` or `source_of_truth:` value is read as what it declares, and
  `--build-state ""` is the usage error (exit 2) unless `--declared` is given. All were
  `WARN  source-of-truth  source_of_truth undeclared`, exit 0. A value no longer keeps the blanks that end it.

### Fixed — a crash and two silent skips

- `toolbelt/report-module.sh`: a module with a part whose `src/rc` is a regular file crashed the report
  (`slice bounds out of range` in the module-map reader, exit 2); it gets its ordinary rows.
- `toolbelt/report-module.sh`: a `src/rc/vendor` symlink that leads nowhere gets the
  `ERROR  vendor-floor  env fault (exit 3)` row; it got no row and the report could read CLEAN.
- `scripts/check-installed-line.sh`: a working-tree entry literally named `<ref>:<group>/build.gradle.kts` no longer
  hides the group (its downgrade was not reported), and a group path holding a control character, a double quote
  or a backslash is printed quoted with C escapes, so a path with a newline is one row.
