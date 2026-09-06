# Contributing to niagara-tools

This file is the recipe book for future Cristian and Claude Code cold-starts. `CLAUDE.md` is the
agent-facing decision tree ("when to invoke `ng-deploy.sh`", "invariants"); this file is the
procedure manual ("how to bump version", "how to add a KB topic", "how to commit"). Where the two
overlap, `CLAUDE.md` carries one-liners and pointers; this file carries the full recipe.

**Scope**: solo developer + AI agents. External-contributor formalism (CoC, issue/PR templates,
governance) is deliberately out of scope while the project is solo. The GitHub remote now exists
(`origin` = github.com/angeles725/niagara-tools) and work lands via pull request (§8), but there
are no public consumers yet, so that formalism stays deferred.

---

## 1. Quick path (4 steps)

1. Make your change (edit code, docs, KB topic, etc.).
2. Run the gate: `bats tests/*.bats && shellcheck scripts/*.sh build-n4-module-kit/toolbelt/*.sh tests/*.bats tests/helpers/*.bash`. Both must exit 0.
3. Commit using Conventional Commits (see §6 Pre-commit checklist).
4. If the change touches `scripts/ng-deploy.sh`'s flag/exit/env surface OR you've accumulated
   2+ MINOR features, run the Release process (§5).

---

## 2. TDD requirement for `scripts/*.sh`

Strict TDD is active for this repo (per `sdd-init/niagara-tools` artifact). Any change to a
script in `scripts/` follows red → green → refactor:

1. **Red**: write the failing bats test FIRST in `tests/<script-name>.bats`. Run
   `bats tests/<script-name>.bats` and confirm it fails for the right reason.
2. **Green**: make the minimum change to the script that turns the test green. Re-run bats;
   all tests (new + pre-existing) must pass.
3. **Refactor**: clean up the script if needed; bats must stay green after every edit.
4. **Lint gate**: `shellcheck scripts/<script>.sh tests/<script-name>.bats` exits 0.

The PATH-injected fakebin pattern (see `tests/ng-deploy.bats setup()`) is the standard for
stubbing external commands (`gradlew`, `unzip`, `tar`, etc.). Do not call real commands that
touch a station, a build, or the filesystem outside `$TMPDIR_T`.

The same rules apply to the kit toolbelt (`build-n4-module-kit/toolbelt/*.sh`): its suites are
`tests/verify-module.bats`, `tests/build-sh.bats`, `tests/mirror-niagara-home.bats`,
`tests/stored-repack.bats` and `tests/kit-links.bats`. Jar/class/`niagara_home` fixtures are
generated at test time by `tests/helpers/n4-fixtures.bash` (`load helpers/n4-fixtures`) — never
commit a binary TEST fixture (generate via `tests/helpers/n4-fixtures.bash`); a scaffolder's shipped
template under `build-n4-module-kit/fixtures/` may carry the gradle wrapper jar when ≤100 KB
[ev: retro campaign7-scaffold]. `tests/ng-deploy.bats` does not load the helpers.

**Pointing the suite at a different client tree (C11 T2):** The ONE blessed client read root is
defined in `tests/lib/client-root.bash`, which exports `CLIENT_READ_ROOT`, `C9_CLIENT_ROOT`,
`C9_CLIENT_REPO`, and `C8_CLIENT_REPO` (all default to the frozen worktree
`Leon-Guanjuato-worktrees/main-ff1b659`). Every bats that needs a client tree does
`load lib/client-root` at file scope — never hardcode the path in the bats body. To run against a
different tree, set the relevant variable in the environment before invoking bats:
`C9_CLIENT_ROOT=/path/to/tree bats tests/` — the `:=` default is skipped when the variable is
already set. [ev: retro campaign11-client-root; design.md D3a/D3b]

- **In a gawk toolbelt script, call `delete arr` in `BEGIN` before `length(arr)`:** checking array length without initializing triggers a gawk warning on some platforms; `delete arr` at the top of `BEGIN` ensures a clean start. [ev: retro schema-risk]
- **Diff heredoc-embedded reference tables against an oracle file as a named mutation:** embed the reference CSV/table as a heredoc, store the same bytes in `tests/fixtures/<name>.csv`, and assert byte equality as a named mutation — the oracle file is the contract. [ev: retro schema-risk]
- **Place `shellcheck disable` directives BEFORE the entire compound command, not inside its body:** a disable inside a `while ... done` body triggers SC1123 ("ShellCheck directives are only valid in front of complete compound commands"); the correct form is one line immediately before `while ... do`. [ev: retro logic-split]

### 2.1 Test runner setup

The gate needs `bats` (bats-core) and `shellcheck` on `PATH`. Install once:

```bash
# Debian / Ubuntu (WSL default)
sudo apt-get install -y bats shellcheck

# linuxbrew (no sudo; the form used on the dev WSL, bats-core 1.14)
brew install bats-core shellcheck
```

