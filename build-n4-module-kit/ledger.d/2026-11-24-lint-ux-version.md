<!-- ledger: kit -->
<!-- date: 2026-11-24 -->
<!-- last-change-note: lint-ux-version.sh checks that a -ux resource's ?v= matches defaultModuleVersion -->
log: 2026-11-24 · `toolbelt/lint-ux-version.sh <module-root>` (Go rule `uxversion`): FAILs a `?v=` on an html, js or css reference under `*-ux/src/rc/` that differs from the group `defaultModuleVersion` and WARNs a local reference with none (FAIL under `--strict`); external URLs, comments, `rc/vendor`, `rc/ext` and `*.min.js` are not read; waiver `lint-ux-version: allow <check-id> <reason>`. Rule: `types/dashboard.md` "Cache-bust every static asset with the module version", `types/frontend-standard.md` §5 and §11 (the servlet caches images 30 days, research block 29).
