<!-- ledger: kit -->
<!-- date: 2026-10-27 -->
<!-- last-change-note: kit-backlog B5.1a: committed eslint lockfile, CI npm ci --ignore-scripts -->
log: 2026-10-27 · kit-backlog B5.1a (feature doc `odd/tasks/kit-backlog-2026-10-27.md`; review fix-forward of 78abca8e R1-001) → `toolbelt/eslint/package-lock.json` committed (77 packages, every `resolved` on registry.npmjs.org, no install scripts; `npm ci --offline` reproduces eslint 10.11.0 + acorn 8.18.0), `toolbelt/eslint/.gitignore` no longer ignores it, the CI step runs `npm ci --ignore-scripts` with the npm cache keyed on the lockfile (bats CL19 in `tests/ci-local.bats`).
