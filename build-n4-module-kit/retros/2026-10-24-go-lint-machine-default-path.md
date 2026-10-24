<!-- review-status: pending -->
<!-- module: kit -->
<!-- deltas: 3 -->
# 2026-10-24 · kit · go-lint-machine-default-path

**Session**: niagara-research quality round 10, branch `r10/machine-default-path-go` from origin/main 2285da0
**Delta count**: 3

## What happened
The owed advisory lint G43 (retro modular-line-efficient-quality Δ5, `METHODOLOGY.md` § rt "No machine- or
session-specific default paths in source", `types/issues-and-gotchas.md` I3) shipped as the rule package
`go/internal/lint/rules/machinedefaultpath` on the shared engine and the wrapper `toolbelt/lint-machine-default-path.sh`.
There was no bash lint to port, so the table tests and the golden cases were written by hand from the doctrine
and watched failing first. The rule reuses the shared scanner only (`javasrc.File.Stripped`, `BlankLiterals` with its
literal callback, `Methods`); no javasrc helper was missing. Decision: **narrow default contexts.** A literal counts
only as an `@NiagaraProperty` defaultValue, a field initializer outside every method body, or a
getProperty/getOrDefault/requireNonNullElse/orElse fallback; a local variable or a call argument does not.

## Evidence
- RED: 20 of 21 finding cases and 2 of 5 golden cases failed on a stub `Check`; GREEN after the rule, then 1 more
  RED/GREEN for the `/home/niagara/` exclusion `[ev: go/internal/lint/rules/machinedefaultpath/machinedefaultpath_test.go]`
- Real trees (2026-10-24): `/home/cristian/modulos_niagara_n4/Cliente` (137 `*.java` outside srcTest and
  worktrees) 0 rows, and a plain grep for the path shapes finds no such literal in it at all; `sources/decompiled`
  (2,706 files) 1 row before the exclusion: niagarad-ext `QnxUsbBackupServlet.java:21`
  `USB_COOKIE_PATH = "/home/niagara/daemon/fp-usb"`, the JACE platform's own account home (false for the
  doctrine: the same on every unit), 0 after `[ev: go/internal/lint/rules/machinedefaultpath/machinedefaultpath.go]`
- Not rows by design on that corpus: `new File("/tmp/daemon_usr1.properties")` and `reloadProperties("/tmp/...")` in
  `NiagaraDaemon.java:1279,1281` and three `new File("/home/niagara/daemon/fp-usb")` calls in
  `QnxUsbBackupServlet.java:62,80,114` (call arguments, not defaults)
- Mutation spot checks: dropping the method-body check, the look-back over earlier lines or the expression path of
  defaultValue each turns 1 to 4 table cases red

## Proposed kit deltas (propose-never-apply)
| Δ | Delta | Target file / § | Token |
|---|---|---|---|
| Δ1 | **G43 reads only three default contexts, on purpose.** A machine path in a local variable, a call argument (`new File("/tmp/x")`), a ternary getenv fallback or a path split over several literals is not a row; the review check in § rt still covers them. Widening a context needs a real-tree row count with each row classified, as here (decompiled: 5 call arguments, 3 `/home/niagara/` and 2 `/tmp/`, would be rows, all platform paths). | `METHODOLOGY.md` § rt (G43 pointer) | `[ev: toolbelt/lint-machine-default-path.sh]` |
| Δ2 | **The shipped G43 differs from the deferred spec in the ledger: Java only, ADVISORY rows.** The deferred spec (ledger.d/2026-10-22-modular-line-efficient-quality.md) also scanned `rc/**/*.js` and `*.properties` and printed WARN rows that fail under `--strict`. This unit was scoped to Java and advisory rows (never counted, `--strict` included). Owed: a `.properties`/`rc` JS pass, if a real tree shows such defaults. Station-name literals stay out of scope: they need the site table (G42) to know which names are a station's. | ledger open_issue G43 | `[ev: go/internal/lint/rules/machinedefaultpath/machinedefaultpath.go]` |
| Δ3 | **G43 is not a report-module member yet.** ER1 (tests/lint-empty-root.bats) picks the wrapper up by its `go_tool_bin` line, so its empty-root ERROR row and exit 3 are already pinned. Wiring it into `report-module.sh` needs a decision first: an `<artifact>/src` with no `*.java` (a JS-only ux part) would be the K20 ERROR row, exit 3, under the engine default. | `toolbelt/report-module.sh` | `[ev: tests/lint-machine-default-path.bats]` |

## Lessons
- A lint with no bash ancestor still gets golden cases: written by hand from the doctrine, they pin the wrapper
  contract (usage, empty root, pruned directories, several roots) the table tests do not reach.
- The first real-tree row was a platform path, not a developer path: classify every row before trusting a shape
  list, and exclude by a test, never by widening the comment.

---
**Status**: PENDING — INDEX row generated from the header: `| 2026-10-24-go-lint-machine-default-path.md | kit | 2026-10-24 | pending | 3 |`
