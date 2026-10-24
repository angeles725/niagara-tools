# niagara-tools Go module

Stdlib-only Go behind the kit's thin shell wrappers. A wrapper sources `go-tool.sh`, whose `go_tool_bin`
rebuilds `bin/<tool>` when a Go source is newer, then execs it. CI runs gofmt, vet, test and
`go build -o bin/ ./cmd/...`.

## Layout

| Path | Holds |
|---|---|
| `cmd/<tool>/` | a thin `main` that calls one package's entry point; no logic |
| `internal/cli/` | injectable stdio (`cli.Stdio`) and the `cli.Main` signature every command uses |
| `internal/strerror/` | `Text`: an OS error as C `strerror()` words it in the C locale, and `FindQuote`: a path as GNU find quotes it in that locale (the text find and awk print), shared by `lint` and `jarversion` |
| `internal/javasrc/` | the Java scanner only: `Strip`, `Methods`, `Lines`, `BlankLiterals` and `BlankLine` (literal contents blanked, quotes kept), `AnnotationSkip` (an annotation and its argument list read past char by char; the lint rules share these) and `File` (one file's lines plus its scan results, computed once) |
| `internal/lint/` | the lint engine: `Rule` (per file), `TreeRule` (all the files under a root at once), `Finding` + `Row()`, `Parse` (the `awk_lint_parse` modes `One`/`First`/`Each`/`Raw`, plus `Exact`: one argument, no flag parsed, and `Leading`: only a first `--strict` is a flag, as a bash lint's `[ "${1:-}" = "--strict" ] && shift`), `Walk` (the fail-closed walk of `toolbelt/lib/scan-files.sh`; dot and `*-worktrees` sub-directories are never entered), `Run` (exit 0 / 1 on a FAIL row or a WARN/FAIL row under `--strict` / 3 env, which wins over 1; PASS and ADVISORY rows never count) and `Command` (n4lint) |
| `internal/lint/rules/<rule>/` | one lint rule: its `Check`, its `Spec()` and its table tests |
| `internal/lint/rules/` | `All()`, the list of registered rules |
| `internal/lint/rules/linescan/` | shared by the line-matching rules (ports of `awk_lint_each` / `awk_lint_strip_run` lints): `Code` (a file's lines with comments blanked and literal contents removed, the view a code token is matched in) and the row texts `Source` (leading space trimmed) and `Trimmed` (both ends) |
| `internal/lint/rules/golden/` | test helper: `Run` checks a rule against its `testdata/*.txt` cases recorded from the bash lint, `Unreadable` its `awk_lint_run` failure contract |
| `cmd/n4lint/` | `n4lint <rule>[,<rule>...] <args>...` or `n4lint --all [--strict] [--] <src-root>...` |

The engine walks each root once and reads and scans each file once for every selected rule. Every row is
`<WARN|FAIL|PASS|ADVISORY>  <check>  <path>:<line>  <message>`, or `<WARN|FAIL|PASS|ADVISORY>  <check>  <path>: <message>` for a
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
   rows with `Line` ≥ 1 are unchanged. `Spec.BarePathRow` prints a `Line` 0 finding as `<path>  <msg>`
   instead (two spaces, no colon), the row of lint-structure. Severity `Pass` prints `PASS` and never affects the exit code.
   `Spec.UsageExit` is the exit code of a usage error when the rule runs alone (0 means 3).
   A directory the walk cannot read prints `<lint>: cannot list every file under <root>: find: '<dir>': <reason>`
   (`<dir>` quoted as GNU find does under `LC_ALL=C`: `\'`, `\\`, `\n`-style C escapes, 3-digit octal for
   other control bytes and bytes from 0x80) and, by default, stops the rule: no rows, no later root, exit 3,
   as `scan_files_or_exit` does. Set `Spec.ContinueOnWalkError` only when the bash lint used `awk_lint_run`,
   which scans the files it did list (env line, rows, next root, exit 3), and write that reason in a comment
   beside it (reference: `rules/nullcontext`).
   Severity `Advisory` prints `ADVISORY` and, like `Pass`, never affects the exit code, `--strict` and
   `--all` included: a note that must not turn into a failure.
   An empty root is an error, never a clean pass, for every rule (the K20 guard of
   `lint-silent-protection.sh`, made the engine default after the 2026-10-22 truth audit): a root where
   the rule selects no file prints `ERROR  <check>  <root>  <msg>` on stdout and stops the rule like a walk
   error (no later root, exit 3). `Spec.EmptyError` (`lint.EmptyRow{Check, Msg}`) only shapes that row:
   `<check>` is `Check`, or the rule ID when empty; `<msg>` is `Msg`, or by default
   `no <pattern>[ or <pattern>...] sources found (wrong path or empty source tree)` from `Select.Names`.
   There is no opt-out: every registered rule reads source files and has no meaning on a root without
   them; a future rule that legitimately accepts an empty root adds an explicit, commented Spec field.
   It is decided per root after the walk, so a walk error wins and prints no ERROR row. `Spec.BareUsage` prints only the usage line on an unknown flag
   (`-h` included), for a bash lint whose `-*)` case printed the usage alone; the exit stays `UsageExit`.
   Every engine feature lands as one `Spec` field (or severity) plus a doc and a table test; a rule opts
   in by setting it, never by a private copy.
   A lint whose report is ordered by check rather than by file implements `lint.TreeRule` instead
   (reference port: `rules/timers`, lint-timers, which also keeps gawk's matching and `for (k in a)` order;
   that order lives in `rules/awkorder`, which the per-file `rules/configsanity` uses too):
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
   A lint that checks a module's layout rather than its sources (directories, files above or beside the
   root) implements `lint.RootRule` (`CheckRoot(root string)`; reference port: `rules/structure`) and sets
   `Spec.Root`: the runner walks nothing for it, so `Select` and `ContinueOnWalkError` do not
   apply; a root with nothing it checks returns `lint.ErrEmptyRoot`, which prints the `Spec.EmptyError` row
   (`Msg` required) and exits 3 like any empty root; it reads files itself and returns a non-nil error, which
   fails closed like a `TreeRule` error, for anything it cannot read. `n4lint --all` (source roots) leaves it
   out; it runs by name or in a rule list.
2. Add the `Spec()` call to `rules.All()`.
3. Write the tests first and watch them fail: record every bats run of the old script as a golden case
   (point the suite's lint variable at a wrapper that runs the old script and writes args, exit, stdout,
   stderr and the tree in the `golden` format; reference: `rules/setnullord`), add edge trees, and call
   `golden.Run(t, Spec(), <min cases>)`. The format is in the `rules/golden` package comment (a
   `-- <path> (noeol) --` section is a file whose last line has no newline). A port of an `awk_lint_run`
   lint also calls `golden.Unreadable(t, Spec(), <src>)` for its walk-error contract. Recorded rows catch
   what hand-written expectations miss (row order, gawk hash order, read and grep quirks); a port whose
   tests restate the bats assertions as substrings, or that copies the golden parser into its own
   package, is a review finding. `[ev: retro go-lint-timers Δ1]` `[ev: retro go-lint-ord-ports Δ2]`
4. Make `toolbelt/lint-<name>.sh` a wrapper like `lint-null-context-write.sh`: keep its header (description,
   Usage, Exit, `# Mutation:` lines) and exec `n4lint <rule-id> "$@"` via `go_tool_bin`.
5. Prove the old and new scripts print the same bytes on the bats fixtures and real trees; list every
   intended difference in the PR.
6. Fix the bash lint's truth bugs only after the byte-identical port, one commit each, RED first: a
   hand-written `testdata/truth-*.txt` case (same format) plus a bats test whose `# Mutation:` pin names it.
   A line-matching rule reads `linescan.Code(f)` (or `f.Stripped()` when the token lives in a literal),
   never the raw line cut at its first `//` (string-unaware: a `//` in "http://h" hid the rest of the line).

Do not copy the walker, the formatter, the argument parser or the comment stripper into a rule package;
change the shared code instead.
