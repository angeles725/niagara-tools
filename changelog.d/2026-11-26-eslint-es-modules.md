### Fixed
- `report-module.sh` ESLint: a dashboard file that parses only as an ES module (loaded with `<script type="module">`) is linted again with the new `toolbelt/eslint.module.config.mjs` instead of being reported as a `parse` FAIL, and content-hashed bundle output (`name-<8+ hex>.js`) is no longer linted with the project rules. Any other parse error stays a FAIL. (#1157)