Check: `bats --version` prints `Bats 1.x` and `bats tests/ng-deploy.bats` reports 26 cases.

---

## 3. KB topic authoring rules

Knowledge-base topics live in `docs/knowledge-base/<topic-name>.md`. Rules:

- **One topic per file**. If a topic grows past ~200 lines, split it.
- **Bidirectional link with `docs/GOTCHAS.md`**: every new topic file is referenced from
  `docs/GOTCHAS.md` in the appropriate anti-pattern row AND the topic file links back to
  `docs/GOTCHAS.md` in its header.
- **Cite engram observation IDs**: when a topic captures a discovery from a prior project
  (e.g. chihuahua), include the observation ID in parentheses (`#1788`, `#1567`, etc.) and
  the source project name. Future agents can `mem_get_observation(id: ...)` for full context.
- **No code dumps**: KB topics explain WHY and WHEN, with minimum reproducible snippets. Full
  implementations live in the relevant repo; the KB links to them.
- **Cross-project search hints**: if the topic relates to a chihuahua/other-project finding,
  add a `mem_search` hint to `CLAUDE.md` §7 so cold-start agents can find it.

---

## 4. Versioning policy

SemVer 2.0.0, currently in `v0.x` "unstable, expect change" phase. Pinning recommendation for
consumers (e.g. chihuahua): pin to a specific tag (`v0.2.0`), not a range, until `v1.0.0`.

CalVer is explicitly rejected: the `ng-deploy.sh` flag/exit-code/env-var API surface IS a
SemVer surface; losing MAJOR/MINOR/PATCH semantics would silently break consumer pinning.

Bump rules for `scripts/ng-deploy.sh` (the public API surface):

| Change | Bump |
|--------|------|
| Flag removed or renamed; flag semantics changed | MAJOR (or MINOR in 0.x with `### Changed — BREAKING` in CHANGELOG) |
| Exit code changed for an existing code path | MAJOR (or MINOR in 0.x with warning) |
| New required env var | MAJOR (or MINOR in 0.x with warning) |
| New flag (defaults off / opt-in) | MINOR |
| New optional env var | MINOR |
| New exit code (no existing code path changed) | MINOR |
| New KB topic file under `docs/knowledge-base/` | MINOR |
| Bug fix without API change | PATCH |
| Doc-only edit, comment, refactor with no behavioral change | PATCH |

While in `v0.x`: MAJOR is replaced by MINOR with a `### Changed — BREAKING` subsection in the
CHANGELOG entry, mirroring chihuahua's convention. Graduate to `v1.0.0` when the API surface
stabilizes (target: when chihuahua adopts `ng-deploy.sh` as a vendored dependency).

---

## 5. Release process

When a release is warranted (per §1 step 4):

1. Edit `VERSION`. Bump per §4 rules (e.g. `0.2.0` → `0.3.0`).
2. Edit `CHANGELOG.md`. Add a new `## [vX.Y.Z] - YYYY-MM-DD` section ABOVE the existing
   newest section. Populate `### Added` / `### Changed` / `### Fixed` / `### Removed` /
   `### Deprecated` / `### Security` subsections that apply (omit the others). Include the
   `### References` subsection with SDD slug + engram observation IDs (mirrors chihuahua style).
3. Run the gate: `bats tests/*.bats && shellcheck scripts/*.sh build-n4-module-kit/toolbelt/*.sh tests/*.bats tests/helpers/*.bash`. Both must exit 0.
4. Commit the version bump + CHANGELOG entry + any code that motivated the bump together.
   Conventional commit subject: `chore(release): vX.Y.Z` if release-only; otherwise fold into
   the `feat:` / `fix:` commit that motivated the release.
5. Tag the commit: `git tag vX.Y.Z`. Tag name uses `v` prefix; the `VERSION` file holds raw
   semver (no `v`).
6. Push to `origin`: `git push && git push --tags` (tags after the release commit is on `main`).
   Changes reach `main` via pull request, not direct pushes — see §8 for the branch → PR → ff-only flow.

`SCRIPT_VERSION` does NOT need to be edited manually. The script reads `VERSION` at startup via
`cat "${SCRIPT_DIR}/../VERSION"` (CWD-agnostic, `BASH_SOURCE[0]`-relative). Editing `VERSION`
is enough. The bats anti-drift test guards the resolution logic against regression.

---

## 6. Pre-commit checklist

Before `git commit`:

- [ ] `bats tests/*.bats` exits 0 (all tests green).
- [ ] `shellcheck scripts/*.sh build-n4-module-kit/toolbelt/*.sh tests/*.bats tests/helpers/*.bash` exits 0 (no warnings on changed lines).
- [ ] Commit message is Conventional Commits (`feat:` / `fix:` / `chore:` / `docs:` /
      `refactor:` / etc.).
