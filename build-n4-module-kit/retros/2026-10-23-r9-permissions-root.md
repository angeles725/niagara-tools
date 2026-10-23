<!-- review-status: pending -->
<!-- module: kit -->
<!-- deltas: 1 -->
# 2026-10-23 · kit · r9-permissions-root

**Session**: niagara-research quality round 9 (Ayudante3), branch `r9/permissions-root` from origin/main 6b0e59a
**Delta count**: 1

## What happened
The three fixture parts that `scaffold-module.sh` copies (MinimalPan-rt, MinimalDash-rt, MinimalDash-ux) shipped
`module-permissions.xml` with the root `<modulePermissions/>`. The NRE reads only `manifest.elem("permissions")`
(NModule.java:455) and the build plugin copies the file's root into META-INF/module.xml unchanged, so every module
scaffolded from the kit would silently ignore any permission added later. `templates/module-permissions.xml`, the
course files and the Workbench wizard all use `<permissions>`. The fixtures now use `<permissions/>`; scaffolded
modules inherit it (the scaffold copies the fixture tree).

## Evidence
- RED: `bats tests/permissions-root.bats` → PR1 named the 3 fixtures (`<modulePermissions>`), PR2 found
  `DemoPerm-rt/module-permissions.xml has root <modulePermissions>` in a fresh logic scaffold
  `[ev: tests/permissions-root.bats PR1]`
- GREEN after renaming the 3 roots; PR2 scaffolds logic and dashboard modules and checks every permissions file
- [CERT] organized/baja/baja/decompiled/com/tridium/sys/module/NModule.java:455 `XElem permissions = manifest.elem("permissions");`
- Not changed here: `types/logic-authoring.md` (~:441) still shows a `<modulePermissions>` example with a
  `<permission name actions>` shape; its correct content is a domain-doc change, handed to the knowledge owner

## Proposed kit deltas (propose-never-apply)
| Δ | Delta | Target file / § | Token |
|---|---|---|---|
| Δ1 | A fixture file the scaffold copies must use the element names the NRE reads; pin every shipped/scaffolded module-permissions.xml root (G98) | `tests/permissions-root.bats` PR1/PR2 | `[ev: tests/permissions-root.bats PR1]` |

## Lessons
- A fixture is a template in disguise: a wrong root there reaches every scaffolded module.
- Checking the scaffold's OUTPUT (PR2), not only the fixture files, keeps the pin valid if the scaffold changes source.

---
**Status**: PENDING — INDEX row generated from the header: `| 2026-10-23-r9-permissions-root.md | kit | 2026-10-23 | pending | 1 |`
