# niagara-tools Go module

Stdlib-only Go behind the kit's thin shell wrappers. A wrapper sources `go-tool.sh`, whose `go_tool_bin`
rebuilds `bin/<tool>` when a Go source is newer, then execs it. CI runs gofmt, vet, test and
`go build -o bin/ ./cmd/...`.

## Layout

| Path | Holds |
|---|---|
| `cmd/<tool>/` | a thin `main` that calls one package's entry point; no logic |
| `internal/cli/` | injectable stdio (`cli.Stdio`) and the `cli.Main` signature every command uses |
| `internal/javasrc/` | the Java scanner only: `Strip`, `Methods`, `Lines` and `File` (one file's lines plus its scan results, computed once) |
| `internal/lint/` | the lint engine: `Rule` (per file), `TreeRule` (all the files under a root at once), `Finding` + `Row()`, `Parse` (the `awk_lint_parse` modes `One`/`First`/`Each`/`Raw`, plus `Exact`: one argument, no flag parsed), `Walk` (the fail-closed walk of `toolbelt/lib/scan-files.sh`; dot and `*-worktrees` sub-directories are never entered), `Run` (exit 0 / 1 on a FAIL row or a WARN/FAIL row under `--strict` / 3 env, which wins over 1; PASS rows never count) and `Command` (n4lint) |
| `internal/lint/rules/<rule>/` | one lint rule: its `Check`, its `Spec()` and its table tests |
| `internal/lint/rules/` | `All()`, the list of registered rules |
| `cmd/n4lint/` | `n4lint <rule>[,<rule>...] <args>...` or `n4lint --all [--strict] [--] <src-root>...` |

The engine walks each root once and reads and scans each file once for every selected rule. Every row is
`<WARN|FAIL|PASS>  <check>  <path>:<line>  <message>`, or `<WARN|FAIL|PASS>  <check>  <path>: <message>` for a
file-level finding (`Line` 0). With `--all` or a rule list, rules run in registry or requested order and each
rule's rows stay contiguous.

## Adding a lint rule

1. Create `internal/lint/rules/<name>/` with a type that implements `lint.Rule` (`ID`, `Severity`,
   `Check(*javasrc.File) ([]lint.Finding, error)`), plus `Spec() lint.Spec` carrying the parse mode,
   usage line and file selection of the bash lint it replaces. `Check` sets `Line` and `Msg`, and
   `Severity` only on a finding that differs from the rule's `Severity()` (a rule that emits WARN and
   FAIL rows); the engine fills an unset one. Return a non-nil error only when the file cannot be
   analyzed (e.g. unbalanced braces): the engine prints `<lint>: cannot scan <path>: <rule-id>: <error>`
   on stderr and exits 3, as for an unreadable file. Never drop a file silently.
   Row columns: `Finding.Check` is the sub-check shown in the check column (empty shows the rule ID), so
   one rule can report several named checks. `Line` 0 is a file-level finding printed as `<path>: <msg>`;
   rows with `Line` ≥ 1 are unchanged. Severity `Pass` prints `PASS` and never affects the exit code.
   `Spec.UsageExit` is the exit code of a usage error when the rule runs alone (0 means 3).
   A lint whose report is ordered by check rather than by file implements `lint.TreeRule` instead
   (reference port: `rules/timers`, lint-timers, which also keeps gawk's matching and `for (k in a)` order):
   `CheckTree(files []*javasrc.File)` gets every selected file under the root in walk order, and sets
   `Path` on each finding; a non-nil error prints the findings returned with it, then `<lint>: <error>`
   (the rule writes the whole text) and exits 3. A spec sets exactly one of `Rule` and `Tree`, and its
   value implements only that interface (`Command` panics otherwise):

   ```go
   type rule struct{}

   func (rule) ID() string              { return "lint-x" }
   func (rule) Severity() lint.Severity { return lint.Fail }
   func (rule) CheckTree(files []*javasrc.File) ([]lint.Finding, error) {
   	var out []lint.Finding
   	for _, f := range files { // first check, every file
   		out = append(out, lint.Finding{Check: "first", Severity: lint.Pass, Path: f.Path, Msg: "ok"})
   	}
   	return out, nil // then the next check over every file, and so on
   }

   func Spec() lint.Spec {
   	return lint.Spec{Tree: rule{}, Mode: lint.One, Usage: "usage: lint-x.sh <src-root>", UsageExit: 2,
   		Select: lint.Selection{Names: []string{"*.java"}}}
   }
   ```
2. Add the `Spec()` call to `rules.All()`.
3. Write the table tests first (port every bats case) and watch them fail.
4. Make `toolbelt/lint-<name>.sh` a wrapper like `lint-null-context-write.sh`: keep its header (description,
   Usage, Exit, `# Mutation:` lines) and exec `n4lint <rule-id> "$@"` via `go_tool_bin`.
5. Prove the old and new scripts print the same bytes on the bats fixtures and real trees; list every
   intended difference in the PR.

Do not copy the walker, the formatter, the argument parser or the comment stripper into a rule package;
change the shared code instead.
