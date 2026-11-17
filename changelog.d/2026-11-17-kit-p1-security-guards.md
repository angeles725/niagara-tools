### Added — client-identifier guard at commit time; servlet write template and lint-servlet-write-guards

- `.githooks/pre-commit` and `.githooks/commit-msg`: niagara-go's `client-guard` now also runs on the staged
  changes and on the proposed commit message, so a client name is refused before a local commit exists (the
  pre-push guard is unchanged). The hooks hand the guard a snapshot commit written to a throwaway object
  directory; the repository's objects and index are never written. `scripts/install-hooks.sh` activates them
  with the other hooks and names them.
- `build-n4-module-kit/types/servlet-write.md`: a copy-ready servlet `doPost` that verifies the Niagara CSRF
  token (`CsrfUtil.verifyCsrfToken`) and the caller's permission on the written TARGET
  (`target.getPermissions(cx)`), never on the servlet or the user.
- `build-n4-module-kit/toolbelt/lint-servlet-write-guards.sh` (Go `go/internal/lint/rules/servletwriteguard`,
  a report-module member): FAIL `self-permission` for `getPermissionsFor(this)`, a permission read off the user
  and similar; WARN `no-target-permission` for a slot write with no target check before it, and `no-csrf` for a
  writing `BWebServlet` with no CSRF check.
