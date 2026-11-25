### Added

- rc-scan: `responsive` check for `--profile lan|both` (WARN, FAIL under `--strict`): an HTML entry page without `<meta name="viewport" content="width=device-width">`, no `@media` width query in the module, and a `width`/`min-width` above 360px outside `@media`; `hmi` and `unknown` run no check (issue #1087 R1).
