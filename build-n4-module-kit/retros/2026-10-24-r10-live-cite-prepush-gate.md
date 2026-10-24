<!-- review-status: pending -->
<!-- module: kit -->
<!-- deltas: 1 -->
# 2026-10-24 · kit · r10-live-cite-prepush-gate

**Session**: niagara-research quality round 10 (`odd/tasks/quality-round10-ayudante1.md`), branch
`feat/r10-live-cite-prepush` from origin/main e49b7df0
**Delta count**: 1

## What happened
Live build-n4-module doctrine (the type cards, the core docs, the skill, the toolbelt index and the repository
READMEs) reached LIVE strict findings 0, but nothing kept it there: `kit-cite-resolve --strict` ran only by hand.
CI cannot run it, because the resolver exits 3 when the research, modules, clients or memory roots are
unreadable and GitHub runners hold none of them. The gate therefore went into the local `.githooks/pre-push`.

The live set is now defined once (`scripts/lib/live-doctrine-set.sh`). A push range that changes a live card or
the resolver runs the strict check on the pushed live set before every early exit, so the trivial waiver and
`pass_early` still waive only retro bookkeeping. Exit 1 refuses with the non-RESOLVED rows. Exit 3 whose message
names an absent research/modules/clients/memory root prints `pre-push: live cite check SKIPPED — roots absent
(<which>)` and continues. Any other exit 3 (an unreadable root, a missing kit root or card, a tool error row) and
any other non-zero refuse.

## Evidence
- `toolbelt/kit-cite-resolve.sh` header "Exits": 0 no finding · 1 any finding under `--strict` · 3 usage, a
  missing or unreadable root or card, a tool error row, no Go toolchain or a failed build.
- `go/internal/citeresolve/command.go:80-85`: the root loop stops at the first unreadable root and prints
  `kit-cite-resolve: cannot read <name> root <dir>: <err>`, so an absent root is told apart from every other
  exit 3 only by `<name>` and an ENOENT `<err>`.
- On origin/main e49b7df0 the real hook, run on a range touching `types/`, refused with the 10 LIVE findings that
  the pending resolver fixes remove (MISSING_FILE 8, MISSING_SECTION 2).
- `tests/pre-push-live-cite.bats` LC1-LC10, RED before the hook change for every case but LC6 (the negative).

## Proposed kit deltas (propose-never-apply)
| Δ | Delta | Target file / § | Token |
|---|---|---|---|
| 1 | **A local-only gate skips only on a named, matched absence.** When a check cannot run on CI because its inputs live only on a developer machine, it runs in the local hook. Its single skip matches the tool's exact message for an absent input and prints a named stderr line; every other non-zero exit, including the same exit code for another cause, refuses. A tool failure is never a zero result. | BUILD-LOOP.md §7 | `local-only-gate-named-skip` |

## Lessons
- An exit code shared by several causes needs the message to tell them apart before a hook may skip on it.
- A hook check that must not be waived goes before the first early exit, and a test pins it under the waiver.
