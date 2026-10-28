### Fixed — rc-scan module-map checks: anchored services exemption, third-party files skipped

- `fetch-outside-services` exempts only files under `rc/js/services/`; a `fetch(` in `js/pages/services/` or
  `rc/services/` is now reported (it was exempt through any `/services/` path segment).
- `fetch-outside-services` and `js-file-size` skip third-party files: `rc/vendor/**` (its HTML included, where an
  inline-script `fetch(` was reported), `rc/ext/**` and `*.min.js`.
