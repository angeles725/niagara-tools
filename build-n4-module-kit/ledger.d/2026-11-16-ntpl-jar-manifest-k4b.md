<!-- ledger: kit -->
<!-- date: 2026-11-16 -->
<!-- last-change-note: lint-ntpl-manifest-deps reads a template module jar and the guide's template.manifest.xml spelling -->
log: 2026-11-16 · template lint K4 remainder → `toolbelt/lint-ntpl-manifest-deps.sh` reads an archive's `template.manifest.xml` when `template-manifest.xml` is absent, and takes a template module `.jar` as the root: each `*.ntpl`/`*.napl` entry is checked as a shipped template (rows `<jar>!/<entry>`, 32 MiB per template and 128 MiB in all, dependencies against the installation alone); the lint engine gains the opt-in `Spec.FileRoot`, which only this rule sets; the 213-line bats suite is split in two. Retro ntpl-jar-manifest-k4b (Δ1-Δ2 applied, Δ3 open, pending review). [ev: retro ntpl-jar-manifest-k4b Δ1] [ev: retro ntpl-jar-manifest-k4b Δ2]
open_issue: retro ntpl-jar-manifest-k4b Δ3 open (2026-11-16): `types/logic-authoring.md` still lists the jar input as owed.
