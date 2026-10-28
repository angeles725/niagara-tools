### Changed — CI installs the kit's npm tools from a committed lockfile

- `build-n4-module-kit/toolbelt/eslint/package-lock.json` is now committed, and the CI bats job installs eslint and
  acorn with `npm ci --ignore-scripts` (exactly the locked tree, no dependency lifecycle scripts) instead of
  `npm install`; npm's download cache is keyed on the lockfile. Local installs use the same command.
