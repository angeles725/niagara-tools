<!-- review-status: folded -->
<!-- module: kit -->
<!-- deltas: 11 -->
# 2026-11-17 · kit · kit-p1-security-guards

**Session**: open-deltas backlog rows OD004 (retro 2026-11-10-delivery-guards-and-ui-contrast Δ1) and OD005
(retro 2026-11-10-mcpn4-history-batch-tools Δ6), both P1 security gaps.
**Delta count**: 11

## What happened
The client-identifier guard ran only at pre-push, so a client name could sit in a local commit (and in its
message) until the push refused it, after which the history had to be rewritten. The guard itself lives in
niagara-go (`client-guard`, read from the pre-push stdin), so the commit-time hooks reuse it unchanged: they write
a snapshot commit of the staged tree, or of HEAD's tree with the proposed message, into a throwaway object
directory (`GIT_OBJECT_DIRECTORY`, the repository's objects as an alternate, a copy of the index for
`git write-tree`) and hand the guard one pre-push line naming it. The hooks are plain bash with no Go build, so a
work tree whose Go code does not compile can still commit; the pre-push gate stays the Go launcher it was.

The servlet write path had the global `OPERATOR_WRITE` bit (types/dashboard.md DWS1) and an X-Requested-With
heuristic, but no template for the session-bound CSRF token or for the caller's permission on the component being
written. The source shows why the misuse is easy: `BIProtected.getPermissions(cx)` is the caller's permission on
THAT object, so `getPermissionsFor(this)` in a servlet reads the servlet's permissions, and
`BUser.getPermissions(cx)` reads the caller's permission on the user object. `types/servlet-write.md` is the
template (CSRF token, target resolved under a pinned base with the caller's Context, `isMounted()`, permission on
the target, `Flags.OPERATOR`), and `lint-servlet-write-guards` flags the misuse; a bats test lints the template
with both servlet lints so they cannot drift apart.

## Evidence
- RED before GREEN: tests/pre-commit-client-guard.bats PC1, PC3-PC5, PC7-PC10 failed with no pre-commit or
  commit-msg hook (a staged or message client name was committed); all ten pass with the hooks.
  `[ev: tests/pre-commit-client-guard.bats]`
- The hooks with the real `client-guard` and a scratch denylist refused a staged match (`f:2: rule #1`) and a
  message match (`message:1: rule #1`) and let a clean commit through. `[ev: .githooks/pre-commit]`
- RED before GREEN: four of the seven golden cases (self-permission, no-target-permission, no-csrf, scope) failed
  against a stub rule; all pass with the rule. Mutations, not committed: dropping `this` from the self pattern,
  counting a self-permission line as a check, ignoring the order on the write's line, not accepting
  `requireTargetWrite(` and reading non-web code each turn a case red.
  `[ev: go/internal/lint/rules/servletwriteguard/testdata/truth-self-permission-1.txt]`
- The template linted by both servlet lints is clean (tests/lint-servlet-write-guards.bats SWG1-SWG2); replacing
  its `requireTargetWrite` call with `getPermissionsFor(this)` turns SWG1 red. `[ev: tests/lint-servlet-write-guards.bats]`
- API facts: `CsrfUtil.verifyCsrfToken(HttpServletRequest)` (web-rt vineflower `CsrfUtil.java:17-20`),
  `BUser.getPermissions(Context)` / `getPermissionsFor(BIProtected)` (`BUser.java:810-840`),
  `BComponent.getPermissions(Context)` (`BComponent.java:896-903`). `[ev: corpus B612 §612.2]` `[ev: corpus B602 §602.3]`

- Native review (lenses risk, resilience, reliability) approved the first candidate with advisory findings. Four
  were hook defects, fixed in a follow-up commit (Δ5): `git commit -v` put the staged diff below the scissors line
  into the scanned message, so scrubbing a name was refused (PC11 red before the fix); an empty array under
  `set -u` aborts bash before 4.4; `git rev-parse --path-format` needs git 2.31; and the guard's own exit status is
  now read before the SIGPIPE status of the line handed to it. `[ev: tests/pre-commit-client-guard.bats]`
- The review of that fix approved it with two more advisories, also fixed: the cut no longer fires on any
  scissors-shaped line, so a -m/-F message cannot hide lines below one (PC12 red before), and a message file that
  cannot be read refuses instead of reading as an empty message (PC13 red before). The next review asked for one
  bounded correction: the cut had matched git's English comment under the scissors line, which a localized git
  translates; it now needs a scissors line, a comment line under it and a `diff --git` line further down (PC14 red
  before; the targeted validator approved it). `[ev: tests/pre-commit-client-guard.bats]`
