<!-- review-status: folded -->
<!-- module: kit -->
<!-- deltas: 2 -->
# 2026-11-13 · kit · remove-dead-bash-lint-libs

**Session**: "all tool logic in Go" pass, first unit: remove the two bash lint libs no script sources any more.
**Delta count**: 2

## What happened
`toolbelt/lib/awk-lint.sh` and `toolbelt/lib/scan-files.sh` were removed with the bats files that only exercised
them, after every lint that sourced them became an `n4lint` wrapper. Two gaps showed while landing it. The live
doctrine still told an author to reuse the deleted libs, and `kit-cite-resolve --strict` reported nothing. The
push was then refused by a doctrine pointer this change did not touch.

## Evidence
- No script sourced either lib: the only mentions outside history were comments in `lint-servlet.sh`, `rc-scan.sh`
  and the Go lint engine, and two bats files. `[ev: 4c379918]`
- `kit-cite-resolve.sh --strict` over `BUILD-LOOP.md` and `skill/SKILL.md` returned 0 findings while five
  `BUILD-LOOP.md` lines and one `SKILL.md` line named `toolbelt/lib/awk-lint.sh` or `lib/scan-files.sh` in
  backticked prose. `[ev: d977fd2c]`
- The pre-push hook refused the branch on `ORCHESTRATION.md` line 39, `[ev: ...]` pointing at
  `~/.claude/agents/sdd-research.md`, a file of the pushing machine's home directory that is no longer installed.
  `[ev: 4fbf5a73]`

## Proposed kit deltas (propose-never-apply)
| Δ | Delta | Target file / § | Token |
|---|---|---|---|
| Δ1 | APPLIED 2026-11-17 (OD045; `go/internal/citeparse/inline.go` `repoPath` → kind `repo-path`, `go/internal/citeresolve/repopath.go`; `TestCardRepoPaths`, `TestRepoPath`, `TestMainRepoPathAndOutsideByScope`). Decisions: checked on LIVE cards only, in every mode (a row always, a finding under `--strict`), history keeps old names; only a whole code span `toolbelt/`, `scripts/`, `go/` or `tests/` path with no space, glob or `..`; looked up at that exact path under the kit then the repository, never by base name; the generated `toolbelt/INDEX.md` and `go/bin/` resolve; the 8 live mentions of deleted files were unbackticked. `kit-cite-resolve --strict` also checks a backticked repo-relative path in live doctrine prose (`toolbelt/...`, `scripts/...`, `go/...`, `tests/...`) and reports one that does not exist, so deleting a file fails the gate until the doctrine that names it is updated | `toolbelt/kit-cite-resolve.sh` (Go `internal/kitcite`) · `BUILD-LOOP.md` § 5 Kit tooling doctrine | `[ev: d977fd2c]` |
| Δ2 | APPLIED 2026-11-17 (OD013; status `OUTSIDE_REPO`, `go/internal/citeresolve/repopath.go` `outsidePath`; `TestOutsideRepo`, `TestEvidenceTagFlag`). Decisions: only `[ev: …]` tags (prose and `[deferred: …]` keep resolving by existence); path, module-less `code` and zip pointers; a strict finding on live cards, HISTORICAL_DRIFT on history cards; CONTRIBUTING.md has no evidence-token section, so the rule is documented in `BUILD-LOOP.md` § Kit tooling and the wrapper header; the 3 live `~/` tokens became `[ev: live read of …]`. An `[ev: ...]` token never points outside the repository: `kit-cite-resolve` reports a home-directory or absolute path as its own finding kind (not `MISSING_FILE` on one machine and a pass on another) | `toolbelt/kit-cite-resolve.sh` (Go `internal/kitcite`) · `CONTRIBUTING.md` evidence tokens | `[ev: 4fbf5a73]` |

## Lessons
- Deleting a kit file is not done until the live doctrine that names it is reworded; history (retros, ledger,
  changelog) keeps the old name.
- A gate that depends on a file in the home directory passes or fails by machine, so it gates nothing.
- Before deleting a shared lib, list its callers by `source`/exec, not by mention: comments keep a name alive long
  after the last caller is gone.

---
**Status**: FOLDED — INDEX row generated from the header: `| 2026-11-13-remove-dead-bash-lint-libs.md | kit | 2026-11-13 | folded | 2 |`
