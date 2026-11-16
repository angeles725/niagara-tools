# Golden cases of a Go port: format, replay, recording

A Go port of a bash tool proves itself by replaying runs recorded from the bash: `golden.Runner` replays
them, `record.sh` (this directory) records them. Both read the same case files.

## Case file (`testdata/<case>.txt`)

The `golden.Read` format: `-- <name> --` header lines, each followed by its body.

| Section | Meaning |
|---|---|
| `args` | one argument per line |
| `want` | `exit <n>`, then the exact stdout; `want (noeol)`: the last line has no newline |
| `stderr` | the exact stderr; `stderr (noeol)` as for want |
| `@kind` | `recorded from <SOURCE> at <COMMIT> (sha256 <SHA256>)` and a `tools:` line (bash, awk, find versions); a `truth-*` case (a declared difference from the bash, never recorded) says `truth: <why>` |
| `@env` | `KEY=VALUE` lines: the whole environment of the run. `PATH`, `HOME` and `LC_ALL` default to `/usr/bin:/bin`, `@DIR@` and `C` when not declared |
| `<path>` | a file of the tree before the run (mode 0644) |
| `<path> (mode NNN)` | the same, with that mode (`755` for an executable, `000` unreadable) |
| `<path> (dir)` | a directory (0755; with `(mode NNN)` another mode) |
| `<path> (symlink)` | a symlink to the body's one line |
| `<path> (noeol)` | last marker: the file's last line has no newline |
| `@after <entry>` | the run leaves this entry (same markers as above) |
| `@gone <path>` | the run removes this entry |

Everything else of the tree must be exactly as before the run: a missing, extra or changed entry (content,
mode, symlink target, directory) fails the case. `@DIR@` is the case directory and `@REPO@` the repository
root (when the package passes it) in every section, and stands for them again in the output.
The lint goldens (`golden.Run`, `golden.RunMain`) use the same file with more content markers (`(base64)`,
`(class N)`, `(jar)`); `record.sh` does not write those.

## Case set (`testdata/`)

- `record.env`: `SOURCE` (repository path of the bash), `COMMIT`, `SHA256` (of the extracted script), `EXTRA`
  (space-separated `<path>:<sha256>` files beside the script that it sources), `CHDIR=1` (the run's cwd is the
  case directory). `golden.Runner` checks `@kind` against it and honors `CHDIR`.
- `setup.sh` (optional, recorder only): `setup_once` (host environment, once: build a Go tool the script's
  children run) and `setup_case` (per case: `RUN`, the script path; `SKIP`, entries no tree shows, `p/` = p and
  all below; `SETUP_ENV`, environment overrides). The Go test's `Runner.Setup` mirrors what the run needs.
- The Go test pins the exact case count.

## Recording

```sh
git show <COMMIT>:<SOURCE> > /tmp/x/tool.sh          # plus each EXTRA file at its path beside it
memcap go/internal/lint/rules/golden/record/record.sh /tmp/x/tool.sh go/internal/<pkg>/testdata /tmp/x/out
diff -r go/internal/<pkg>/testdata /tmp/x/out          # only truth-* cases (never recorded) differ
```

A new case is a file with its inputs (args, @env, tree); recording fills in the rest. `record.sh` refuses
root, a script or EXTRA file whose sha256 is not the pinned one, a script that runs a Go tool (`go_tool_bin`,
`go-tool.sh`: the recording would compare the port with itself) and an out or work dir inside a git
worktree. Each run is `env -i` with only the case's environment, under `set -euo pipefail` and a cleanup
trap.

## Re-recording through the Go port (`-update`)

When the port's output changes on purpose (a new member row, a declared difference), re-record the cases that
moved instead of failing them:

```sh
memcap go test ./internal/<pkg> -update
git diff go/internal/<pkg>/testdata                  # review: every changed line must be intended
memcap go test ./internal/<pkg>                      # green, and writes nothing
```

`-update` is a flag of every test binary that imports this package (so such a package must not declare its own
flag named `update`). A case whose exit code, stdout or stderr differs gets its `-- want --` and `-- stderr --`
sections rewritten (with `(noeol)` when the output has no final newline; a stderr section is added after want
when the case had none); every other byte of the file is kept. A tree difference is still a failure. The runner
resolves the case directory to an absolute path first, so `CHDIR=1` cases are rewritten in place too. A
re-recorded case of a set with `record.env` no longer shows what the pinned bash printed: say in the set's
README which cases moved and why.
