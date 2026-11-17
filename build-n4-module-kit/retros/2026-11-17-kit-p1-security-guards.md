<!-- review-status: pending -->
<!-- module: kit -->
<!-- deltas: 4 -->
# 2026-11-17 · kit · kit-p1-security-guards

**Session**: open-deltas backlog rows OD004 (retro 2026-11-10-delivery-guards-and-ui-contrast Δ1) and OD005
(retro 2026-11-10-mcpn4-history-batch-tools Δ6), both P1 security gaps.
**Delta count**: 4

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

## Proposed kit deltas (propose-never-apply)
| Δ | Delta | Target file / § | Token |
|---|---|---|---|
| Δ1 | APPLIED 2026-11-17. The client-identifier guard also runs at pre-commit (staged changes) and commit-msg (the message) | `.githooks/pre-commit` | `[ev: tests/pre-commit-client-guard.bats]` |
| Δ2 | APPLIED 2026-11-17. A servlet write template with the CSRF token and the permission on the TARGET, and a lint for `getPermissionsFor` misuse | `build-n4-module-kit/types/servlet-write.md` § `Template` | `[ev: tests/lint-servlet-write-guards.bats]` |
| Δ3 | niagara-go `client-guard` takes a context word so a commit-time refusal does not say "push refused" / "pushed commits" | niagara-go `internal/clientguard/prepush.go` | `[ev: .githooks/pre-commit]` |
| Δ4 | lint-servlet-write-guards reads the servlet's dispatch class too (a TreeRule), so a token check in a DUX1 router counts for its servlet | `go/internal/lint/rules/servletwriteguard/servletwriteguard.go` | `[ev: build-n4-module-kit/toolbelt/lint-servlet-write-guards.sh]` |
