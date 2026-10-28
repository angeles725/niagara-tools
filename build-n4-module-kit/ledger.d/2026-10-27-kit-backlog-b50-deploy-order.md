<!-- ledger: kit -->
<!-- date: 2026-10-27 -->
<!-- last-change-note: kit-backlog B5.0: ng-deploy runs every check before the copy -->
log: 2026-10-27 · kit-backlog B5.0 (feature doc `odd/tasks/kit-backlog-2026-10-27.md`) → `scripts/ng-deploy.sh` `main` runs the type-count verify, the cache-buster check and the verify-module gate on `build/libs` before `copy_jars`, after the version-bump and schema-risk prechecks; a failing check copies nothing (bats NG-order-gate, NG-order-verify, NG-order-pass in `tests/ng-deploy.bats`). `skill/SKILL.md`, BUILD-LOOP §4/§5 and `build-verify.md` (tool table, deploy section, slotomatic modes line) name the new order; the build-verify slotomatic line no longer claims mode C regenerates -ux slots.
