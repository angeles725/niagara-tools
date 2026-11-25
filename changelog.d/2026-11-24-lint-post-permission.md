### Added — `lint-post-permission.sh`: a handler's `post(` with no permission check ahead FAILs

- `build-n4-module-kit/toolbelt/lint-post-permission.sh <src-dir> [--strict]` (rule
  `go/internal/lint/rules/postpermission`) enforces `types/security.md` § 2.4: `BComponent.post(action, arg, cx)` queues
  the action and the queue runs it with a null Context (baja `BComponent.java:590-593`, `EngineManager.java:310`), so the
  caller's permission must be checked before `post(`. In a request, RPC or action handler (doGet/doPost/doPut/doDelete/
  service, a method taking a WebOp or a servlet request/response, `@NiagaraRpc`, a `do<Action>` taking a Context) a `post(`
  with no `checkInvoke`, `checkWrite`, `getPermissionsFor`, `canInvoke`, `hasOperatorInvoke`, `hasAdminInvoke` (or a
  `check(`/`has(` naming `BPermissions`) before it in the same method is `FAIL  lint-post-permission  <file>:<line>`
  (exit 1). A method with no request and no Context (a timer's `started()`, `doTick()`, a private helper) is never judged.
  `// post-permission: ok <reason>` on the line or the line above waives one `post(`; a bare `ok` waives nothing.
  `--help` prints the header. Lexical, one method at a time (limits in the header). Pinned by
  `tests/lint-post-permission.bats` PP1-PP4 and the golden cases of the rule package.
