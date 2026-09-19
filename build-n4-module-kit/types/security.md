# Security — audit, RBAC, CSRF, secrets, injection

Four cross-cutting concerns every module must handle. Format per entry: what it is /
correct pattern / gotcha. Anti-patterns section at the end.

---

## 1 · Audit — who-changed-what

### 1.1 The null-Context double hazard

`parent.set(prop, val, null)` has **two** silent effects:

1. **Skips the audit record** — no entry appears in the audit history.
2. **Grants `BPermissions.all`** — the null context bypasses every permission check;
   any caller-supplied permission that would normally be refused is silently granted.

This is the most dangerous single-line mistake in a servlet handler.

**Correct servlet pattern:**

```java
// Servlet or @NiagaraRpc handler — attribute the write to the real user
NiagaraSuperSession session = SessionManager.getCurrentNiagaraSuperSession();
Subject subject = session.getSubject();
BUser user = BUser.getUserFromSubject(subject);
Context cx = new BasicContext(user);              // import javax.baja.sys.BasicContext

component.set(myProp, newValue, cx);              // audit fires, permissions checked
component.invoke(myAction, arg, cx);              // same rule for invoke()
```

Both `set()` and `invoke()` must receive a real `Context`. Passing `null` to either
skips the record and opens the permission gate. [ev: code BAlarmList.java `new BasicContext(user)`; corpus B507; devguide security]

### 1.2 oBIX PUT attribution gap (known limitation)

Writes arriving via oBIX PUT go through a shared machine-level login user (not the
individual browser user). The audit entry shows the shared user name, not the human
operator. This is a Niagara platform limitation, not a module bug — document it in
your module's README so integrators are not surprised. [ev: memory PANCCADIA-access-model; corpus B507 §SEC-02]

---

## 2 · RBAC — role-based access control

### 2.1 BPermissions bit table

`BPermissions` encodes six bits. The two tiers (operator / admin) each carry read,
write, and invoke:

| Constant | Bit | Meaning |
|---|---|---|
| `operatorRead` | 1 (0x01) | operator can read |
| `operatorWrite` | 2 (0x02) | operator can write |
| `operatorInvoke` | 4 (0x04) | operator can invoke actions |
| `adminRead` | 16 (0x10) | admin can read |
| `adminWrite` | 32 (0x20) | admin can write |
| `adminInvoke` | 64 (0x40) | admin can invoke actions |
| `all` | 119 (0x77) | all bits set — null-Context grants this implicitly |

Usage: `perms.has(BPermissions.operatorWrite)` — returns true when the permission set
includes the required bit. [ev: code BPxViewTag.java `BPermissions.operatorRead`; devguide security]

### 2.2 Slot tier selection

`Flags.isOperator(component, slot)` returns `true` when the slot carries the
`OPERATOR` flag; it picks the operator tier for that slot's permission check.
Slots without `OPERATOR` default to the admin tier.

**Rule of thumb:** use `ADMIN_INVOKE` on every destructive or irreversible action
(`faultReset`, `clearHistory`, device wipe). For a read-only status action that
an operator panel calls, add `Flags.OPERATOR` to reduce the required permission
to `operatorInvoke`. [ev: code Flags.java; corpus B507 §SEC-05]

---

## 3 · CSRF + secrets

### 3.1 Official CSRF filter (preferred)

`javax.baja.web.filters.CsrfProtectedFilter` is the platform-provided, correct CSRF
guard. Wire it in `web.xml` for every servlet that mutates state:

```xml
<!-- web.xml — protect POST on your custom servlet -->
<filter>
  <filter-name>csrfMyServlet</filter-name>
  <filter-class>javax.baja.web.filters.CsrfProtectedFilter</filter-class>
  <init-param>
    <param-name>httpMethod</param-name>
    <param-value>POST,PUT,DELETE</param-value>
  </init-param>
</filter>
<filter-mapping>
  <filter-name>csrfMyServlet</filter-name>
  <url-pattern>/myServlet</url-pattern>
</filter-mapping>
```

The token must travel with every mutating request. Three carrier options:

- **AJAX header (strongly preferred for POST/PUT):** `"x-niagara-csrfToken": token`
- Query string: `?csrfToken=<token>` (GET only, state-read operations)
- Form field: auto-embedded in all Niagara profiles

Retrieve the token in JS: `csrfUtil.getCsrfToken()` (bajaux) or `hx.getCsrfToken()`
(HxViews). In Java: `SessionManager.getCurrentNiagaraSuperSession().getCsrfToken()`.
[ev: devguide csrfProtection.html; code bacnetAws BackupCommand.js `x-niagara-csrfToken`]

