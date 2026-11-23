# Servlet write template — CSRF token + permission on the TARGET

A copy-ready `doPost` for any `-ux`/`-rt` servlet endpoint that writes a slot. It adds the two guards a write
needs beyond the global `OPERATOR_WRITE` bit of `types/dashboard.md` § DWS1: the session-bound **CSRF token**,
and the caller's permission **on the component being written**, never on the servlet or on the user.
`toolbelt/lint-servlet-write-guards.sh` flags code that drops either one; `tests/lint-servlet-write-guards.bats`
lints the template below, so the template and the lint cannot drift apart.
`[ev: retro mcpn4-history-batch-tools Δ6]` `[ev: retro kit-p1-security-guards Δ2]`

## Template

```java
public final class BPanelServlet extends BWebServlet {
  /** The only subtree this endpoint writes: a pinned service ORD, never a client-supplied absolute ORD. */
  private static final String BASE = "station:|slot:/Services/PanelService/";

  @Override
  public void doPost(WebOp op) throws Exception {
    // 1. CSRF: the session-bound Niagara token (x-niagara-csrfToken header, else the csrfToken parameter)
    try {
      CsrfUtil.verifyCsrfToken(op.getRequest());
    } catch (CsrfException e) {
      op.getResponse().sendError(403);
      return;
    }
    // 2. an authenticated caller: with no user in the Context every permission reads as BPermissions.all
    if (op.getUser() == null) {
      op.getResponse().sendError(403);
      return;
    }
    // 3. the TARGET, resolved under the pinned base with the caller's Context (op), never with null
    String rel = op.getRequest().getParameter("path");
    String slot = op.getRequest().getParameter("slot");
    double value;
    try {
      value = Double.parseDouble(op.getRequest().getParameter("value"));
    } catch (NumberFormatException | NullPointerException e) {
      op.getResponse().sendError(400);
      return;
    }
    if (rel == null || slot == null || rel.indexOf(':') >= 0 || rel.contains("..")) {
      op.getResponse().sendError(400);
      return;
    }
    BObject obj = BOrd.make(BASE + rel).get(Sys.getStation(), op);
    if (!(obj instanceof BComponent)) {
      op.getResponse().sendError(404);
      return;
    }
    BComponent target = (BComponent) obj;
    Property prop = target.getProperty(slot);
    // 4. the caller's permission ON THE TARGET, and the slot must be an operator slot
    if (!requireTargetWrite(target, prop, op)) {
      op.getResponse().sendError(403);
      return;
    }
    // 5. write with the caller's Context: the audit names the user (types/security.md § 1.1)
    target.set(prop, BDouble.make(value), op);
    op.getResponse().setStatus(204);
  }

  /** Write gate on the TARGET: a mounted component, OPERATOR_WRITE on it for this caller, an OPERATOR slot. */
  static boolean requireTargetWrite(BComponent target, Property prop, Context cx) {
    if (cx == null || cx.getUser() == null || prop == null || !target.isMounted()) {
      return false;
    }
    if (!target.getPermissions(cx).hasOperatorWrite()) {
      return false;
    }
    return (target.getFlags(prop) & Flags.OPERATOR) != 0;
  }
}
```

Adapt the parameter names, the base and the value type; keep the five steps and their order. In a
web-module servlet (`HttpServlet` + `WEB-INF/web.xml`, the web-module recipe of `types/dashboard.md`) map
`CsrfProtectedFilter` on its write verbs instead of step 1 (`types/security.md` § 3.1); steps 2-5 stay.

## Why each guard is there

