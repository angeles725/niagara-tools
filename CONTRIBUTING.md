# Contributing to niagara-tools

This file is the recipe book for future Cristian and Claude Code cold-starts. `CLAUDE.md` is the
agent-facing decision tree ("when to invoke `ng-deploy.sh`", "invariants"); this file is the
procedure manual ("how to bump version", "how to add a KB topic", "how to commit"). Where the two
overlap, `CLAUDE.md` carries one-liners and pointers; this file carries the full recipe.

**Scope**: solo developer + AI agents. External-contributor formalism (CoC, issue/PR templates,
governance) is deliberately out of scope until the repo has a GitHub remote AND public consumers.

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
commit a binary fixture. `tests/ng-deploy.bats` does not load the helpers.

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
6. Push deferred until `niagara-tools-github-remote` change unlocks the remote:
   `git push && git push --tags`. Until then, tags are local-only — see §8 Limitations.

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

## 8. Limitations (current solo phase)

- **No GitHub remote yet**. Tags are local-only; if the machine is lost, tag history goes with
  it. Unlock condition: `niagara-tools-github-remote` SDD change. Until then, the release
  process (§5) ends at step 5 (tag locally); step 6 (push tags) is deferred.
- **No automated check that flag-surface changes bump VERSION**. Enforced by §1 step 4 + §6
  checklist + agent discipline. Future unlock: pre-commit hook in `niagara-tools-github-remote`.
- **No CI**. The bats + shellcheck gate is human/agent-run (manual or via the SDD apply/verify
  phases). Future unlock: GitHub Actions in `niagara-tools-github-remote`.
