### Added — rc-scan fetch-outside-services and js-file-size under a module map

- `rc-scan` reads the module map mode above the artifact. Under a map, `fetch-outside-services` reports a
  `fetch(` in rc JavaScript (a `.js` file or an inline `<script>`) outside an `rc/js/services/` file, and
  `js-file-size` a `.js` file over 400 lines: WARN in a `modularity: legacy` module (FAIL under `--strict`), FAIL
  in a `modularity: strict` one. A module with no map gets neither check; its rows are unchanged.