| # | Fact | Grade | Source |
|---|---|---|---|
| 1 | `CsrfUtil.verifyCsrfToken(HttpServletRequest)` reads `x-niagara-csrfToken` (URL-decoded), else the `csrfToken` parameter, and compares it to the session token; a missing or different token throws `CsrfException` (a `SecurityException`) | `[CERT]` | web-rt (vineflower) `javax/baja/web/CsrfUtil.java:13-14,17-20,36-55`; corpus B602 §602.3, B803 §803.5 |
| 2 | The framework CSRF filter covers only the paths it is mapped on (`/rpc/*`, `/file/*`, …); a `BWebServlet` gets no CSRF check unless it calls one; `X-Requested-With` alone is a same-origin heuristic, not a session-bound token | `[CERT]` | corpus B813 §813.4, B1063 §1063.2 |
| 3 | `BIProtected.getPermissions(cx)` is `cx.getUser().getPermissionsFor(this)`: the caller's permissions on THAT object. `BComponent.getPermissions(cx)` first returns `slotMap.getCachedPermissions()` when it is set, which only sync sets, on proxy components (`SyncDecoder.java:74`, `LoadOp.java:71`), so on a station component it is `getPermissionsFor(this)` | `[CERT]` | baja `BIProtected.java:14`, `BComponent.java:896-903`; corpus B612 §612.2 `[ev: retro kit-p1-security-guards Δ10]` |
| 4 | So `getPermissionsFor(this)` / `this.getPermissions(cx)` in a servlet reads the permissions on the SERVLET, and `user.getPermissions(cx)` reads the caller's permissions on the `BUser` object (admin-read gated), neither on the written target | `[CERT]` | baja `BUser.java:810-835` |
| 5 | `getPermissionsFor(target)` returns `BPermissions.all` for an unmounted component with no category mask, and `getPermissions(cx)` returns `all` when `cx` has no user: check `isMounted()` and the user first | `[CERT]` | baja `BUser.java:837-840`, `BComponent.java:896-903`, `BISpaceNode.java:27`; `types/security.md` § 4.4 |
| 6 | The global `OPERATOR_WRITE` bit of the user (`getUser().getPermissions().has(…)`) is one station-wide capability; a category-scoped station can grant it and still deny the target | `[CERT]` | corpus B763 §763.2 (residual) |
| 7 | `Flags.OPERATOR` (256) on the slot is the operator tier; `BComplex.getFlags(Slot)` returns the int | `[CERT]` | baja `Flags.java:17`, `BComplex.java:462`; `types/dashboard.md` (generic per-ORD write endpoint) |

## Anti-patterns the lint flags

- `getPermissionsFor(this)`, `this.getPermissions(op)`, an unqualified `getPermissions(op)` in the servlet — the
  servlet's own permissions (`self-permission`, FAIL).
- `user.getPermissionsFor(user)`, `getPermissionsFor(op.getUser())`, `op.getUser().getPermissions(op)` — the
  permissions on the user object (`self-permission`, FAIL); `super.getPermissions(op)`, and the same reads through a
  local bound to the user or the servlet (`BUser caller = op.getUser(); caller.getPermissions(op)`,
  `BComponent self = this; self.getPermissions(op)`), FAIL too. `[ev: retro kit-p1-security-guards Δ9]`
- A `set(…)` / `setDouble(…)` / `invoke(…)` with no target check earlier in the same method (`no-target-permission`,
  WARN): keep the check in the writing method, or call `requireTargetWrite(…)` there. Only a component write counts: a
  `set(v)` with one argument, a numeric first argument (JDBC `setInt(1, v)`, `List.set(0, x)`), and a receiver the file
  declares only as a `PreparedStatement`, `List`, `Atomic*`, `Calendar` or `ThreadLocal` are not writes, and a file with
  only those is no "servlet writes" either. `[ev: retro kit-p1-security-guards Δ6]` The check must name the written
  receiver (`other.getPermissions(op)` does not guard `target.set(…)`) and its result must be used in a condition or
  an assignment; a `requireTargetWrite` declared in the same file counts only when it checks its first parameter.
- A `BWebServlet` that writes with no CSRF token check (`no-csrf`, WARN): only a `verifyCsrfToken(` call or a mapped
  `CsrfProtectedFilter` counts, never a variable named `csrfToken`; `lint-servlet` `csrf-xrw-only` reports the
  X-Requested-With-only case. A servlet that is a thin adapter over its dispatch class (`types/dashboard.md` DUX1)
  takes the check from it: a type the servlet names in code, declared in the servlet's own directory by a file named
  after it, that calls `verifyCsrfToken(` (a dispatch class in another directory is not seen).
  `[ev: retro kit-p1-security-guards Δ4]`
