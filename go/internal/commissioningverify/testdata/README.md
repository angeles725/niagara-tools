# commissioning-verify golden cases

Each `*.txt` file is one run of `commissioning-verify` in the `golden.Runner` format
(`go/internal/lint/rules/golden/record/README.md`). `@DIR@` is the case tree and `@REPO@` the repository root.
`golden_test.go` replays every case in its tree as cwd, with only the case's environment, through a copy of the
toolbelt whose members are symlinks to the real one (the case's `toolbelt/<member> (mode NNN)` sections are
stubs that replace a member), and checks the exact output and that the run left the tree as it was.

## Recorded cases (every name not starting with `truth-`)

Recorded from the bash tool as it stood before the port: `build-n4-module-kit/toolbelt/commissioning-verify.sh`
at commit `f86eeddf` (`record.env` pins its sha256). They cover every run of `tests/commissioning-verify.bats`
(`bats-*`), argument, env, discovery, stub-exit and station edges (`edge-*`) and values-owed parser edges
(`owed-*`). To re-record them and prove the port byte-identical:

```sh
git show f86eeddf:build-n4-module-kit/toolbelt/commissioning-verify.sh > /tmp/cv/commissioning-verify.sh
memcap go/internal/lint/rules/golden/record/record.sh /tmp/cv/commissioning-verify.sh \
  go/internal/commissioningverify/testdata /tmp/cv/out
diff -r go/internal/commissioningverify/testdata /tmp/cv/out   # only the truth-* cases and README.md differ
```

`setup.sh` runs the script as `commissioning-verify.sh` of the same toolbelt copy; the source lints it runs are
this checkout's Go wrappers.

## Truth cases (`truth-*`)

Hand-kept expectations that differ from the bash tool on purpose: each one is a declared fix (retro
go-commissioning-verify) or a behaviour the bash tool cannot show. They were recorded through the Go tool after
the fix and checked by hand.

| Case | Fix | Bash tool |
|---|---|---|
| `truth-edge-bog-malformed`, `truth-edge-bog-nocomponent` | T1: the bog-audit exit-3 SKIP row names what exit 3 means | named python3 |
| `truth-edge-bog-check15`, `truth-edge-bog-check19` | T2: CHECK15/CHECK19 WARN rows are relayed | dropped them |
| `truth-edge-stub-cs-exit127`, `truth-edge-stub-sp-exit2`, `truth-edge-stub-rp-exit1`, `truth-edge-stub-cs-noexec` (mode 644: EACCES) | T3: a non-zero lint exit with no row is a FAIL row | PASS |
| `truth-edge-stub-cs-noshebang` | a stub with no `#!` line is ENOEXEC, exit 126, FAIL row (every real wrapper starts with `#!/usr/bin/env bash`) | ran it as a shell script (exit 0, PASS) |
| `truth-owed-unreadable` | T4: an unreadable values-owed table is exit 3 | PASS beside an awk error |
| `truth-edge-bog-nul-char` | a bog holding `&#x0;` (Niagara reads it) relays bog-audit's one `XML  WARN` row as `bog-xml` | SKIP (bog-audit exit 3) |

`golden_test.go` pins the case count (88). A case holding a `(mode 000)` file fails when the tests run as root
(root reads it, so the case would prove nothing) unless `GOLDEN_ALLOW_ROOT_SKIP=1` is set, which skips it before
any of its tree is written.
