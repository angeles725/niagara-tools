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
| `internal/javasrc/` | the Java scanner only: `Strip` (and `StripComments`, which also returns each line's comment text), `Methods`, `Lines`, `BlankLiterals` and `BlankLine` (literal contents blanked, quotes kept), `MaskLiterals` (the same contents replaced by blanks, line lengths kept), `AnnotationSkip` (an annotation and its argument list read past char by char), `NiagaraProperties` (each `@NiagaraProperty` annotation joined to its balanced `)`, with `ParenDepth`), `Tokens` (Strip output as tokens, a literal or text block one token), `Decls` (the method and constructor bodies of each class body, with class, parameter types and `@Override`), `Statements` and `Shape` (a body's statement count and normalized token sequence), `Unescape` (the \\ \" \' \` escapes of a literal's text read) — the lint rules share these — and `File` (one file's lines plus its scan results, computed once; `File.Marker`: a Slot-o-Matic `RegionBegin`/`RegionEnd` marker inside a comment, never in a literal) |
| `internal/jssrc/` | the JavaScript tokenizer only: `Tokens` (comments dropped; strings, template literals with their `${...}` nesting and regex literals as single tokens, each with its line; a template's `Subs` hold each substitution's tokens), `Code`, `HTMLScripts`, and over tokens `Match` (each bracket's closer), `Expr` (where an expression ends, ASI included) and `Bindings` / `Lookup` (lexical const/let/var, parameter and for-of names with the token range each is visible in), for a rule that reads `rc/**/*.js` or a page's scripts |
| `internal/props/` | `Read`: a `.properties` file as `java.util.Properties.load` reads it (continuations, `#`/`!` comments, `=`/`:`/space separators, escapes), each entry with the line it starts on |
| `internal/lint/` | the lint engine: `Rule` (per file), `TreeRule` (all the files under a root at once), `Finding` + `Row()`, `Parse` (the `awk_lint_parse` modes `One`/`First`/`Each`/`Raw`, plus `Exact`: one argument, no flag parsed, and `Leading`: only a first `--strict` is a flag, as a bash lint's `[ "${1:-}" = "--strict" ] && shift`), `Walk` (the fail-closed walk of `toolbelt/lib/scan-files.sh`; dot and `*-worktrees` sub-directories are never entered; a `Selection` takes find `-name` patterns, `Except` (`-not -name`), `Paths` (`-path` on the file) and `PrunePaths` (`--prune-path`)), `Run` (exit 0 / 1 on a FAIL row or a WARN/FAIL row under `--strict` / 3 env, which wins over 1; PASS and ADVISORY rows never count) and `Command` (n4lint) |
| `internal/lint/rules/<rule>/` | one lint rule: its `Check`, its `Spec()` and its table tests |
| `internal/lint/rules/` | `All()`, the list of registered rules |
| `internal/htmltext/` | Python's text layer for the HTML-text lints, byte for byte as CPython 3.15: `Decode` (an `errors="replace"` text read with universal newlines), `Unescape` (`html.unescape`), `DropMarkup` (a page's text with script/style/comments/tags, plus any named elements, dropped and its line numbers kept) and the `str` facts and folds generated from Python's `unicodedata` into `unicode.txt` (`IsSpace`, `IsAlnum`, `IsDecimal`, `IsWord`, `StripMarks`, `CaseFold`, `Strip`, `CollapseSpace`; `HTMLTEXT_PARITY=1` checks the table against a matching python3, `HTMLTEXT_REGEN=1` rewrites it) |
| `internal/xmlsrc/` | the XML scanner the XML and `.bog` lints share: `Open` (a zipped `.bog` is read through the same call) and `Parse` return a well-formed `Doc` or a positioned `*Error`; `Doc.Elements`, `Find`, `Children`, `Text`, `Element.Attr`, `Element.Empty` (written as `<name/>`) and `UnescapeName` (`.bog` slot names) |
| `internal/ktssrc/` | the Kotlin scanner the `*.gradle.kts` lints share: `Tokens` (identifiers, `"..."` and `"""..."""` strings with their `$x`/`${...}` templates, chars, numbers, punctuation; `//` and nested `/* */` comments dropped; each with its byte span) or a line-numbered `*Error`, and `Code`, the tokens back as code lines (comments gone, literals reduced to their quotes) for a line-wise match; used by lint-uberjar-api-conflict and lint-structure L14/L15; `TestSweepRealFiles` (`KTSSRC_SWEEP_ROOTS`) reads real build scripts |
| `internal/signblock/` | `Parts`: the one Go table of the root signing-block requirements (`toolbelt/lib/signing-block.sh`, pinned entry by entry by its test and `tests/lint-structure.bats` LS15-sync) and `Missing` (the parts no code line matches), read by lint-structure L15 and preflight sample-sign |
| `internal/preflight/` + `cmd/preflight/` | the BUILD-LOOP §0.b environment preflight of `toolbelt/preflight.sh` (win-path, jdk8, plugin-pin, jar-lock, version-drift, `--sample` rows, the `--toolchains` row that reads `gradlew -q javaToolchains`): one func per check in `checks`; lsof, `java -version`, `df` and `gradlew` run through the injectable `Host`; `testdata/*.txt` are golden runs recorded from the bash script by `rules/golden/record/record.sh` and replayed by `golden.Runner` |
| `internal/lint/rules/linescan/` | shared by the line-matching rules (ports of `awk_lint_each` / `awk_lint_strip_run` lints): `Code` (a file's lines with comments blanked and literal contents removed, the view a code token is matched in) and the row texts `Source` (leading space trimmed) and `Trimmed` (both ends) |
| `internal/lint/rules/golden/` | test helper: `Runner` replays a `testdata/*.txt` case set hermetically (the case's env only, exact output, the whole post-run tree), `record/record.sh` records one from the pinned bash (`record/README.md`: format and recording); `Run`/`RunMain` are Runner wrappers for the lint goldens, `Unreadable` checks the `awk_lint_run` failure contract, `BuildJar` builds the zip a `(jar)` section spells |
| `internal/bogaudit/` | `bog-audit` (`toolbelt/bog-audit.sh`): a station bog read once through `xmlsrc` and replayed through the python line scanner's state machine it replaced, `--source-dir` read as that scanner read it (`htmltext.Decode`, `javasrc.ParenDepth`), Python's `repr`/`str(float)`/`float()` in `pyfmt.go`; its own CLI, not an n4lint rule (one bog file plus repeatable `--module`, `CHECK<n>  <STATUS>` rows); goldens in `testdata/` (`<path> (zip)` sections are zipped as `file.xml`) |
| `cmd/bog-audit/` | `bog-audit <config.bog\|file.xml> --module <MOD>... [--source-dir <DIR>] [--strict]` |
| `cmd/n4lint/` | `n4lint <rule>[,<rule>...] <args>...` or `n4lint --all [--strict] [--] <src-root>...` |
| `internal/citeparse/` | the kit citation grammar, pure: `Card` returns every evidence pointer of a card (`[ev: …]`/`[deferred: …]` clauses, per-card path aliases, prose citations) as a `Pointer` |
| `internal/citeresolve/` | one `Resolver` per pointer kind over injected `Roots` and one file cache; `Main` of `kit-cite-resolve` (status per pointer with an excerpt; a pre-filter, never a truth verdict) |
| `cmd/kit-cite-resolve/` | `kit-cite-resolve [--kit DIR] [--research DIR] [--modules DIR] [--memory DIR] [--version VER] [--format tsv\|json] [--strict] <card.md>...` |
| `internal/commissioningverify/` | `Main` of `commissioning-verify`, the §6.b commissioning punch-list of toolbelt/commissioning-verify.sh: `sourceDirs` (the `<root>/*/*-rt/src` walk), `sourceChecks` (the four source lints run through their toolbelt wrappers, the toolbelt from `COMMISSIONING_VERIFY_TOOLBELT`), `stationChecks` (bog-audit in process, CHECK11 and CHECK13-20 folded) and `owedRows` (the values-owed table reader); exit 0 clean / 1 any FAIL / 3 usage or env; goldens replayed by `golden.Runner` (`testdata/README.md`) |
| `cmd/commissioning-verify/` | `commissioning-verify <module-root> [--bog <config.bog>] [--module <MOD>] [--values-owed <file>] [--wiring-map <file>] [--strict]` |
| `internal/reportmodule/` | `Main` of `report-module`, the aggregated module conformance report of toolbelt/report-module.sh: `discover` (the artifact walk), the `perArtifact`/`perModule` member tables (`members.go`: script, suffix gate, input, row filter, verdict), `parseRows` (the one row parser, `rows.go`), the special members (`special.go`, `eslint.go`) and `summary`; every member runs through its toolbelt wrapper (`REPORT_MODULE_TOOLBELT`); `Scripts` feeds INDEX.md's Auto column; exit 0 / 1 FAIL / 2 usage / 3 env / 4 INCOMPLETE; goldens replayed by `golden.Runner` (`testdata/README.md`) |
| `cmd/report-module/` | `report-module <module-root> [--target-version x.y] [--console-dir <dir>] [--profile hmi\|lan\|both\|unknown] [--legacy] [--wiring-map <file>] [--first-deploy] [--lint-only]` |
| `internal/schemarisk/` | `Main` of `schema-risk`: the B795 §795.4 slot-diff classifier of toolbelt/schema-risk.sh. `Parse` harvests the `@NiagaraProperty`/`@NiagaraAction`/`@NiagaraTopic` slots of a snapshot (an annotation joined until its parentheses balance, `javasrc.ParenDepth`), `Diff` classifies each change, and the embedded `b795-795.4.csv` gives its verdict; exit 0 SAFE / 1 LOSSY / 2 OUTAGE / 3 usage / 4 env |
| `cmd/schema-risk/` | `schema-risk <before-dir> <after-dir>` |
| `internal/newretro/` | `Command` of `new-retro` (`toolbelt/new-retro.sh`): writes the retro stub, regenerates retros/INDEX.md in process through `retroindex.Generate`/`Output` (the stub is removed again when that fails) and flips retro_pending in BUILD-STATE.src.md (`Flip`, the awk it replaced); the kit root is `$KIT`, a leading `<kit-root>` holding retros/, or `NEW_RETRO_DEFAULT_ROOT`; goldens in `testdata/` recorded by `rules/golden/record/record.sh` and replayed by `golden.Runner` (the date is injected) |
| `cmd/new-retro/` | `new-retro [<kit-root>] <module\|kit> <slug> [--date YYYY-MM-DD] [--deltas N]` |
| `internal/triageconsole/` | `Main` of `triage-console` (`toolbelt/triage-console.sh`): own-module exception triage over station console logs, the awk program it replaced (`scanner`: channels C1 own frame, C2/C2b own tag or own-package path, C3 [sys] fatal, [sys.xml] drift, [sys.registry] missing own class; `--site` classes in `siteClass`), own rows in gawk for-in order (`awkorder`), awk `-v` escape processing (`escaper`); the default package, tag and module-prefix lists come from the wrapper (`TRIAGE_CONSOLE_DEFAULT_*`); goldens in `testdata/` recorded by `rules/golden/record/record.sh` and replayed by `golden.Runner` |
| `cmd/triage-console/` | `triage-console [--package PKG] [--tag TAG_LIST] [--module-prefix PFX_LIST] [--site] [--console-dir DIR] <console.txt\|dir>...` |
| `internal/kitticket/` | `Command` of `kit-ticket` (`toolbelt/kit-ticket.sh`): `gh issue create` with the ticket body, or the ticket in retros/tickets/<date>-<slug>.md when gh is absent (SKIP, exit 0) or fails (LOCAL exit 4 / FAIL exit 3); the kit root is `$KIT` or `KIT_TICKET_DEFAULT_ROOT`, and `KIT_TICKET_SCRIPT` names the script in the shell-style error messages; goldens in `testdata/` recorded by `rules/golden/record/record.sh` (fake gh in the case tree) and replayed by `golden.Runner` (date and pid injected) |
| `cmd/kit-ticket/` | `kit-ticket "<title>" [--from <retro-file>] [--repo <owner/repo>]` |
| `internal/stationsnapshot/` | `Command` of `station-snapshot` (`toolbelt/station-snapshot.sh`): copies config.bog and console*.txt as `cp -p` + `chmod 0644`, records history/** as size pointers and writes manifest.json (sha256, bytes, UTC stamp); `STATION_SNAPSHOT_SCRIPT` names the script in the shell-style error messages; goldens in `testdata/` recorded by `rules/golden/record/record.sh` and replayed by `golden.Runner` (the clock is injected) |
| `cmd/station-snapshot/` | `station-snapshot <station-dir> <out-dir>` |
| `internal/rcscan/` | `Main` of rc-scan, the browser-resource lint of a module's `rc/` assets (`toolbelt/rc-scan.sh` header: checks, flags, rows, exits): the awk port read line by line under gawk's `LC_ALL=C` rules (bytes, POSIX classes, leftmost-longest spans), each pattern pre-checked by a literal its match must hold; not an n4lint rule (its `<artifact-dir>` first, `--profile`/`--legacy` and `<check-id>: ` message contract), golden cases read with `rules/golden.Read` |
| `cmd/rc-scan/` | `rc-scan <artifact-dir> [--strict] [--profile hmi\|lan\|both\|unknown] [--legacy]` |
| `internal/obixlinkaudit/` | `Main` of `obix-link-audit` (`toolbelt/obix-link-audit.sh`): `DeclaredRows` (the selected wiring-map table), `Links` (each `display="Indirect\|Direct: <src> -> <tgt>"` of an `internal/xmlsrc` document; a malformed response is exit 3), `Report` (MATCH/MISMATCH/MISSING/SKIP rows) and the live mode's one read-only `net/http` GET (credentials from `OBIX_USER`/`OBIX_PASS` only) |
| `cmd/obix-link-audit/` | `obix-link-audit --map <wiring-map.md> (--xml <component.xml> \| --obix <base-url> --component <path>) [--table 1\|2] [--insecure]` |
| `internal/wiringmap/` | `Main` of `generate-wiring-map` (`toolbelt/generate-wiring-map.sh`): `Slots` (the OPERATOR and SUMMARY/READONLY `@NiagaraProperty` slots of one source, read through `javasrc.NiagaraProperties`), the bare-key lexicon display names and the three-section render whose tables `internal/obixlinkaudit` reads back; `testdata/*.txt` are golden runs recorded from the bash script by `rules/golden/record/record.sh` and replayed by `golden.Runner` in the case tree as cwd (`CHDIR=1`) |
| `cmd/generate-wiring-map/` | `generate-wiring-map <facade-src-dir> [--lexicon <module.lexicon>] [--output <file>]` |
| `internal/storedrepack/` | `Main` of `stored-repack` (`toolbelt/stored-repack.sh`): the entries of a jar streamed through `archive/zip` into a temporary file beside the output, every one STORED with its sizes in the local header, in signed-jar order (manifest, each `META-INF/<ALIAS>.SF` with its blocks, then the rest in input order), checked and linked into place; `jarlist/` (test side) lists an output jar, and the input jars are `golden.BuildJar` specs; `testdata/*.txt` are golden runs recorded from the bash script with the `testdata/setup.sh` hooks, compared through that listing |
| `cmd/stored-repack/` | `stored-repack <in.jar> <out.jar>` |
| `internal/niagaratest/` | `Main` of `run-niagara-test` (`toolbelt/run-niagara-test.sh`, kit spec C40, Go tool G33): `Verdict` reads a test run log line by line (`bufio.Scanner`, lines capped at `MaxLine`) and passes it only when every `Total tests run: N` summary has N > 0 and 0 failures and no line reports a stray failure count, `invalid test target(s)` or `failed signature validation`; the exit code and BUILD SUCCESSFUL are never read; `TestNG` (`verdict --testng <testng-results.xml> [<log>]`) streams the results file of `test.exe -output:<dir>`, the authoritative record (its summaries go to stderr; `encoding/xml` decoder, file capped at `MaxTestNGBytes`, 64 MiB), and passes it only when total > 0, failed = skipped = 0, no test-method (config included) has status FAIL and the root counts match the test-methods; exit 0 PASS / 1 FAIL / 2 usage / 3 unreadable log; `run` copies the install under `%TEMP%` (robocopy), builds `:<part>:jar :<part>:moduleTestJar` against the copy, runs `bin\test.exe -output:` and judges with `verdict --testng`, every process behind the `Runner` seam (`execRunner` kills a timed-out process group); it refuses an install under a user home or used by a running station (a service's unreadable `station.exe` attributed through its `niagarad` service, else refused), never writes the install (snapshot compare), fails when the NRE writes a real `%USERPROFILE%\Niagara4.*` registry or security dir or leaves no `etc`/`security` in the scratch user home (passed to test.exe through `WSLENV niagara_user_home/w`), builds with a WSL `--jdk8` and the caller's `--gradle-user-home`, with `--module-verification low` (test only, off by default) appends `niagara.moduleVerificationMode=low` to the copy's `defaults/system.properties` only (refused when it resolves into `--install` or out of the scratch dir; the install's file is sha256-checked before and after), with `--exclude-jar <glob>` (repeatable, a case-insensitive jar basename of `*` `?`) leaves the matching top-level `modules\*.jar` out of the copy (a glob matching no jar is FAIL, exit 2, before any copy), fails when the build changes the `--gradle-root` git worktree (`git status --porcelain=v1 -z --untracked-files=all` plus a sha256 per listed file, before the copy and after cleanup; a root that is no git worktree is exit 3 before any copy), with `--restore-gradle-root` checks out only the files that were clean before and the build modified (`--literal-pathspecs`, never a dirty or untracked file, the run stays FAIL), on a failed run (exit 1/3/4) keeps a redacted run log (`diagnostics.go`: the runner's rows, the last 200 stderr and 50 stdout lines of test.exe through a line ring cut at 4 KiB a line, the `-output:` listing, the exit status; `<UTC timestamp>-<module>.log`, 0600, under `--log-dir`, default `$XDG_STATE_HOME/run-niagara-test`; scratch, copy and home paths become `<scratch>`/`<copy>`/`<you>`, install paths stay), kills only its own Windows processes by PID and deletes only the dir it created; exit 0 PASS / 1 FAIL / 2 usage or refused / 3 environment / 4 timeout; hand-written CLI goldens replayed by `golden.Runner` (`testdata/README.md`), the orchestration tested on a fake runner, a live run opt-in (`RUN_NIAGARA_TEST_LIVE=1`) |
| `internal/retrodeltas/` | `Main` of `count-retro-deltas` (`toolbelt/count-retro-deltas.sh`, G39): per retro, the rows of its delta section vs its `**Delta count**`, `<!-- deltas: N -->` header and generated `retros/INDEX.md` Deltas cell; `OK`/`MISMATCH` rows and a TOTAL line; exit 0/1/3 |
| `cmd/count-retro-deltas/` | `count-retro-deltas <retro.md>...` |
| `internal/ledgercheck/` | `Main` of `ledger-check` (`toolbelt/ledger-check.sh`): a client's `VERSION-LEDGER.md` against the kit's `templates/VERSION-LEDGER.md` (title, sections, Versions header, Status vocabulary, dates, declared classes, defects with a class, the template's sample row); rows `FAIL|WARN  ledger-check  <file>[:<line>]  LC<n>: ...`; exit 0/1/3 |
| `cmd/ledger-check/` | `ledger-check [--template <VERSION-LEDGER template>] <VERSION-LEDGER.md>` (the wrapper sets `LEDGER_CHECK_TEMPLATE`) |
| `cmd/run-niagara-test/` | `run-niagara-test verdict [--testng <testng-results.xml>] [<log>\|-]`; `run-niagara-test run --install <dir> --gradle-root <dir> --part <part> --module <module> [--timeout <dur>] [--keep-home] [--scratch <dir>] [--jdk8 <dir>] [--gradle-user-home <dir>] [--module-verification low] [--exclude-jar <glob>]... [--restore-gradle-root] [--log-dir <dir>]` |

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
   instead (two spaces, no colon), the row of lint-structure. The runner fills `Finding.Path` only when the
   rule leaves it empty, so a per-file rule may set it to any subject: with `BarePathRow` a row such as
   `WARN  lint-status-parity  <Class>  N=…` needs no new field (the lint-status-parity port, branch r10/status-parity-go). Severity `Pass` prints `PASS` and never affects the exit code.
   `Spec.UsageExit` is the exit code of a usage error when the rule runs alone (0 means 3).
   A directory the walk cannot read prints `<lint>: cannot list every file under <root>: find: '<dir>': <reason>`
   (`<dir>` quoted as GNU find does under `LC_ALL=C`: `\'`, `\\`, `\n`-style C escapes, 3-digit octal for
   other control bytes and bytes from 0x80) and, by default, stops the rule: no rows, no later root, exit 3,
   as `scan_files_or_exit` does. Set `Spec.ContinueOnWalkError` only when the bash lint used `awk_lint_run`,
   which scans the files it did list (env line, rows, next root, exit 3), and write that reason in a comment
   beside it (reference: `rules/nullcontext`).
   Severity `Advisory` prints `ADVISORY` and, like `Pass`, never affects the exit code, `--strict` and
   `--all` included: a note that must not turn into a failure.
   Severity `Info` prints `INFO` and never affects the exit code either (lint-servlet's information rows).
   `Spec.StrictFails` prints the rule's WARN rows as FAIL under `--strict`, for a bash lint that rewrote its
   severity column (`v = strict ? "FAIL" : "WARN"`); the exit code is 1 either way. `Spec.RelPaths` prints a
   per-file rule's row path as `${file#"$root/"}` (relative to the root; a root given with a trailing `/`
   keeps the whole path, as the expansion does). `Mode` `Trailing` parses `<root> [--strict]...`: the first
   argument is the root as given, every later one must be `--strict`, and anything else prints
   `<lint>: unknown option: <arg>` (exit 3) before the root is checked (reference: `rules/servlet`).
   `Mode` `Valued` parses `[<option> <value>]... <root>` for a lint with one value option (`Spec.Option`,
   e.g. `--wiring-map`): the option and the root in any order, the last value wins, and a missing value, any
   other `-` argument or a second root prints the usage line alone. The rule reads the value by implementing
   `lint.Configurer`: run alone, it gets the parsed `Args` after the root check and before the walk and
   returns a new rule value of its kind holding what it read, which runs instead (never mutate the
   registered rule: a later run in the same process, or a rule list, would see the values); a non-nil error
   is printed verbatim on stderr and exits 3 (reference: `rules/linktargetflags`). Repeatable
   value options go in `Spec.Options` (every value, in order, in `Args.Values[<option>]`) and `--strict` is a
   flag only with `Spec.ValuedStrict` (reference: `rules/inlinesiteliteral`). Boolean flags go in `Spec.Flags`
   (e.g. `--no-default-allow`): anywhere, no value, `Args.Flags[<flag>]` true when given; `<flag>=<v>` is an
   unknown option (reference: `rules/privateapiimport`).
   `Mode` `Anywhere` takes `--strict` before or after the root; another `-` argument prints
   `<lint>: unknown option: <arg>` and a second root `<lint>: unexpected argument: <arg>` (exit 3, no usage
   line), and no root the usage line (reference: `rules/wbthreading`). `Spec.CheckFirst` prints a finding as
   `<check>  <SEV>  <path>:<line>  <msg>` (the check before the severity), the row lint-wb-threading always
   printed; the empty-root ERROR row keeps the standard shape.
   `Mode` `FileFirst` parses `[--strict] <file> <root>...` for a lint whose first operand is a file (only a
   first `--strict` is a flag; the file must be a regular file, its error `<lint>: <Spec.Option> not found:
   <file>` printed before any root is checked; every later operand is a root). `Spec.Option` is the file's
   noun here, read by the same `ParseValued`/`ValueOptions.Option` as a `Valued` option; the rule reads the
   file (`Args.Option`) in `Configure`; `n4lint --all` leaves it out and a rule list refuses it. `Spec.Joined` gives
   a `TreeRule` the files of every root in one `CheckTree` call after the last root, and a finding with
   `Line` 0 and no `Path` prints `<SEV>  <check>  <msg>`; severity `Summary` prints `SUMMARY` and never
   affects the exit code (reference: `rules/manuallabels`, one UI haystack and one total over its roots).
   An empty root is an error, never a clean pass, for every rule (the K20 guard of
   `lint-silent-protection.sh`, made the engine default after the 2026-10-22 truth audit): a root where
   the rule selects no file prints `ERROR  <check>  <root>  <msg>` on stdout and stops the rule like a walk
   error (no later root, exit 3). `Spec.EmptyError` (`lint.EmptyRow{Check, Msg}`) only shapes that row:
   `<check>` is `Check`, or the rule ID when empty; `<msg>` is `Msg`, or by default
   `no <pattern>[ or <pattern>...] sources found (wrong path or empty source tree)` from `Select.Names`.
   The one opt-out is `Spec.EmptyOK`, for a lint whose own contract makes an empty root clean (reference:
   `rules/lexiconascii`, LA5: a module with no `*.lexicon` has nothing to check): no ERROR row, exit 0.
   `Spec.OptionalInAll` applies that opt-out under `n4lint --all` only, for a rule whose files belong to one
   profile (`rules/spapoll` reads a -ux root's `*.html`/`*.js`; `--all` also runs on -rt roots without them);
   run by name or in a rule list the rule keeps the ERROR row.
   It is decided per root after the walk, so a walk error wins and prints no ERROR row. `Spec.BareUsage` prints only the usage line on an unknown flag
   (`-h` included), for a bash lint whose `-*)` case printed the usage alone; the exit stays `UsageExit`.
   `Spec.IntFlags` (with `Spec.Configure`) takes a bash lint's own numeric options (`--max-class <n>`, its
   `AL_FLAG_HOOK`) among the leading flags and builds the rule from their values; a bad value prints
   `<lint>: <flag> needs a non-negative integer` and the usage line (exit 3).
   `Spec.ShortScanError` words an unreadable file as `awk_lint_each` did (`<lint>: cannot scan <path>: <reason>`)
   instead of `awk_lint_run`'s awk error (reference: `rules/agentonshape`).
   Every engine feature lands as one `Spec` field (or severity) plus a doc and a table test; a rule opts
   in by setting it, never by a private copy.
   A lint whose report is ordered by check rather than by file implements `lint.TreeRule` instead
   (reference port: `rules/timers`, lint-timers, which also keeps gawk's matching and `for (k in a)` order;
   that order lives in `rules/awkorder`, which the per-file `rules/configsanity` uses too):
   `CheckTree(files []*javasrc.File)` gets every selected file under the root in walk order, and sets
   `Path` on each finding; a non-nil error prints the findings returned with it, then `<lint>: <error>`
   (the rule writes the whole text) and exits 3. A spec sets exactly one of `Rule` and `Tree`, and its
   value implements only that interface (`Command` panics otherwise). A `TreeRule` whose selected files hold
   nothing it checks (reference: `rules/duplicatehelper`, no `-rt`/`-ux`/`-wb` `src/` file under the root)
   returns `lint.ErrEmptyRoot`: the `Spec.EmptyError` row, the rule stops and the exit is 3:

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
   `Spec.ValuedNamed` gives Mode `Valued` lint-write-path's named errors instead of the usage line alone
   (`<lint>: unknown option: <arg>`, `<lint>: <option> requires a file argument`, `<lint>: unexpected
   argument: <arg>`, all exit 3) and lets an empty argument leave the root unset; with `Spec.Options` and
   `ValuedStrict` it parses `<root>` with `--strict` and several value options anywhere, and the rule reads
   the last value of each (`Args.Values`) through `lint.Configurer` (reference port: `rules/writepath`). A `TreeRule`
   or `RootRule` error that is a `*lint.RowError` prints its `Row` on stdout (after the findings returned
   with it) instead of `<lint>: <error>` on stderr, still exit 3, for a bash lint whose env errors were
   stdout rows. `Finding.Status` prints another first column (`STALE`, `DRIFT`) for a row that counts as
   its `Severity`.
   A rule of another kind whose roots are module roots, not source roots (reference: `rules/duplicatehelper`),
   sets `Spec.ModuleRoot`: `n4lint --all` leaves it out too.
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
   A port that is not a lint rule (a command with env, side effects, modes or symlinks) replays its cases
   through `golden.Runner` and records them with `rules/golden/record/record.sh` from the pinned bash
   (`rules/golden/record/README.md`); it keeps no golden loop or recorder of its own.
   A new rule with no bash lint behind it (reference: `rules/machinedefaultpath`) writes its golden cases by
   hand from the doctrine (usage, empty root, pruned directories, several roots) and its table tests from the
   doctrine's positive and negative shapes, RED first.
   A command that writes files records each one as an `@after <file>` section of its case, and runs
   every case with the case tree as its working directory (`CHDIR=1` in `record.env`), so relative
   `--output` paths replay too (reference: `internal/wiringmap`). `[ev: retro r10-generate-wiring-map-go Δ1]`
   A command whose output is a binary archive pins its observable contract, not its bytes: the case's
   `setup.sh` builds each input from a text spec, and the run goes through a wrapper that writes a listing
   of the output (entry names, order, method, size, content hash) that the Go replay writes too
   (reference: `internal/storedrepack`, `testdata/setup.sh`). `[ev: retro r10-stored-repack-go Δ1]`
   Budget a port's tests against the code it replaces: reuse `golden.Runner` (its `Setup` hook, one shared
   listing) before adding a test-only package, keep a unit test only for what no golden can see, and prove a
   slimming with the same mutation table before and after. `[ev: retro r10-stored-repack-go Δ3]`
4. Make `toolbelt/lint-<name>.sh` a wrapper like `lint-null-context-write.sh`: keep its header (description,
   Usage, Exit, `# Mutation:` lines) and exec `n4lint <rule-id> "$@"` via `go_tool_bin`.
5. Prove the old and new scripts print the same bytes on the bats fixtures and real trees; list every
   intended difference in the PR.
6. Fix the bash lint's truth bugs only after the byte-identical port, one commit each, RED first: a
   hand-written `testdata/truth-*.txt` case (same format) plus a bats test whose `# Mutation:` pin names it.
   A line-matching rule reads `linescan.Code(f)` (or `f.Stripped()` when the token lives in a literal),
   never the raw line cut at its first `//` (string-unaware: a `//` in "http://h" hid the rest of the line).
   An annotation harvest joins `@NiagaraProperty` lines with `javasrc.PropertyScan` over the
   `javasrc.MaskLiterals` view of the `javasrc.Strip` lines and reads the name and flags from the
   stripped text: on raw lines a `(` in a javadoc or a string literal held the scan open and dropped every
   annotation after it, and a flag word in a comment changed the table. `[ev: retro r10-generate-wiring-map-go Δ3]`

Do not copy the walker, the formatter, the argument parser or the comment stripper into a rule package;
change the shared code instead.
A tool that writes a file another tool reads (generate-wiring-map writes the map obix-link-audit reads)
tests its output through the reader's parser (`wiringmap.TestMapReadBack` calls `obixlinkaudit.DeclaredRows`),
and the reader finds a column by its header cell, never by a fixed index: a column added to the generator
shifted the audited RT slot onto the display name. `[ev: retro r10-generate-wiring-map-go Δ2]`
A STORED zip entry written from Go goes through `zip.Writer.CreateRaw` with the CRC and size of the source
header and the bytes streamed from `zip.File.Open`, which fails at EOF when they do not match: `CreateHeader`
always sets the data-descriptor flag, and `java.util.zip.ZipInputStream` rejects a STORED entry that has one.
`[ev: retro r10-stored-repack-go Δ2]`

## XML rules

R8-D12: an XML or `.bog` lint is ported only onto `internal/xmlsrc`, never onto its own regexes or a second
tokenizer. Each file is read once (UTF-8, ISO-8859-1 or US-ASCII; a zipped `.bog` unzipped in memory) and is either
a well-formed `Doc` or an `*Error` printed as `path:line:col: message`; a malformed file is a finding, never an empty
result. One exception follows Niagara: a `bajaObjectGraph` may hold U+0000 (a `&#x0;`, as
`XWriter.safe` writes it, or a literal byte) in text, attribute values, comments and CDATA, because `XParser` reads it
there (nre `javax/baja/xml/XParser.java:958-1030`); the `Doc` holds it as U+0000 and carries one `Doc.Warnings` entry at
the first, which every `.bog` consumer prints (`bog-audit` an `XML  WARN  <path>:<line>:<col>` row that
`commissioning-verify` relays as `bog-xml`, each lint rule that reads a `.bog` (lint-bformat-call, lint-write-path `--bog`,
lint-bacnet-status-flags) a WARN row through `lint.XMLWarnings`); a name holding it, and any other document, stay an `*Error`. A rule that returns that `*Error` from `Check` has it printed once, `<lint>: path:line:col: message` (exit 3),
not behind the engine's `cannot scan <path>: <rule>:` prefix. Writing XML and XPath are out of scope. `TestParityExpatSuite` (`XMLSRC_EXPAT_SUITE` = niagara-go's
`internal/pyxml/testdata/xmlerrors.json`) checks the well-formedness verdict against expat; `TestSweepRealFiles`
(`XMLSRC_SWEEP_ROOTS`) opens real client and module files. `bog-audit.sh` (`internal/bogaudit`) reads station bogs through it. The first consumer is `lint-agent-on-shape.sh` (`rules/agentonshape`: every `<on>` element of each module-include.xml, a malformed file a positioned `path:line:col` error, exit 3); the second is `lint-agent-on-include-sync.sh` (`rules/agentonincludesync`, G99: each part's module-include.xml `<type class>` line against its sources' `@NiagaraType` `@AgentOn` types, read with `javasrc.Tokens`; its `test-type-in-prod` sub-check FAILs a production `<type>` whose class is under the part's `srcTest/` or in its moduleTest-include.xml, and verify-module's `moduletest` row reuses it through `agentonincludesync.PartTestLeaks`); the third is `lint-bacnet-status-flags.sh` (`rules/bacnetstatusflags`, G36: every `*.bog` under a station directory, BACnet Present_Value proxies whose point facets lack `statusFlags=true`, one row per device); `split-package-check.sh` only looks for its build markers by name and reads no XML.
