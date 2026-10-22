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
| `internal/lint/` | the lint engine: `Rule`, `Finding` + `Row()`, `Parse` (the `awk_lint_parse` modes `One`/`First`/`Each`/`Raw`), `Walk` (the fail-closed walk of `toolbelt/lib/scan-files.sh`; dot and `*-worktrees` sub-directories are never entered), `Run` (exit 0 / 1 under `--strict` / 3 env) and `Command` (n4lint) |
| `internal/lint/rules/<rule>/` | one lint rule: its `Check`, its `Spec()` and its table tests |
| `internal/lint/rules/` | `All()`, the list of registered rules |
| `cmd/n4lint/` | `n4lint <rule>[,<rule>...] <args>...` or `n4lint --all [--strict] [--] <src-root>...` |

The engine reads and scans each file once for every selected rule, and every rule prints the same row:
`<WARN|FAIL>  <rule-id>  <path>:<line>  <message>`.

## Adding a lint rule

1. Create `internal/lint/rules/<name>/` with a type that implements `lint.Rule` (`ID`, `Severity`,
   `Check(*javasrc.File) ([]lint.Finding, error)`), plus `Spec() lint.Spec` carrying the parse mode,
   usage line and file selection of the bash lint it replaces. `Check` sets `Line` and `Msg`, and
   `Severity` only on a finding that differs from the rule's `Severity()` (a rule that emits WARN and
   FAIL rows); the engine fills an unset one. Return a non-nil error only when the file cannot be
   analyzed (e.g. unbalanced braces): the engine prints `<lint>: cannot scan <path>: <rule-id>: <error>`
   on stderr and exits 3, as for an unreadable file. Never drop a file silently.
2. Add the `Spec()` call to `rules.All()`.
3. Write the table tests first (port every bats case) and watch them fail.
4. Make `toolbelt/lint-<name>.sh` a wrapper like `lint-null-context-write.sh`: keep its header (description,
   Usage, Exit, `# Mutation:` lines) and exec `n4lint <rule-id> "$@"` via `go_tool_bin`.
5. Prove the old and new scripts print the same bytes on the bats fixtures and real trees; list every
   intended difference in the PR.

Do not copy the walker, the formatter, the argument parser or the comment stripper into a rule package;
change the shared code instead.
