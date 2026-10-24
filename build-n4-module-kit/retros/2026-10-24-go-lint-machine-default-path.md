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
A first cut read Java only with ADVISORY rows; the coordinator then ruled the deferred spec
(ledger.d/2026-10-22-modular-line-efficient-quality.md) the contract, so the shipped rule WARNs (exit 1 under
`--strict`) and also reads rc JavaScript and `.properties` (Δ2, closed). Two shared packages landed for that:
`go/internal/jssrc` (a literal-aware JavaScript tokenizer) and `go/internal/props` (a java.util.Properties.load
reader). lint-structure's L10 was not a reusable properties reader (a per-line regex kept byte-identical to the
bash lint), so structure is unchanged.
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
- Spec follow-up RED: 25 table and golden cases failed before the WARN/JS/properties change; 28 jssrc and props
  cases failed on stubs `[ev: go/internal/jssrc/jssrc_test.go]` `[ev: go/internal/props/props_test.go]`
- Real trees after the follow-up: Cliente 9 rows, all true: `niagara_user_home=C:\\Users\\equipo\\...` in 9 module
  `gradle.properties` (a developer's Windows home; L10 flags the same lines); no rc JavaScript or Java row;
  sources/decompiled 0 rows (exit 0 under `--strict`)
- Mutation spot checks (follow-up): dropping the rc/ path check, the build/node_modules prune, the property
  `{`/`,` check, the template `${` check or the case-insensitive flag each turns 1 to 8 cases red
- Mutation spot checks: dropping the method-body check, the look-back over earlier lines or the expression path of
  defaultValue each turns 1 to 4 table cases red

## Proposed kit deltas (propose-never-apply)
| Δ | Delta | Target file / § | Token |
|---|---|---|---|
| Δ1 | **G43 reads only three default contexts, on purpose.** A machine path in a local variable, a call argument (`new File("/tmp/x")`), a ternary getenv fallback or a path split over several literals is not a row; the review check in § rt still covers them. Widening a context needs a real-tree row count with each row classified, as here (decompiled: 5 call arguments, 3 `/home/niagara/` and 2 `/tmp/`, would be rows, all platform paths). | `METHODOLOGY.md` § rt (G43 pointer) | `[ev: toolbelt/lint-machine-default-path.sh]` |
| Δ2 | **CLOSED 2026-10-24: G43 follows the deferred spec.** The first cut read Java only with ADVISORY rows; the deferred spec (ledger.d/2026-10-22-modular-line-efficient-quality.md) is the contract, so the rule WARNs (exit 1 under `--strict`), its message starts with the spec's row text, prefixes are case-insensitive, and `rc/**/*.js` (const/let/var initializers at any depth, since rc code lives in IIFE/AMD wrappers, and object-literal property defaults) and every `*.properties` value are read through shared scanners. Kept as declared differences: the `/home/niagara/` exclusion and the superset of prefixes (`/Users/`, any `/mnt/<drive>/`, any drive letter). Station-name literals stay out of scope: they need the site table (G42). | ledger open_issue G43 | `[ev: go/internal/lint/rules/machinedefaultpath/machinedefaultpath.go]` |
| Δ3 | **G43 is not a report-module member yet.** ER1 (tests/lint-empty-root.bats) picks the wrapper up by its `go_tool_bin` line, so its empty-root ERROR row and exit 3 are already pinned. Wiring it into `report-module.sh` needs a decision first: it reads module roots (gradle.properties sits beside the profiles), while report-module runs members per `<artifact>/src`. | `toolbelt/report-module.sh` | `[ev: tests/lint-machine-default-path.bats]` |

## Lessons
- A lint with no bash ancestor still gets golden cases: written by hand from the doctrine, they pin the wrapper
  contract (usage, empty root, pruned directories, several roots) the table tests do not reach.
- The first real-tree row was a platform path, not a developer path: classify every row before trusting a shape
  list, and exclude by a test, never by widening the comment.

---
**Status**: PENDING — INDEX row generated from the header: `| 2026-10-24-go-lint-machine-default-path.md | kit | 2026-10-24 | pending | 3 |`
