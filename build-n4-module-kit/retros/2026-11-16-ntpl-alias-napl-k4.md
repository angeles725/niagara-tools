<!-- review-status: folded -->
<!-- module: kit -->
<!-- deltas: 4 -->
# 2026-11-16 · kit · ntpl-alias-napl-k4

**Session**: kit row K4 of the niagara-research retro "coordination-stretch-deltas" (2026-11-15): the second half of the template check, the modules a template's bog and Px files use, and `.napl` application archives.
**Delta count**: 4

## What happened
The first slice of lint G92 compared a template manifest's `<dependency>` rows with the module's build files.
The other half of the 2026-10-23 G92 spec, what the template itself uses, was not built (gate-followups-and-
template-lints Δ13). This slice adds it as the sub-check `ntpl-uses` on the same walk, dependency reader,
installation reader and row format, and reads `.napl` application archives too.

## Evidence
- A real vendor template (a Spyder VAV `.ntpl`) lists `control-rt`, `alarm-rt`, `honIrmConfig-rt` and the like,
  while its bog aliases name modules without a part (`m="c=control"`, `m="irmn=honIrmConfig"`), and its Px
  file imports modules the same way: a dependency row names a part, a use names a module. Run without an
  installation, the new check reports only `kitIo` and `platform`, both stock modules every installation
  ships. `[ev: go/internal/lint/rules/ntplmanifest/uses.go]`
- Differential, base `e0817798` against the branch: `report-module --lint-only --niagara-home` on a module
  without templates is byte-identical; on a module with a `.ntpl` and a `.napl` whose bog uses an unlisted
  module, the only change is the member's PASS row becoming the two new `ntpl-uses` FAIL rows and the counts.
  `[ev: go/internal/lint/rules/ntplmanifest/uses_test.go]`

## Proposed kit deltas (propose-never-apply)
| Δ | Delta | Target file / § | Token |
|---|---|---|---|
| Δ1 | APPLIED 2026-11-16. K4: sub-check `ntpl-uses`: a module that `template.bog` (aliases, `module://` ORDs) or a `*.px` entry (imports, `module://` ORDs) uses, that no `<dependency>` names (a part counts) and the installation does not ship, is a row per entry (FAIL with the installation's list, WARN without); baja and the module's own parts are exempt; an entry that does not parse is a FAIL row | `toolbelt/lint-ntpl-manifest-deps.sh` header · Go `internal/lint/rules/ntplmanifest` `uses.go`, `usesrows.go`, `archive.go` | `[ev: go/internal/lint/rules/ntplmanifest/usesrows.go]` |
| Δ2 | APPLIED 2026-11-16. K4: `.napl` application archives are read as templates (same zip), the extension in any letter case | Go `internal/lint/rules/ntplmanifest` `messages.go` | `[ev: go/internal/lint/rules/ntplmanifest/messages.go]` |
| Δ3 | OPEN. Still not read from the 2026-10-23 G92 spec: a template module `.jar` as input, and the guide's `template.manifest.xml` spelling as a fallback name; the 2026-10-23 open_issue can be narrowed to those two | `toolbelt/lint-ntpl-manifest-deps.sh` · `ledger.d/2026-10-23-r8-u10-platform-templates.md` | `[ev: build-n4-module-kit/ledger.d/2026-10-23-r8-u10-platform-templates.md]` |
| Δ4 | APPLIED 2026-11-16. Review of #702: each bog or Px entry was bounded at 32 MiB but an archive's entries were not bounded in all, so one with many entries could inflate without limit; the entries of one archive now share 128 MiB, past which one FAIL row names the first entry not read and no later entry is opened; only the modules of an entry are kept, never its bytes | Go `internal/lint/rules/ntplmanifest` `usesread.go` | `[ev: go/internal/lint/rules/ntplmanifest/usesread.go]` |
