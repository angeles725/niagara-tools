# run-niagara-test golden cases

Each `*.txt` file is one CLI run of `run-niagara-test` in the `golden.Runner` format
(`go/internal/lint/rules/golden/record/README.md`); `golden_test.go` replays them. There is no bash tool to
record from, so every case is hand-written and its `@kind` says so.

Every `run.log` is synthetic: it is built from line shapes quoted from real `bin\test.exe <module> -v:2` and
gradle `niagaraTest` runs (kit retros `testing-parity-btestng` and `r10-dr5-driver-exemplar`), not copied from a
client log. Real anonymized logs are owed as a follow-up (ledger fragment `r10-run-niagara-test-go`).

The parser's line shapes and edges (CRLF, prefixes, malformed numbers, a huge line) are the table of
`verdict_test.go`; these cases pin only the CLI: its rows, usage, exits and unreadable logs.

The `testng-*` cases pin `verdict --testng`: every `testng-results.xml` is synthetic and anonymized
(`ModuleA`, `com.example`), built from the element and attribute shapes of a real station-test run of
`bin\test.exe <module> -v:2 -output:<dir>`, whose console holds no summary line. Its XML edges (total 0,
malformed or truncated XML, counts that disagree, bad root attributes, the size cap) are the table of
`testng_test.go`.
