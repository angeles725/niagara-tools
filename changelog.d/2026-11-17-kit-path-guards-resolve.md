### Fixed — path guards: resolution after "..", nothing made before the checks

- `orient-guard.sh`: once a `..` after a missing directory climbs back into existing directories, their
  symlinks are followed again (as `realpath -m`), so `<prefix>/missing/../linkout/x` with `linkout` leaving the
  prefix FAILs where it leads. A final newline is part of a name: a sibling `Cliente\n` is not the prefix.
- `mirror-niagara-home.sh`: every spelling of the mirror dir is judged by that same walk, and the guards, the
  wipe and the rebuild all use the judged path, so `ml/../o` (with `ml` a link to `<out>/o`) refreshes the
  mirror at `<out>/o` instead of deleting it and failing. Nothing is created before every check has passed
  (a refused `missing/../srclink/a/b/c/x` or a relative source leaves no directory). A source or mirror dir
  starting with `-` (`-` alone too) is a usage error (exit 1).
