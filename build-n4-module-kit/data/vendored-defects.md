# Known defects of vendored browser libraries

Read by `toolbelt/lint-vendored.sh` (`--defects`). A vendored file that `rc/vendor/VENDORED.md` lists at a library
and version in this table gets a WARN naming the defect and the fix (`types/frontend-standard.md`, Vendored
libraries). Add a row when a vendored library ships a defect that a build, a test and `lint-vendor-floor` do not
show, and cite the evidence in the commit. The first row is from issue #1123: three.js `OrbitControls` `_getZoomScale`
divided by the truncated `window.devicePixelRatio` (0 below 100 % zoom); r161 fixed it (the issue's evidence, not
re-checked against three.js here).

- **Library**: the name as the ledger's Library cell spells it (case-insensitive).
- **File**: a `*` / `?` pattern matched on the ledger file's base name (`*` for every file of the library).
- **Affected**: comparisons joined by commas, all of which must hold: `<`, `<=`, `>`, `>=`, `=`, then a version.
  A version is a three.js r-number (`r161`) or a semver (`1.4.2`; a leading `v` and a missing minor or patch
  are fine). A release is above its pre-releases. An r-number does not compare with a semver: such a pair is
  a WARN "check by hand", so write the row in the form the ledger uses.
- **Fix**: what to do, in one line.

| Library | File | Affected | Defect | Fix |
|---|---|---|---|---|
| three.js | OrbitControls*.js | < r161 | zoom scale is wrong at devicePixelRatio < 1 | upgrade to >= r161 |
