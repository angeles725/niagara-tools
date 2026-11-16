<!-- ledger: kit -->
<!-- date: 2026-11-16 -->
<!-- last-change-note: the template lint checks the modules a template's bog and Px files use against its manifest, and reads .napl application archives -->
log: 2026-11-16 · template uses and application archives → `toolbelt/lint-ntpl-manifest-deps.sh` adds `ntpl-uses`: a module `template.bog` (aliases, `module://` ORDs) or a `*.px` entry (imports, `module://` ORDs) uses that no `<dependency>` names (a part counts) and the installation does not ship is a row per entry (FAIL with `--niagara-home`/`NIAGARA_HOME`, WARN without; baja and the module's own parts exempt; an entry that does not parse is FAIL), and `.napl` application archives are checked like `.ntpl`. Retro ntpl-alias-napl-k4 (Δ1-Δ2 applied, pending review). [ev: retro ntpl-alias-napl-k4 Δ1] [ev: retro ntpl-alias-napl-k4 Δ2]
open_issue: retro ntpl-alias-napl-k4 Δ3 open (2026-11-16): a template module `.jar` as input and the `template.manifest.xml` fallback name.
