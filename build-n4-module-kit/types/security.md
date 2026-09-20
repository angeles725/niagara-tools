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

### 2.3 Fail-closed authorization default for WB controls and PX widgets [ev: retro honeywell-wb-rt-wb-deltas Δ3]

A control whose authorization mixin or PIN slot is absent must be **DISABLED and HIDDEN** by default — never open-by-default. The rule applies to both Workbench manager rows and PX widgets:

1. Default the authorization slot/PIN to `-1` (no assignment = no access).
2. On `loadValue()` or render, check the pin against the current session's permissions.
3. If absent or insufficient: call `widget.setEnabled(false)` **and** `widget.setVisible(false)`.
4. Never fall through with `setEnabled(true)` when the authorization check returns an inconclusive or missing result.

A missing mixin ≠ "unprotected"; it equals "no access". See `types/wb-widgets.md §Integer visibilityPin/actionPin` for the integer-pin slot pattern and `types/wb-widgets.md §Good -wb artifact doctrine` rule 11. `[ev: corpus B1079]`

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

### 3.3 Credential header redaction in structured logging [ev: retro honeywell-wb-rt-wb-deltas Δ7]

Redact `Authorization`, `X-Api-Key`, and similar credential headers **at every log level, including `FINEST`**. A cloud/HTTP connector that logs the full `HttpURLConnection` request properties at FINEST leaks Bearer tokens to the station log history:

```java
// WRONG — logs the Authorization header value at FINEST
LOG.finest("request headers: " + conn.getRequestProperties());

// CORRECT — redact before logging
private static final Set<String> REDACT_HEADERS =
    Collections.unmodifiableSet(new HashSet<>(Arrays.asList(
        "Authorization", "X-Api-Key", "X-Auth-Token")));

Map<String, List<String>> safe = new LinkedHashMap<>();
for (Map.Entry<String, List<String>> e : conn.getRequestProperties().entrySet()) {
    safe.put(e.getKey(),
             REDACT_HEADERS.contains(e.getKey()) ? Collections.singletonList("***") : e.getValue());
}
LOG.finest("request headers: " + safe);
```

The production log level is typically INFO, but a support engineer enabling FINEST for diagnostics must not accidentally capture credentials. Add this guard to any code that iterates HTTP headers or logs connection state. `[ev: corpus B1082]`

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

## 7 · Servlet response-header checklist

**Platform-level filter note** (`TridiumSecurityFilter`): the station's `WebService/httpHeaderProviders`
are applied globally to EVERY servlet context — including every custom `BWebServlet` at `/api/` — by
`TridiumSecurityFilter`, installed on `/*` of every context by `configureNiagaraWebApp()`. When the
station has header providers configured and enabled, a custom JSON API servlet receives the configured
headers automatically with no per-servlet wiring. `setApiHeaders()` and the `lint-servlet.sh` check are
therefore a **defense-in-depth fallback layer** for deployments where the station-level providers are
absent or disabled; a servlet-level `resp.setHeader()` call OVERRIDES the filter value for the same
header name. `[ev: retro module-hardening-reqexec-closed-deltas Δ1]`

Every `BWebServlet` API response path **must** call a `setApiHeaders()` helper (or
equivalent) that sets at minimum the following security headers:

| Header | Required value | Scope |
|---|---|---|
| `X-Content-Type-Options` | `nosniff` | **all** API responses |
| `X-Frame-Options` | `SAMEORIGIN` | **all** API responses |
| `Content-Security-Policy` | `default-src 'self'; script-src 'self' 'unsafe-inline'` | HTML responses only |

`DashboardPan`'s `setApiHeaders()` (v2026-08-31, `BDashboardServlet.java:540–543`) omits
both `X-Content-Type-Options` and `X-Frame-Options`, and ships no `Content-Security-Policy`
anywhere in the module — gaps ODA2-G1/G2. The two-line fix is:

