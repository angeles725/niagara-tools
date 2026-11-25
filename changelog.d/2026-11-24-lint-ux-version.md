### Added — lint-ux-version: a -ux resource's `?v=` must match `defaultModuleVersion`

- `toolbelt/lint-ux-version.sh [--strict] <module-root>` (Go rule `go/internal/lint/rules/uxversion`, run by name;
  `-h`/`--help` prints the usage): reads the html, js and css under a module's `*-ux/src/rc/` and FAILs
  (`stale-version`) every reference whose `?v=` differs from the first uncommented `defaultModuleVersion("X")` of the
  group `build.gradle.kts` (the `vendorversion.SourceVersion` reader; the gradle root or `<repo>/<MOD>` may be the
  argument). A local reference with no `?v=` is a `no-version` WARN, a FAIL under `--strict` (`dashboard.md`: every
  `<link href>`, `<script src>` and image reference carries it). The servlet caches images for 30 days
  (`max-age=2592000`), so a stale suffix keeps a kiosk on the old file after a `-ux`-only deploy.
- A reference is the `href`/`src`/`poster` of a `<link>`, `<script>`, `<img>`, `<source>`, `<video>`, `<audio>`,
  `<image>` or `<input>`, a CSS `url(...)` or `@import`, or a JavaScript string that holds `url(...)` or is one
  asset path. Not references: an external URL, `data:`, `#fragment`, a template, a `?v=` with no value (appended at
  run time), a comment, `rc/vendor`, `rc/ext`, `*.min.js`.
- Waiver: a trailing `lint-ux-version: allow <check-id> <reason>` comment prints a PASS row with the reason. A root
  with no `*-ux/src/rc` file is an `ERROR` row; a missing version is exit 3.
- `types/frontend-standard.md` §11 names the lint.