- Known limit at merge (#769), closed by the follow-up below: message mode could not see which cleanup git will
  apply, so a '#' line that `commit -m` or `-F` keeps was not scanned at commit time.
- Follow-up of the independent verifier's advisories on #769 (no blockers), each probe a regression test observed
  RED before its fix: a scissors-shaped line with a comment and a `diff --git` line under it in a `-m` message hid
  the lines below it (PC17, PC21 with nothing staged); a '#' line that `-m` or `--cleanup=verbatim` keeps was not
  read (PC18, PC19; PC6 now pins that a comment naming a client is refused even under `--cleanup=strip`, the price
  of not knowing the cleanup). The hook now reads comment lines and, below a cut, drops only the lines that are
  lines of a diff git appends (staged, against HEAD's parent for `--amend`, unstaged for `-vv`); PC11, PC14 and
  PC20 pin that a real `commit -v` / `--amend -v` scrub still commits. `[ev: tests/pre-commit-client-guard.bats]`
- Same follow-up, the lint: `BUser caller = op.getUser(); caller.getPermissions(op)`, `BComponent self = this;
  self.getPermissions(op)` and `super.getPermissions(op)` passed (truth-self-alias red); a check on another variable,
  a discarded `target.getPermissions(op);`, a `requireTargetWrite` declared in the file that checks nothing, and
  `target.invoke(…)` guarded or skipped a write (truth-target-receiver red); a local named `csrfToken` cleared
  no-csrf (truth-no-csrf red). Mutations, not committed: matching any receiver, counting a discarded check, trusting
  a declared helper by name, dropping `invoke`, dropping `super`, dropping the aliases and accepting the
  `csrfToken` name each turn a case red. `[ev: go/internal/lint/rules/servletwriteguard/testdata/truth-target-receiver-1.txt]`

## Proposed kit deltas (propose-never-apply)
| Δ | Delta | Target file / § | Token |
|---|---|---|---|
| Δ1 | APPLIED 2026-11-17. The client-identifier guard also runs at pre-commit (staged changes) and commit-msg (the message) | `.githooks/pre-commit` | `[ev: tests/pre-commit-client-guard.bats]` |
| Δ2 | APPLIED 2026-11-17. A servlet write template with the CSRF token and the permission on the TARGET, and a lint for `getPermissionsFor` misuse | `build-n4-module-kit/types/servlet-write.md` § `Template` | `[ev: tests/lint-servlet-write-guards.bats]` |
| Δ3 | niagara-go `client-guard` takes a context word so a commit-time refusal does not say "push refused" / "pushed commits" | niagara-go `internal/clientguard/prepush.go` | `[ev: .githooks/pre-commit]` |
| Δ4 | lint-servlet-write-guards reads the servlet's dispatch class too (a TreeRule), so a token check in a DUX1 router counts for its servlet | `go/internal/lint/rules/servletwriteguard/servletwriteguard.go` | `[ev: build-n4-module-kit/toolbelt/lint-servlet-write-guards.sh]` |
| Δ5 | APPLIED 2026-11-17. The commit-msg guard drops the `commit -v` diff section (found by structure, not by git's translated text), refuses an unreadable message file, and runs on bash before 4.4 and git before 2.31 | `.githooks/pre-commit` | `[ev: tests/pre-commit-client-guard.bats]` |
| Δ6 | PARTLY APPLIED 2026-11-17 (Δ9: a target check must name the written receiver and be used in a condition or an assignment). Open: lint-servlet-write-guards narrows `write` to component writes (a JDBC `setInt`, `List.set` or `AtomicReference.set` in web code WARNs today) | `go/internal/lint/rules/servletwriteguard/patterns.go` | `[ev: go/internal/lint/rules/servletwriteguard/testdata/truth-target-checks-1.txt]` |
| Δ7 | APPLIED 2026-11-17 another way (Δ8: the hook reads comment lines too, so nothing git may keep goes unread). Was: niagara-go `client-guard` reads a commit message file itself with git's cleanup mode (or the hook passes it), so a '#' line that `commit -m`/`-F` keeps is guarded at commit time too | `.githooks/pre-commit` | `[ev: tests/pre-commit-client-guard.bats]` |
| Δ8 | APPLIED 2026-11-17. The commit-msg guard reads comment lines too and, below a `commit -v` cut, drops only the lines of a diff git appends, so a cut-shaped line in a -m/-F message hides nothing | `.githooks/pre-commit` | `[ev: tests/pre-commit-client-guard.bats]` |
| Δ9 | APPLIED 2026-11-17. lint-servlet-write-guards: self-permission through a local bound to the user or the servlet, and `super`; a target check must name the written receiver and be used; `invoke` is a write; a declared `requireTargetWrite` must check its first parameter; no-csrf needs a `verifyCsrfToken(` call or `CsrfProtectedFilter` | `go/internal/lint/rules/servletwriteguard/checks.go` | `[ev: go/internal/lint/rules/servletwriteguard/testdata/truth-target-receiver-1.txt]` |
| Δ10 | APPLIED 2026-11-17. types/servlet-write.md fact #3 names the `slotMap.getCachedPermissions()` early return of `BComponent.getPermissions`, set only on proxies by sync | `build-n4-module-kit/types/servlet-write.md` § `Why each guard is there` | `[ev: tests/lint-servlet-write-guards.bats]` |
| Δ11 | niagara-go `client-guard` also reads binary content and file paths (a client name only in a binary file or only in a path is not refused today, at pre-push or at commit time) | niagara-go `internal/clientguard` | `[ev: .githooks/pre-commit]` |
