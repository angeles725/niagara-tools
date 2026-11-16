<!-- review-status: pending -->
<!-- module: kit -->
<!-- deltas: 5 -->
# 2026-11-16 · kit · ntpl-jar-manifest-k4b

**Session**: the open remainder of row K4 of the 2026-11-15 coordination-stretch retro: lint-ntpl-manifest-deps
reads the guide's manifest spelling and takes a template module jar as input.
**Delta count**: 5

## What happened
#702 left two parts of K4 open. The Templates guide spells the manifest `template.manifest.xml` (niagara-help
4.13.2.18 `Templates/ExampleTemplate.manifest.xmlFileTem-9BAE5349.txt:1-17`), while shipped archives and the
guide's own file-structure page use `template-manifest.xml` (`ExistingFileAndComponentStructuresF-B8CDA09C.txt:13`).
A template module is one `.jar` of `.ntpl` files (Make Module, `MakingATemplateModule-8A3D4B56.txt:7`), and the
lint engine refused any root that is not a directory, so the jar needed an opt-in in the shared engine.

## Evidence
- RED before GREEN: an archive with only the dotted manifest name was "lacks template-manifest.xml"; a jar root
  was exit 3 "not a directory"; the engine test fails both with the opt-in ignored and with it accepted for every
  spec. `[ev: go/internal/lint/fileroot_test.go]` `[ev: go/internal/lint/rules/ntplmanifest/jar_test.go]`
- The two bats files keep the 16 cases of the 213-line suite byte-equal with the same results; NTPL15 adds the jar.
  `[ev: tests/lint-ntpl-manifest-deps-paths.bats]`

## Proposed kit deltas (propose-never-apply)
| Δ | Delta | Target file / § | Token |
|---|---|---|---|
| Δ1 | APPLIED 2026-11-16. An archive without `template-manifest.xml` is read through `template.manifest.xml`; with both, the hyphenated one alone | Go `internal/lint/rules/ntplmanifest` `archive.go` · `toolbelt/lint-ntpl-manifest-deps.sh` header | `[ev: go/internal/lint/rules/ntplmanifest/manifestname_test.go]` |
| Δ2 | APPLIED 2026-11-16. A template module jar as the root: each `*.ntpl`/`*.napl` entry checked, rows `<jar>!/<entry>`, 32 MiB per template and 128 MiB in all, dependencies against the installation alone; engine opt-in `Spec.FileRoot`, set by this rule only | Go `internal/lint` `spec.go`, `args.go`, `roots.go`, `run.go` · `internal/lint/rules/ntplmanifest` `jar.go`, `archive.go`, `rows.go` · wrapper header | `[ev: go/internal/lint/rules/ntplmanifest/jar.go]` |
| Δ3 | APPLIED 2026-11-16. `types/logic-authoring.md` § Shipping a template module listed "a template module jar as input" as owed and showed only `<module-root>`; it now gives the jar input, its rows and bounds, and the dotted spelling, and closes both owed G92 items | `types/logic-authoring.md` § Shipping a template module | `[ev: build-n4-module-kit/types/logic-authoring.md]` |
| Δ4 | APPLIED 2026-11-16 (#710 verifier A1-A3). The on-disk bound of a `.ntpl` is read from its metadata before the file is opened again, so an unreadable archive over 32 MiB is its FAIL row (exit 1) as on main, not exit 3; every ntpl-manifest and ntpl-uses message names the manifest entry read (`template.manifest.xml` when the fallback was used); the usage line says `<module-root \| template-module.jar>` | Go `internal/lint/rules/ntplmanifest` `archive.go`, `manifest.go`, `rows.go`, `usesrows.go`, `messages.go` | `[ev: go/internal/lint/rules/ntplmanifest/untrusted_test.go]` `[ev: go/internal/lint/rules/ntplmanifest/manifestname_test.go]` |
| Δ5 | OPEN (#710 verifier A4). Many tiny templates in one jar that each inflate a lot stay inside the byte budgets but cost CPU per template; a bound on the template count or on the inflation ratio is a follow-up | Go `internal/lint/rules/ntplmanifest` `jar.go` | `[ev: go/internal/lint/rules/ntplmanifest/jar.go]` |
