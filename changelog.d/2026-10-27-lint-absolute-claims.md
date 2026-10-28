### Added — lint-absolute-claims

- New `toolbelt/lint-absolute-claims.sh <types-dir>...` (G38): an INFO report of every line holding "never",
  "always", "silently", "forbidden" or "throws" whose bullet or paragraph cites only corpus or retro inference (no
  code, course, devguide, CERT or javap tag and no `.java:` line), for the K33 check against Tridium sources. It
  never fails (exit 0, `--strict` included); a root with no `*.md` exits 3. Not run by `report-module`.
