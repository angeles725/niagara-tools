# commissioning-verify golden cases

Each `*.txt` file is one run of `commissioning-verify`, in the `rules/golden.Read` format (`-- args --`,
`-- want --` = `exit <n>` and the exact stdout, `-- stderr --`, then one section per file of the case tree).
`@DIR@` is the case tree and `@REPO@` the repository root. `golden_test.go` `materialize` also reads
`<path> (symlink)` (a symlink to the section's line), `<path> (dir)` (an empty directory), `<path> (mode 000)`
(an unreadable file), `toolbelt/<member>` (the toolbelt is copied, every entry symlinked, and that member
replaced by an executable stub) and `toolbelt/<member> (mode 644)` (the same stub, not executable).

## Recorded cases (every name not starting with `truth-`)

Recorded from the bash tool as it stood before the port: `build-n4-module-kit/toolbelt/commissioning-verify.sh`
at commit `f86eeddf` (origin/main when the port branched). They cover every run of
`tests/commissioning-verify.bats` (`bats-*`), argument, env, discovery, stub-exit and station edges (`edge-*`) and
values-owed parser edges (`owed-*`). To re-record them and prove the port byte-identical:

```sh
git show f86eeddf:build-n4-module-kit/toolbelt/commissioning-verify.sh > /tmp/cv-bash.sh
cd go && CV_RECORD=/tmp/cv-bash.sh go test -count=1 -run TestRecord ./internal/commissioningverify
git diff --exit-code internal/commissioningverify/testdata   # no change = the recordings still hold
go test -count=1 -run TestGolden ./internal/commissioningverify
```

`TestRecord` runs the script as `commissioning-verify.sh` of a toolbelt copy (the case's stubs included), with the
case tree as cwd and only `PATH` and `HOME` in the environment, and rewrites each case's want and stderr sections.
It never touches a `truth-*` case.

## Truth cases (`truth-*`)

Hand-kept expectations that differ from the bash tool on purpose: each one is a declared fix (retro
go-commissioning-verify) or a behaviour the bash tool cannot show. They were recorded through the Go tool after
the fix and checked by hand.

| Case | Fix | Bash tool |
|---|---|---|
| `truth-edge-bog-malformed`, `truth-edge-bog-nocomponent` | T1: the bog-audit exit-3 SKIP row names what exit 3 means | named python3 |
| `truth-edge-bog-check15`, `truth-edge-bog-check19` | T2: CHECK15/CHECK19 WARN rows are relayed | dropped them |
| `truth-edge-stub-cs-exit127`, `truth-edge-stub-sp-exit2`, `truth-edge-stub-rp-exit1`, `truth-edge-stub-cs-noexec` | T3: a non-zero lint exit with no row is a FAIL row | PASS |
| `truth-owed-unreadable` | T4: an unreadable values-owed table is exit 3 | PASS beside an awk error |

`golden_test.go` pins both counts exactly (`recordedCases`, `truthCases`).
