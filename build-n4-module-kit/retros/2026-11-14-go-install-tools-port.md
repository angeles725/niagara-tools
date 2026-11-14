<!-- review-status: pending -->
<!-- module: kit -->
<!-- deltas: 4 -->
# 2026-11-14 · kit · go-install-tools-port

**Session**: "all tool logic in Go" pass, wave 3: `install-skill.sh`, `check-skill-drift.sh`, `kit-resolve.sh`, `install-hooks.sh` and `install-client-hooks.sh` become launchers over Go commands.
**Delta count**: 4

## What happened
The five install and bootstrap scripts were ported with a differential of 143 rows (stdout, stderr, exit code and
the resulting tree with modes, symlinks, git config, HEAD and status): 125 identical, 18 declared. Nine of the
declared rows are places where the bash tool reported success over a result it could not have checked (an
unreadable kit served as if it were the default one, hooks installed and silenced, `/.claude` targeted with an
empty `$HOME`); the port refuses there. None of the five is sourced by anything, so none had to stay shell.

## Evidence
- The launchers of `install-skill` and `check-skill-drift` still demand `sha256sum` or `shasum` before the build
  step although Go computes the digests: `tests/install-skill.bats` IS5 and `tests/check-skill-drift.bats` SD8 run
  with a PATH that cannot reach a Go binary and pin that message. `[ev: 30bfc8db]`
- `go/internal/installtools` starts git itself (`GitStatus`) and `kit-resolve` sets `GIT_TERMINAL_PROMPT=0` in its
  own process, because `gitcmd.Run` returns no exit status and takes no environment. `[ev: 689d4d07]`
  `[ev: 2cc8d9c0]`
- `tests/c8-close.bats` to `tests/c11-close.bats` call `install-skill.sh --dry-run` against the real home
  directory of the machine that runs them. `[ev: 30bfc8db]`
- `CONTRIBUTING.md` asks for a port in two steps (a byte-identical commit, then one droppable commit per fix);
  here the nine refusals landed inside the port commits, each pinned by a named test. `[ev: 609df7a6]`

## Proposed kit deltas (propose-never-apply)
| Δ | Delta | Target file / § | Token |
|---|---|---|---|
| Δ1 | The sha256 tool requirement is dropped from the two skill launchers: IS5 and SD8 are rewritten to pin what still matters (no Go toolchain is exit 3 with its message), and `scripts/lib/sha256.sh` keeps only its real callers | `scripts/install-skill.sh` · `scripts/check-skill-drift.sh` · `tests/install-skill.bats` · `tests/check-skill-drift.bats` | `[ev: 30bfc8db]` |
| Δ2 | `gitcmd` is the only git runner: it gains the exit status and an environment argument, and `installtools.GitStatus`, the `kitresolve` fetch and `prepush/git.go` move onto it | `go/internal/gitcmd` · `installtools` · `kitresolve` · `prepush` | `[ev: 689d4d07]` |
| Δ3 | The close bats run `install-skill.sh --dry-run` with `--home` on a throwaway directory; a lint refuses a bats case that runs an install script without `--home` or an overridden `HOME` | `tests/c8-close.bats` to `tests/c11-close.bats` · `tests/shell-hygiene.bats` | `[ev: 30bfc8db]` |
| Δ4 | The port rule states its exception: a false success found by the differential (exit 0 over a result that was not checked) is turned into a refusal inside the port, declared in the changelog and pinned by a named test; any other fix stays a separate commit | `CONTRIBUTING.md` (porting a script) | `[ev: 609df7a6]` |

## Lessons
- A test that narrows PATH pins whatever the launcher does before it reaches a binary; such a contract lives in
  the launcher or the test is wrong.
- Two git runners in one module means two answers to "what if git fails". One runner, with the exit status.
- `git config -z --list` separates an unset key from an unreadable configuration; `git config --get || true`
  cannot.
- A differential pins its base by commit sha: the shared `origin/main` ref moves when a peer fetches.

---
**Status**: PENDING — INDEX row generated from the header: `| 2026-11-14-go-install-tools-port.md | kit | 2026-11-14 | pending | 4 |`
