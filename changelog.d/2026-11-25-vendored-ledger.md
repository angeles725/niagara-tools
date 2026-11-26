### Added — lint-vendored

- New `toolbelt/lint-vendored.sh [--strict] [--defects <file>] <rc-dir>` (issue #1123): reads `rc/vendor/VENDORED.md`,
  a markdown table (`| File | Library | Version | Source URL |`, one row per vendored file, version an r-number or
  a semver), and WARNs a vendored file whose library and version the kit's known-defects table lists, naming the
  defect and the fix. The table is the data file `data/vendored-defects.md` (library, file pattern, affected
  versions, defect, fix); its first row is three.js `OrbitControls*.js` below r161 (zoom wrong at
  `devicePixelRatio` < 1, the issue's evidence). It also WARNs a file under `rc/vendor/` that no row lists, vendor
  files with no `VENDORED.md` (once, saying how to create it), and a row with an empty cell or an unreadable
  version; an rc dir with no vendor files is clean. A new `n4lint` rule (`go/internal/lint/rules/vendored`), apart
  from `lint-vendor-floor`, which parses the files with node and acorn. `types/frontend-standard.md` § Vendored
  libraries defines the ledger.
