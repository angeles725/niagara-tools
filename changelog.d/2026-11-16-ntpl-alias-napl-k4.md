### Added — the template lint checks what a template uses, and reads .napl archives

- `toolbelt/lint-ntpl-manifest-deps.sh`: new sub-check `ntpl-uses`. A module that `template.bog` uses (each
  `m="<alias>=<module>"`, each `module://` ORD) or a `*.px` entry uses (each `<import><module name>`, each
  `module://` ORD) must be named by a `<dependency>` of the manifest (a part counts: `control-rt` lists
  `control`), unless the installation ships it (`--niagara-home`, else `NIAGARA_HOME`), it is `baja`, or it is
  one of the module's own parts. One row per entry, names sorted: FAIL with the installation's module list, WARN
  without it. A bog or Px entry that is not well-formed XML, or inflates past 32 MiB, is a FAIL row naming it.
- `.napl` application templates (the same zip) are checked like `.ntpl`, the extension in any letter case.
- `report-module` relays the new rows through its existing template lint member.
