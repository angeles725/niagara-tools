### Added — lint for the template archives a module ships

- `toolbelt/lint-ntpl-manifest-deps.sh [--] <module-root>...` (Go lint G92): every `*.ntpl` under a module root must
  hold `template.bog` and `template-manifest.xml`, a well-formed manifest and only `<dependency>` names that a
  `*.gradle.kts` of the module declares (`api`, `nre`, `implementation`); a template under `newStations/` must carry
  `isStation="true"` on its manifest root. Rows `FAIL  ntpl-entries|ntpl-manifest|ntpl-deps|ntpl-station`; exit 0
  clean (one PASS row when the module ships no `.ntpl`), 1 any FAIL, 3 usage or environment. The archive is read as
  untrusted input: 32 MiB, 10000 entries and a 4 MiB manifest at most, and an unsafe entry name refuses it.
  `report-module` does not run it yet.
- `types/logic-authoring.md` § What a template upgrade keeps and what it wipes; `types/distribution.md` § 8 Check.

### Changed — four gate tools refuse inputs they used to pass

- `scripts/check-bats-skips.sh`: an allowlist line using `{,2}`, `{}`, any other unescaped brace that opens no
  `{n}`, `{n,}` or `{n,m}` interval, or a class name in single brackets (`[:alpha:]`) is exit 3 naming the
  construct. Such a line was matched as Go reads it (a literal, or a set of letters), which is not what GNU
  `grep -E` reads. Write `\{` for a literal brace and `[[:alpha:]]` for a class.
- `toolbelt/lint-vendor-floor.sh`: an rc directory the process may not search is exit 3
  (`env fault: cannot look for vendor/ under <rc-dir>: <reason>`). It was "no vendor/ under <rc-dir>", exit 0.
- `scripts/check-client-source.sh`: a BUILD-STATE file with CRLF line endings is read as what it declares, and
  `--build-state ""` is the usage error (exit 2) unless `--declared` is given. Both were
  `WARN  source-of-truth  source_of_truth undeclared`, exit 0.

### Fixed — two silent skips

- `toolbelt/report-module.sh`: a `src/rc/vendor` symlink that leads nowhere gets the
  `ERROR  vendor-floor  env fault (exit 3)` row; it got no row and the report could read CLEAN.
- `scripts/check-installed-line.sh`: a working-tree entry literally named `<ref>:<group>/build.gradle.kts` no longer
  hides the group (its downgrade was not reported), and a group path holding a control character, a double quote
  or a backslash is printed quoted with C escapes, so a path with a newline is one row.
