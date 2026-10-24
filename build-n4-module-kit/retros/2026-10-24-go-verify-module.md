<!-- review-status: folded -->
<!-- module: kit -->
<!-- deltas: 3 -->
# 2026-10-24 · kit · go-verify-module

**Session**: niagara-research quality round 10 (Ayudante1), branch `r10/verify-module-go` from origin/main a4d97437
**Delta count**: 3

## What happened
`toolbelt/verify-module.sh` (747 lines: THE jar gate, 19 checks, the `coverage` subcommand and `--plano`) was
ported to Go as `go/internal/verifymodule` + `go/cmd/verify-module`, byte for byte (phase A). The script is now
the `go_tool_bin` wrapper. Its false results are listed below as open issues for phase B; none was fixed here.

- **Golden format.** The golden case format held text files only. It now spells `(base64)`, `(class N)`,
  `(dir)` and `(jar)` sections; a jar section lists its entries in archive order with their method (stored or
  deflated), so `stored`, `bytecode` and entry order replay exactly.
- **Goldens.** 175 cases recorded from the bash on origin/main: every run of verify-module.bats, facets-lint.bats,
  plano-check.bats and verify-coverage.bats, plus edge trees built in scratch from the kit fixture helpers.
- **Shared pieces.** `lint.Walk` for java_files, `lint.Basename`, `strerror`, and the jarversion zip reader,
  which now exports `Open`, `Index`, `Head` and `Read` instead of a second copy in the new package.
- **Text model.** The checks read their input as GNU grep 3.11 printed it under C.UTF-8. A line with an encoding
  error (a Latin-1 comment) still matches and counts, but grep never prints it. A file with a NUL byte prints
  nothing. `$(...)` drops NUL bytes. `sort -V` is a port of gnulib filevercmp, checked against 289 GNU pairs.
- **Bash arithmetic.** An operand bash cannot evaluate (`08`, `2.5`) inside the `coverage` or `--plano` block made
  bash drop that whole block and parse the rest as jar arguments (exit 2 "not a jar"). An unset name ended the
  run with exit 1. The port does the same; this is the strangest fail-open of the script.
- **Usage.** The bash printed its own lines 2-20 with sed. They are embedded as `usage.txt`, so the wrapper
  header could be rewritten without changing `--help`.

Declared stderr differences (stdout and exit unchanged): bash's `<script>: line N:` prefix on arithmetic errors;
the walk line quotes paths in the C locale (as `lint.Walk` does for every Go lint), where find under C.UTF-8
printed curly quotes; bash's NUL-byte warning, gawk's multibyte warning and the awk/test errors on an unreadable
.java are not reproduced. An unreadable META-INF/module.xml entry is exit 3 `cannot read`, where bash exited
with unzip's own status.

## Evidence
- Identity: 175 goldens + `TestUnreadableSource`, `TestArithmeticFallThrough`, `TestVerLE` green `[ev: go/internal/verifymodule/testdata]`
- Suites unchanged and green on the wrapper: verify-module, facets-lint, plano-check, verify-coverage, go-wrapper-symlink, build-sh, report-module, ng-deploy `[ev: tests/verify-module.bats]`
- Parity (local, not committed), old bash vs Go: client tree A, 960 vendor jars with `--stored --target-version 4.14 --strict`: 960/960 identical stdout and exit, 959/960 identical stderr (the one difference is the two bash/gawk warnings above). Client tree B, 3 modules: 20/20 runs identical in stdout, stderr and exit (15 gate runs over three flag sets, 5 `--plano`) `[ev: retro go-verify-module]`

## Proposed kit deltas (propose-never-apply)
| Δ | Delta | Target file / § | Token |
|---|---|---|---|
| Δ1 | A command that reads jars is recorded with `(jar)` golden sections: entries in archive order with their method, `(class N)` for the fixture class headers, `(base64)` for any other binary, `(dir)` for an empty directory. Record only the paths an argument names, so the jar's source tree does not double the golden | `go/README.md` § Adding a lint rule (step 3) | `[ev: go/internal/lint/rules/golden/golden.go]` |
| Δ2 | Before porting a bash tool that greps under C.UTF-8, record one edge golden with a Latin-1 line and one with a NUL byte: grep counts both but prints neither, so `grep -n` and `grep -o` rows vanish silently | `go/README.md` § Adding a lint rule (step 3) | `[ev: go/internal/verifymodule/testdata/edge-src-shapes.txt]` |
| Δ3 | Before porting `$(( ))` over input text, record an operand that bash cannot evaluate (`08`, `2.5`, a bare name): in a non-interactive bash, the error drops the whole top-level command it occurs in, and the script goes on | `go/README.md` § Adding a lint rule (step 3) | `[ev: go/internal/verifymodule/verifymodule_test.go]` |

## Lessons
- A command substitution drops NUL bytes and a grep pipeline drops badly encoded lines, so a bash gate can see less input than the file holds and still print a clean verdict.
- A bash arithmetic error does not stop the script: it drops the top-level command around it, and a subcommand can then fall through into the main flow.
- When a port needs the same zip or walk logic, export it from the existing package, as with `jarversion.Open`, `Index`, `Head` and `Read`. Never copy it.

## Fold
- Δ1 → `BUILD-LOOP.md` § 5 Golden cases come from the recorder: `(jar)`, `(class N)`, `(base64)` and `(dir)` sections.
- Δ2 → `BUILD-LOOP.md` § 5 Record the shell's silent input drops as edge goldens: grep under C.UTF-8 with a Latin-1 line or a NUL byte.
- Δ3 → `BUILD-LOOP.md` § 5 same bullet: a `$(( ))` operand bash cannot evaluate drops the top-level command.
- The phase B truth fixes stay open issues in `ledger.d/2026-10-24-go-verify-module.md`; the `--src` per-file reads keep the existing `[deferred: retro r9-m1-walkers Δ2]` token, now scoped to verify-module only.

---
**Status**: FOLDED 2026-10-24 (branch `docs/r10-fold-tail`; log in `ledger.d/2026-10-24-r10-fold-tail.md`) — Δ1-Δ3 cited in `BUILD-LOOP.md` § 5.