```java
// Inside setApiHeaders() — add after any existing header calls
resp.setHeader("X-Content-Type-Options", "nosniff");
resp.setHeader("X-Frame-Options", "SAMEORIGIN");
// HTML endpoint only:
resp.setHeader("Content-Security-Policy", "default-src 'self'; script-src 'self' 'unsafe-inline'");
```

**Proposed lint check** (not yet in `toolbelt/lint-servlet.sh`):
- `api-response-headers` — flag a `doGet`/`doPost` handler that writes a response but
  does NOT call a method name containing `setHeader` with `X-Content-Type-Options` in
  the same method or a helper it delegates to.

`[ev: retro our-dashboard-audit-deltas Δ1]`

**Post-deploy header probe** (see also `toolbelt/commissioning-verify.sh`
`servlet-response-headers` MANUAL step):
After deploying a `BWebServlet`-based module, verify the live endpoint returns the required
headers:

```bash
curl -sI -H 'X-Requested-With: XMLHttpRequest' -u admin:pass \
     'http://<station>/<module>/api/equipment' \
  | grep -iE 'x-content-type-options|x-frame-options'
```

Both headers must appear in the response. A missing header is an ODA2-G2-style gap
detectable in minutes — the static code fix (`setApiHeaders()`) is equally low-effort.
`[ev: retro our-dashboard-audit-deltas Δ5]`

---

---

## 8 · CORS recipe

To allow cross-origin access from a browser SPA served from a different origin than the station, there
are two paths — a static global route and a dynamic per-request route.

### Static ACAO via `BGenericHttpHeaderProvider` (recommended for fixed-origin deployments)

Add a `BGenericHttpHeaderProvider` under `WebService/httpHeaderProviders`:

| Slot | Value |
|---|---|
| `headerName` | `Access-Control-Allow-Origin` |
| `headerValue` | the allowed origin (e.g. `https://myapp.example.com`); `*` for open |
| `appendHeader` | `false` |

The provider is managed entirely in the station BOG; no code change is required.
`TridiumSecurityFilter` emits the header on every response via `applyHeaders()`. `[ev: corpus B1134]`

**Gotcha — `BProfileFilterFactory` is closed to third parties:** the platform mechanism for registering
a custom servlet `Filter` enforces a hard-coded 2-entry allowlist (`BProfileFilterFactory.java:17-18`).
A custom CORS `Filter` submitted via `BProfileFilterFactory` silently fails to register. The supported
CORS paths are: (a) the station BOG `BGenericHttpHeaderProvider` (static ACAO); (b) the servlet body.
`[ev: corpus B1134]`

### Dynamic per-request Origin reflection (for multi-origin or preflight handling)

When a static `Access-Control-Allow-Origin` is insufficient — multiple allowed origins, preflight
`OPTIONS` handling — implement the CORS logic in the servlet body:

```java
String origin = req.getHeader("Origin");
if (isAllowedOrigin(origin)) {            // your explicit allowlist check
    resp.setHeader("Access-Control-Allow-Origin", origin);
    resp.setHeader("Access-Control-Allow-Credentials", "true");
}
if ("OPTIONS".equalsIgnoreCase(req.getMethod())) {
    resp.setHeader("Access-Control-Allow-Methods", "GET, POST, PUT, OPTIONS");
    resp.setHeader("Access-Control-Allow-Headers", "Content-Type, x-niagara-csrfToken");
    resp.setStatus(HttpServletResponse.SC_NO_CONTENT);
    return;
}
// … normal handler follows …
```

**Rule:** never reflect the `Origin` header without an explicit allowlist check. A servlet-level
`resp.setHeader(...)` OVERRIDES the station-level filter value for the same header name.

`[ev: retro module-hardening-reqexec-closed-deltas Δ2]`

---

**See also:** `types/actions.md` §3 (ADMIN_INVOKE flag), `types/actions.md` §6
(`@NiagaraRpc` CSRF surface), `types/observability.md` (audit vs log decision table).