- [ ] No `Co-Authored-By` trailers. No AI attribution of any kind.
- [ ] If `scripts/ng-deploy.sh` flag/exit/env surface changed: `VERSION` bumped,
      `CHANGELOG.md` entry added.
- [ ] If new KB topic added: `docs/GOTCHAS.md` updated, bidirectional links present.
- [ ] No accidentally committed `.env.local` files (`.gitignore` covers `.env.*` with explicit
      `!.env.local.example` exception — verify `git status` is clean of personal env files).

### 6.1 Enabling the retro-enforcement gate (opt-in, per-clone, reversible)

The `build-n4-module` kit ships a pre-push gate (`.githooks/pre-push`) that blocks a push touching
build-relevant kit files unless it also records continuity (a `BUILD-STATE.md` update + a pending retro +
its INDEX row), carries a `Retro: none (trivial: <reason>)` trailer, or is a promotion with a structural
anchor (an `INDEX.md` **or** `BUILD-STATE.md` diff). It ships **inert** — nothing runs until you enable it:

```sh
scripts/install-hooks.sh              # activate: git config core.hooksPath .githooks
scripts/install-hooks.sh --uninstall  # deactivate: restore the default hooks
```

It is `--local` (this clone only) and idempotent. It REFUSES to overwrite a pre-existing custom
`core.hooksPath` (your own hooks are never silently clobbered) — pass `--force` to override. See
`build-n4-module-kit/BUILD-LOOP.md` §7 for the close-gate contract the hook enforces.

---

## 7. Content boundary (CLAUDE.md vs CONTRIBUTING.md vs README.md)

| Topic | README.md | CLAUDE.md | CONTRIBUTING.md |
|-------|-----------|-----------|-----------------|
| What the repo is + quick-start | Primary | — | — |
| Repo layout | Primary | — | — |
| Versioning pointer (1-line link to CHANGELOG) | Primary | — | — |
| Deploy decision rules (when to use ng-deploy.sh) | — | Primary | — |
| Agent invariants (backup → build → copy → verify) | — | Primary | — |
| When to bump version (rule) | — | One-liner + pointer | — |
| How to bump version (recipe) | — | — | Primary (§5) |
| TDD requirement for `scripts/*.sh` | — | One-liner + pointer | Primary (§2) |
| KB topic authoring rules | — | — | Primary (§3) |
| Bats + shellcheck install commands | — | Primary (§5 install) | Pointer back |
| Bats + shellcheck run commands | — | Primary (§5 run) | Pointer back |
| Conventional commits + no-AI-attribution rule | — | Pointer to `~/.claude/CLAUDE.md` | Primary (§6) |
| Pre-commit checklist | — | — | Primary (§6) |
| Engram `topic_key` conventions | — | Primary (§6) | — |
| Cross-project search hints | — | Primary (§7) | — |
| Code of conduct, external-contributor docs | — | — | Deferred (solo scope) |

**Duplication rule**: never duplicate policy. Only duplicate one-liners as pointers.

---

## 8. Workflow & limitations (current solo phase)

- **GitHub remote + PR workflow (active)**. `origin` = github.com/angeles725/niagara-tools.
  Changes land as pull requests, not direct commits to `main`: branch off `main` → open a PR
  (`gh pr create --base main`) → merge **ff-only / rebase** (no squash, no merge commits — keep a
  linear history). Tags (§5 step 6) are pushed with `git push --tags` after the release commit is on
  `main`.
- **No automated check that flag-surface changes bump VERSION**. Enforced by §1 step 4 + §6
  checklist + agent discipline. Future: a pre-commit hook.
- **CI active** (`ci.yml`, since Campaign 5): GitHub Actions on `origin` runs shellcheck + bats + sweep on every push and PR — server-side, un-bypassable complement to the opt-in client hook. **Pin every tool the CI gate depends on** (shellcheck version, Java version, junit sha256); an unpinned tool drifts newer on the runner and turns green→red with no code change. [ev: retro ci-server-side-enforcement] (K10/A10)

---

## 9. SDD ledger discipline

- **One active attempt per change** regardless of work-unit label: the ledger is serial — a new acquire blocks while a prior attempt for the same change is open. [ev: retro campaign7-retro-fold]
- **Evidence goal = one short line:** `--evidence-goal` must be a single terse sentence parseable at a glance.
- **Inventory sha from the error, not cached:** the untracked inventory sha changes whenever files appear; always read it from the error output, never from a prior run.
- **Doc-store commits early or budget sized to content:** a large openspec commit blows a code-sized `--max-changed-lines` budget; commit docs early or pass a budget that matches the PR's real content. [ev: retro close-process-meta-lessons]
- **Two independent reads for every research fold:** author fidelity read + a mechanical gate with verbatim spot-checks of each correction — one read cannot catch what two can. [ev: retro research-fold] [ev: retro close-process-meta-lessons] [ev: retro campaign7-retro-fold]
