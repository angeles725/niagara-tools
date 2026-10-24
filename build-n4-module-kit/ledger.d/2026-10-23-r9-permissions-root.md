<!-- ledger: kit -->
<!-- date: 2026-10-23 -->
<!-- last-change-note: quality round 9: fixture module-permissions.xml roots are <permissions> -->
log: 2026-10-23 · quality round 9 → the module-permissions.xml of the MinimalPan-rt, MinimalDash-rt and MinimalDash-ux fixtures (copied by scaffold-module.sh) used the root `<modulePermissions/>`, which the NRE never reads (NModule.java:455 `manifest.elem("permissions")`), so scaffolded modules silently ignored permissions added later; the roots are now `<permissions/>`, matching templates/module-permissions.xml, the course and the wizard. Pinned by tests/permissions-root.bats PR1 (every shipped file) and PR2 (fresh logic + dashboard scaffolds). Retro retros/2026-10-23-r9-permissions-root.md (1Δ, pending). [ev: retro r9-permissions-root Δ1]