**Legacy fallback note:** our modules currently use the `X-Requested-With` header
guard as the CSRF check. That is a weaker heuristic — valid for same-origin XHR, but
not a token-based proof. Migrate to `CsrfProtectedFilter` in new modules; keep the
`X-Requested-With` check only where a full migration is not yet planned.

### 3.2 BPassword — safe handling

`BPassword` is a guarded value. Rules:

- `toString()` returns the masked string `"--password--"` — safe to log, safe in
  error messages.
- `getValue()` returns the plaintext `char[]`. It requires a Java security privilege;
  call it inside `AccessController.doPrivileged(...)`. **Never log the result.**
- Use `SecretChars` (try-with-resources) to zero the char array after use — prevents
  the plaintext from sitting in a GC-able heap object.

```java
char[] plain = AccessController.doPrivileged(
    (PrivilegedAction<char[]>) () -> myBPassword.getValue());
try (SecretChars sc = SecretChars.of(plain)) {
    // use sc.chars() — zeroed on close
}
// Never: log.info("password=" + myBPassword.getValue())
```

[ev: devguide security; corpus B507 §SEC-06]

---

## 4 · Anti-injection — BQL and ORD

BQL has **no parameterized query API**. Constructing a query by string concatenation
with user-supplied input is a direct injection vector.

### 4.1 Safe literal embedding

```java
// SAFE: escape the value before embedding
BSimple userVal = BString.make(req.getParameter("name"));
String literal  = BqlQuery.toBqlLiteral(userVal);      // quotes and escapes
String bql      = "bql:select * from module:MyType where displayName=" + literal;

// SAFE: escape a slot name from user input
String safeName = SlotPath.escape(rawSlotName);
```

[ev: code BRelationInfoList.java `SlotPath.escape(...)`; tagdictionary-rt; corpus B507 §SEC-09]

### 4.2 BOrd from client input — allowlist prefix mandatory

`BOrd.make(clientOrdString)` executed without validation lets a caller reach any
station resource, file, or remote host reachable from the server.

```java
// Require a known-safe prefix before resolving
String raw = req.getParameter("ord");
if (raw == null || !raw.startsWith("slot:/MyModule/")) {
    resp.sendError(400, "ord not allowed");
    return;
}
BOrd ord = BOrd.make(raw);   // only after allowlist check
```

Never accept an arbitrary BOrd from client-supplied input without an allowlist prefix
check. [ev: corpus B507 §SEC-10]

---

## 5 · Licensing your own module

`BILicensed` is the interface to implement when a module requires a Niagara license
feature. Call `feature.check()` in `serviceStarted()` so the station reports a clear
fault immediately if the license is absent, rather than silently allowing unlicensed
operation until the first use:

```java
// In your BNetwork or BService subclass
@Override
public void serviceStarted() throws Exception {
    super.serviceStarted();
    Feature f = getLicenseFeature();  // implement BILicensed
    if (f != null) f.check();         // throws LicenseException if absent
}

@Override
public final Feature getLicenseFeature() {
    return Sys.getLicenseManager().getFeature("yourVendor", "yourFeature");
}
```

[ev: code BAaPhpNetwork.java `getLicenseFeature()`; code BBacnetAwsNetwork.java; corpus B507 §SEC-07]

A fuller treatment of license file format, DSA signing, and OEM trust certs belongs in
a future dedicated licensing doc.

---

## 6 · Anti-patterns (do NOT copy)

These patterns were found in real third-party modules during the Reflow security audit.
Do not reproduce them.

| Anti-pattern | Risk | Evidence |
|---|---|---|
| HTTP (plain) license phone-home including `hostId` (IP) in the payload | Exposes station identity; MITM-able | corpus B507 §SEC-08; clOnboardIO Helper.java `hostId = "ip:" + ...` |
| Destructive action reachable via HTTP GET | CSRF-exploitable via `<img>` tag; no browser CORS protection on GETs | corpus B507 §SEC-11 (Reflow FileTree) |
| `BOrd.make(clientInput)` used as an open gateway without an allowlist prefix | Full resource traversal from the browser | corpus B507 §SEC-10 |
| `FilePath("^")` root enumeration without a permission check | Lets authenticated users list all station files | corpus B507 §SEC-11 |
| `BqlQuery.make("...select...where name='" + userInput + "'")` | BQL injection | corpus B507 §SEC-09 |

**Proposed lint checks** (not yet in verify-module.sh):
- `null-context-write` — flag `set(slot, val, null)` / `invoke(action, val, null)` outside framework internals.
- `bql-string-concat` — flag `BqlQuery.make(...)` with a `+` operand.
- `arbitrary-ord-input` — flag `BOrd.make(x)` where `x` is a request parameter without a prefix guard.

---

**See also:** `types/actions.md` §3 (ADMIN_INVOKE flag), `types/actions.md` §6
(`@NiagaraRpc` CSRF surface), `types/observability.md` (audit vs log decision table).
