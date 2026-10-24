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
skips the record and opens the permission gate. [ev: code BAlarmList.java `new BasicContext(user)`; corpus B1316 §1316.2; devguide security]

### 1.2 oBIX PUT attribution gap (known limitation)

Writes arriving via oBIX PUT go through a shared machine-level login user (not the
individual browser user). The audit entry shows the shared user name, not the human
operator. This is a Niagara platform limitation, not a module bug — document it in
your module's README so integrators are not surprised. [ev: memory PANCCADIA-access-model; corpus B1273 §1273.2]

### 1.3 Servlet write audit — the customer-visible record `[ev: retro servlet-write-audit Δ1]`

Platform audit (§1.1) and a customer-visible audit are different deliverables: the station
audit history needs an admin and Workbench, a shared write-server/oBIX user erases the human
identity (§1.2), and a record nobody can read does not answer "who changed this".

**Contract — every servlet write handler (`doPost`/`doPut`/`doDelete`):**

1. **Record** user (the human identity, carried in the payload when a shared service account
   performs the station write), target ord/slot, old value, new value, source
   (`HMI` / `viewer` / `API`) and timestamp — for every accepted write, before answering success.
2. **Persist** the record in a BOUNDED store (ring of the last N entries, or a bounded history)
   that survives a station restart; never only in memory, never unbounded.
3. **Make it readable** to authorized users through a read endpoint; the dashboard's
   "recent changes" view reads that store, never the browser's local state.
4. **Echo what was stored:** the write response carries the value ACTUALLY stored (after
   coercion/clamping) — the same value the audit record holds as "new".

Still pass the real `Context` to `set()`/`invoke()` (§1.1) so the platform audit fires too.
Client side (per-field confirmation, recent-changes view):
[`dashboard.md` § Critical-write step-up auth](dashboard.md) — not restated here.

Enforced: `toolbelt/lint-write-path.sh` WARNs a src file that declares a servlet write handler
and contains no audit-recording call (an identifier containing `audit` invoked, getters and
comments excluded); advisory for legacy modules, `--strict` promotes it to exit 1.
`[ev: retro servlet-write-audit Δ2]`

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

**Rule of thumb:** leave `Flags.OPERATOR` off every destructive or irreversible action, so it needs `adminInvoke`
(`faultReset`, `clearHistory`, device wipe). For a read-only status action that
an operator panel calls, add `Flags.OPERATOR` to reduce the required permission
to `operatorInvoke`. [ev: code docSource Flags.java:191,451; BUser.java:1680-1686]

### 2.2a Slot flags are not a security boundary `[ev: retro n4-course-slots-components-contracts Δ2]`

`set(prop, value, cx)` with a user checks only operator vs admin write (`BUser.checkWrite`, vineflower baja 4.14.0.162
`javax/baja/user/BUser.java:902-912`); it does not look at READONLY. READONLY counts in `BComponent.canWrite(OrdTarget)`
(vineflower baja 4.14.0.162 `javax/baja/sys/BComponent.java:925-939`), which a surface that asks the `OrdTarget` honours; whether every remote surface
asks is not verified. CONFIRM_REQUIRED is a UI hint. A user with admin write can also change a frozen slot's flags:
the `setFlags` path calls the no-op hook `checkSetFlags` (`BComponent.java:619`) and has no frozen-READONLY check
(corpus B1231 §1231.9). So: protect a slot with the operator/admin tier and categories (§ 2.2), gate sensitive values
in the setter path with the Context's permissions, and override `checkSetFlags` (throw a
`LocalizableRuntimeException`) when a flag must not be cleared.

### 2.3 Fail-closed authorization default for WB controls and PX widgets [ev: retro honeywell-wb-rt-wb-deltas Δ3]

A control whose authorization mixin or PIN slot is absent must be **DISABLED and HIDDEN** by default — never open-by-default. The rule applies to both Workbench manager rows and PX widgets:

1. Treat an absent authorization mixin as no access. In Galileo a pin of `-1` means any level passes once the user carries the mixin, so `-1` is not the closed state (corpus B1079).
2. On `loadValue()` or render, check the pin against the current session's permissions.
3. If absent or insufficient: call `widget.setEnabled(false)` **and** `widget.setVisible(false)`.
4. Never fall through with `setEnabled(true)` when the authorization check returns an inconclusive or missing result.

A missing mixin ≠ "unprotected"; it equals "no access". See `types/wb-widgets.md §Integer visibilityPin/actionPin` for the integer-pin slot pattern and `types/wb-widgets.md §Good -wb artifact doctrine` rule 11. `[ev: corpus B1079]`

### 2.4 `post()` drops the caller's RBAC context — always check permission before post() `[ev: retro module-hardening-failure-modes-deltas Δ4]`

`BComponent.invoke(action, arg, context)` enforces permission via `checkInvoke` on the caller's `context` (operator = bit 4 `operatorInvoke`; admin = bit 64 `adminInvoke`; a null context skips the check and is **system-privileged**).

**`post()` is DIFFERENT:** `post(action, arg)` enqueues the dispatch asynchronously. The enqueued invocation runs with **NO RBAC enforcement** — the caller's `Context` is dropped entirely. The dispatched action executes as system-privileged regardless of who posted it.

**Security rule:** check the caller's permission yourself **before** calling `post()` for any action that requires `OPERATOR` or `ADMIN` authority. Pattern:

```java
// Before posting a privileged action
BUser user = cx.getUser();  // cx = the caller's Context
if (user == null) throw new PermissionException("no user");
user.checkInvoke(myComponent, myAction);  // throws PermissionException if denied
post(myAction, arg, cx);  // safe to post after the check
```

Never rely on the framework to enforce RBAC across the `post()` boundary. This complements §1.1 (null-Context write hazard): that rule covers `set()`/`invoke()` with explicit null; this rule covers `post()` which implicitly drops context on every call.

**Worker threads keep the caller's Context.** When an action hands work to its own thread or executor, capture
the action's `Context` before the hand-off and pass it to every `BOrd.get/resolve`, BQL query and `toString(cx)`
inside the worker: with `null`, BQL and file reads are no longer filtered by the invoking user (§ 4.4) and value
text falls back to the default locale. `[ev: retro review2-signing-build-route Δ5]`

### 2.5 oBIX exposure — licence, writes, watches, hidden ≠ protected `[ev: retro devguide415-r2-drivers-protocols Δ9]`

The PANCCADIA read/write users talk oBIX (`types/dashboard.md` § Facade contract). Rules for the station side
(4.15.6.20 guide; encoder read in obixDriver-rt 4.14.0.162 vineflower; other versions not verified):
- **Licence:** the server needs the `obixDriver` feature with `export=true` and the `web` feature
  (`Obix/LicenseRequirement-OBIXDriver-9F25675C.txt:9`). Server disabled → HTTP 410 to every request; enabled but
  unlicensed (fault) → HTTP 500 (`Obix/obixDriver-ObixNetwork.txt:19`).
- **External writes to a control point** go through an `ObixExport` descriptor at a priority level the module
  never drives; a level that is already linked faults the descriptor (`Obix/obixDriver-ObixExport.txt:19-20`).
  `set` (fallback, persists) and `override` (timed) are different writes: pick one on purpose (corpus B607 §607.2).
- **Watches** are leased: a client polls the watch well inside the lease and re-creates it after a lease expiry;
  the client driver's requested lease is the larger of 2 × interval and interval + safety factor
  (`Obix/obixDriver-ObixClient.txt:30`). `[ev: corpus B1343]`
- **HIDDEN is not protection.** The encoder drops a HIDDEN slot from child listings only (`ObixEncoder.java:588-589`);
  a direct request for the slot's path checks `canRead()` alone (`ObixEncoder.java:234-236`). The boundary for the
  read user is categories and permissions, never the flag. `[ev: corpus B1347 §1347.1]`
  `[ev: retro devguide415-r3-web-session-headers Δ6]`
- Bind each oBIX user to the authentication scheme its client actually speaks (corpus B605).

### 2.6 Least privilege for a module's users `[ev: retro devguide415-r2-security-tags-analytics Δ2]`

The README names the category and role each module user needs (read user, write user, kiosk user) instead of
`admin`. Facts that shape that role (4.15.6.20 `StationSecurity/` guide; corpus B1326 rows 4-7):
- **Ancestors are automatic:** a user granted a component by category gets operator read-only on its ancestors so
  the tree stays navigable; the station recomputes this periodically, and CategoryService `Update` forces it
  (`role_AncestorPermissions.txt:7-9`). Do not add categories to parent folders just for navigation.
- **Histories need the operator tier:** history views open only with the operator permission level set and
  operator read (`r`) granted (`role_HistoryPermissions.txt:7`). Test a module's history ext with an operator user.
- **`UserService` slots are per own account:** with the Operator flag on the slot, `r` reads and `rw` writes the
  user's own Email, Password, Cell Phone and Facets; all other users stay hidden. A self-service password change
  needs `rw` on `UserService.Password` (`role_UserServiceComponentPermission.txt:26-39`).

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
    <param-value>POST,PUT</param-value>
  </init-param>
</filter>
<filter-mapping>
  <filter-name>csrfMyServlet</filter-name>
  <url-pattern>/myServlet</url-pattern>
</filter-mapping>
```

`httpMethod` is split on `,` and matched exactly against `getMethod()` (javap web-rt 4.15.6.20
`CsrfProtectedFilter.init`/`doFilter`), so write it without spaces and list every mutating method the servlet
accepts. The shipped 4.15.6.20 modules use `POST`, `POST,PUT`, `GET,POST` or `GET`; none lists `DELETE`
(6 of the 14 jars with a `WEB-INF/web.xml` map the filter, e.g. web-rt `WEB-INF/web.xml:35` `POST,PUT`,
bacnetAws-rt `:27` `GET,POST`). Add `DELETE` only if your servlet accepts it.

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
- `getValue()` returns the plaintext `String`; `getSecretChars()` returns a `SecretChars`. It requires a Java security privilege;
  call it inside `AccessController.doPrivileged(...)`. **Never log the result.**
- Use `SecretChars` (try-with-resources) to zero the char array after use — prevents
  the plaintext from sitting in a GC-able heap object.
- A secret a component must keep is a `BPassword` **slot**: a private Java field is not saved to `config.bog`
  and is lost on restart. `[ev: corpus B1416 §1416.4 b]` `[ev: retro course-recordings Δ7]`

```java
try (SecretChars sc = AccessController.doPrivileged(
         (PrivilegedAction<SecretChars>) myBPassword::getSecretChars)) {
    // use the chars; zeroed on close
}
// Never: log.info("password=" + myBPassword.getValue())
```

[ev: devguide security; code docSource BPassword.java:440-450, :486-492]

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

**Security messages carry no stack trace:** `log(Level, msg, ex)` appends the exception's trace; for an
authentication, authorization or crypto failure log the message only (`log(Level.WARNING, msg)`). `[ev: corpus B1418 §1418.2 d]` `[ev: retro course-recordings Δ7]`

### 3.4 Unattended panels — session lifetime and client-certificate SSO

A wall panel's session ends by idle auto logoff, by the 4.15 absolute logoff, or by a reboot; the recipe and the
kiosk-user checklist live in `types/dashboard.md` § Session lifetime. For a panel that must log back in on its
own after a reboot, prefer client-certificate SSO over a stored password: a `ClientCertAuthScheme` (one ships by
default) with Auto Attempt Single Sign-On = true and Ignore Auto SSO if User Cookie Present = false
(4.15.6.20 guide `Touch/SettingUpAuth-1ABCB56A.txt:12,21`). `[ev: retro devguide415-r2-web-ui-hmi Δ7]`
`[ev: retro devguide415-r3-web-session-headers Δ2]`

---

## 4 · Anti-injection — BQL and ORD

BQL has **no parameterized query API**. Constructing a query by string concatenation
with user-supplied input is a direct injection vector. Where tags can express the query, NEQL is the
injection-safe route: it "supports parameterized queries" (devguide `neql.txt:16`), reading `{facetName}`
values from the resolve Context — `BOrd.make("neql:childValue50 = {integerValue50}").resolve(root, facets)` with
`facets = BFacets.make("integerValue50", 50)` (`neql.txt:262-285`; recipe in `types/logic.md` § Ship a tag dictionary). `[ev: corpus B1308 §1308.2 C3]`
`[ev: retro devguide415-nav-subscriptions-tags-ords Δ9]`

### 4.1 Safe literal embedding

```java
// SAFE: escape the value before embedding
BSimple userVal = BString.make(req.getParameter("name"));
String literal  = BqlQuery.toBqlLiteral(userVal);      // quotes and escapes
String bql      = "bql:select * from module:MyType where displayName=" + literal;

// SAFE: escape a slot name from user input
String safeName = SlotPath.escape(rawSlotName);
```

[ev: code BRelationInfoList.java `SlotPath.escape(...)`; tagdictionary-rt; code BqlQuery.java:87 `toBqlLiteral`]

### 4.1a BQL literal rules `[ev: corpus B1237 §1237.6]`

Paths are `bql-rt/…` (vineflower) unless named.

- **The whole query text is slot-unescaped before it is tokenized** (`javax/baja/bql/BqlQuery.java:22-25`): a `$xx`
  sequence in an embedded value is decoded first. To match a literal `$20` write `'%$2420%'`. `toBqlLiteral`
  slot-escapes the encoded value (`BqlQuery.java:87-97`), so § 4.1 round-trips; a raw string concatenation does not.
- **Enum literals** take the frozen-enum dot form `<module>:<Enum>.<tag>` (`alarm:SourceState.offnormal`); the
  parser accepts only a `BFrozenEnum` there, else `invalid.enum.tag` (`com/tridium/bql/compiler/ExprParser.java:398-414`).
  A dynamic enum cannot use the dot form.
- **Alarm scans:** query `FROM openAlarms` (or `ackPendingAlarms`) when only those rows matter — the extent names
  are case-sensitive (`alarm-rt/…/BAlarmDatabase.java:228-231`); a BQLTIME window bounds the rest, and that it
  narrows the record scan is `[INFER]`.
- **Custom functions** are called `<module>:<Type>.<fn>(args)` (`ExprParser.java:390,398-403`; devguide `bql.txt:165`);
  user-defined aggregates are "still considered experimental" (devguide `bql.txt:232`).

[ev: retro n4-course-drivers-bql-ords-files Δ2]

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
check. [INFER]

### 4.3 Lexicon text in HTML — `getHtmlSafeText`

A lexicon value rendered raw into HTML is an XSS vector.
Servlet and HTML output uses `lex.getHtmlSafeText(key)` (or `getHtmlSafeText(key, args…)`), which returns
`Encode.forHtml(getText(key))`; the same method exists on `LexiconText` and `LexiconModule`. It encodes for HTML
element content only: JavaScript, URL and attribute contexts still need their own encoder. A missing key comes back
encoded too. `[ev: corpus B1418 §1418.1]` `[ev: retro course-recordings Δ6]`

### 4.4 A permission check is only as good as the Context behind it `[ev: retro devguide415-r2-runtime-contracts-formats Δ2]`

- `BComponent.getPermissions(cx)` returns `BPermissions.all` when `cx` is null or carries no user (vineflower baja
  4.14.0.162 `BComponent.java:896-903`; devguide source 4.15.6.20 `BComponent.java:1971-1982`).
- `OrdTarget.canRead/canWrite/canInvoke` ask the target's `BIProtected` with the resolve Context and return
  **true when there is no security target** (vineflower baja 4.14.0.162 `OrdTarget.java:281-294`; devguide source
  4.15.6.20 `OrdTarget.java:476-503`). So `ord.resolve(base, null)` followed by `canRead()` is a gate that never
  closes. Resolve with the caller's Context: a servlet with its `WebOp`, an action with its own `cx`, a worker
  with the Context captured before the hand-off (§ 2.4). Recipe: `types/logic-authoring.md` § ORD resolution.
  `[ev: corpus B1316 §1316.2]`
- **A custom `BOrdScheme`** (registered with `ordScheme="id"`, `types/module-wiring.md`) returns its own
  `OrdQuery` from `parse(body)` (default: a `BasicQuery`), implements `resolve(OrdTarget base, OrdQuery query)`
  (docSource 4.14.0.162 `javax/baja/naming/BOrdScheme.java:105-119`), throws `InvalidOrdBaseException` for a base it
  cannot use (shipped precedent `BSlotScheme.java:194`, same copy) and `UnresolvedException` for a miss, and checks
  `canRead()` on every hop it returns, so the scheme cannot be a side door around permissions.
  `[ev: retro devguide415-nav-subscriptions-tags-ords Δ8]` `[ev: corpus B1308 §1308.1]`

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

[ev: code BAaPhpNetwork.java `getLicenseFeature()`; code BBacnetAwsNetwork.java `getAwsLicenseFeature()` :127]

### 5.1 · Ongoing-work gate — `serviceStarted()` only guards STARTUP `[ev: retro secure-authoring-isoperational-gate Δ1]`

The framework's own license enforcement is **advisory, not blocking**: `BAbstractService`'s
`checkLicense()` sets a fatal fault and logs SEVERE on an unlicensed/expired feature, but the
engine caller (`ServiceManager.startService`) runs the subclass's `serviceStarted()`
unconditionally right after — there is no fault guard between them, and the whole call is
wrapped in a `catch(Throwable)` that logs and continues station boot. So relying on the
framework to withhold start on its own is wrong: the `feature.check()` throw in
`serviceStarted()` above only aborts **startup**. A service that does nothing further is
protected. A service whose later callbacks (`changed()`, timers, servlet write handlers) keep
acting is NOT — they still fire while the station displays a fault.

A licensed service must therefore also gate its ONGOING work on `isOperational()`:

```java
@Override
public void changed(Property p, Context cx) {
    super.changed(p, cx);
    if (!isOperational()) return;   // !isFatalFault() && !isDisabled() && !isFault()
    // ... licensed work here ...
}
```

This is exactly what the shipped Tridium services do by convention — `BAlarmService`,
`BSearchService`, `BHierarchyService`, `BTagDictionaryService`, `BBatchJobService`, `BBoxService`,
and `BCloudConnector` all gate ongoing work on `isOperational()` / `isFatalFault()` — but nothing
in the framework forces it, so a module that skips the gate runs unlicensed under a fault icon.

[ev: corpus B1143 §1143.2-3 — BAbstractService.java:332-393,519-523]
[ev: corpus B1145 — ServiceManager.java:291-322]
[ev: corpus B1146 — BAlarmService.java:289,397,448,578 et al.]

Lint: `toolbelt/lint-license-isoperational-gate.sh` (advisory WARN, run by `report-module.sh`) flags a class
whose `getLicenseFeature()` returns a feature (not the `return null;` default) and whose `changed()`, action
(`do<Name>`, where `Clock.schedule` timers land) or servlet write handler (`doPost`/`doPut`/`doDelete`/`doPatch`) acts
with no `isOperational()`/`isFault()`/`isFatalFault()` call in its own body. A gate placed in a helper the callback
calls is not followed, so such a row is cleared by moving the gate into the callback. No overlap with
`lint-status-parity` (config/status slot ratio) or `lint-silent-protection` (an unsurfaced protection trip): neither
reads the license feature or the operational gate. [ev: retro secure-authoring-isoperational-gate Δ1]

A fuller treatment of license file format, DSA signing, and OEM trust certs belongs in
a future dedicated licensing doc.

---

## 6 · Anti-patterns (do NOT copy)

These patterns were found in real third-party modules during the Reflow security audit.
Do not reproduce them.

| Anti-pattern | Risk | Evidence |
|---|---|---|
| HTTP (plain) license phone-home including `hostId` (IP) in the payload | Exposes station identity; MITM-able | [INFER] — no source found |
| Destructive action reachable via HTTP GET | CSRF-exploitable via `<img>` tag; no browser CORS protection on GETs | corpus B65 AP-10 (Reflow backups via GET) |
| `BOrd.make(clientInput)` used as an open gateway without an allowlist prefix | Full resource traversal from the browser | [INFER] |
| `FilePath("^")` root enumeration without a permission check | Lets authenticated users list all station files | corpus B57 (AP-33, filesystem listing without RBAC) |
| `BqlQuery.make("...select...where name='" + userInput + "'")` | BQL injection | corpus B65 AP-21 |

**Proposed lint checks** (not yet in verify-module.sh):
- `null-context-write` — flag `set(slot, val, null)` / `invoke(action, val, null)` outside framework internals.
- `bql-string-concat` — flag `BqlQuery.make(...)` with a `+` operand.
- `arbitrary-ord-input` — flag `BOrd.make(x)` where `x` is a request parameter without a prefix guard.

---

## 7 · Servlet response headers — the station already sets them

**Every module servlet inherits the station's headers.** Jetty gives each `BWebServlet` its own context and
`configureNiagaraWebApp()` adds `TridiumSecurityFilter` on `/*` of it (vineflower `BJettyWebServer.java:1178-1180,1462-1468`);
the filter applies the WebService's Http Header Providers before the servlet runs (vineflower `TridiumSecurityFilter.java:22-27`).
Out of the box that is `X-Content-Type-Options: nosniff` (vineflower `BXContentTypeOptionsHeaderProvider.java:38`), the CSP
provider (`default-src 'self' workbench`, `script-src 'self' workbench 'unsafe-inline' 'unsafe-eval'`,
vineflower `BCspHeaderProvider.java:66,130`) and `X-Frame-Options` (the guide documents `Sameorigin` as the default, Hx needs
it, `User/web-HttpHeaderProviders.txt:20`; the provider class default is `deny`, vineflower `BXFrameOptionsHeaderProvider.java:34`,
so read the station's value at commissioning). `[ev: corpus B1349 C1]` `[ev: corpus B1328 row 9]` `[ev: retro r8-t9i-headers Δ1]`
`[ev: retro devguide415-r3-web-session-headers Δ3]` `[ev: retro devguide415-r2-web-ui-hmi Δ5]`

- **Set a header in servlet code only to deviate on purpose.** A `resp.setHeader()` for a name the filter already set
  REPLACES the station value for that response, so a hand-written CSP silently drops the operator's policy.
  `[ev: retro module-hardening-reqexec-closed-deltas Δ1]`
- **A hand-rolled header helper is not a gap by itself.** An earlier audit flagged DashboardPan's and Apillm's
  `setApiHeaders()` for missing `nosniff`/`X-Frame-Options`; with the filter in place those headers arrive anyway.
  Check the live response, not the source. `[ev: retro our-dashboard-audit-deltas Δ1]`
  `[ev: retro apillm-headless-servlet-rt-4.14-deltas Δ13]`
- **Embedding a station page elsewhere:** never set `X-Frame-Options` to `Any`; the operator adds a CSP
  `frame-ancestors` directive on the station's CSP provider (`User/web-HttpHeaderProviders.txt:20`). This is a
  commissioning setting, not module code. `[ev: corpus B1349 §1349.1]` `[ev: retro devguide415-r3-web-session-headers Δ5]`
- **Hx views need the defaults:** `Sameorigin` and the default CSP sources; tightening them breaks Hx views.

**Lint** (`toolbelt/lint-servlet.sh`, check `api-response-headers`, INFO): a `doGet`/`doPost`/`service` handler that
calls `getWriter()`/`getOutputStream()` in a class with no `X-Content-Type-Options` / `setApiHeaders()` /
`applyHeaders()` reference gets an INFO row ("station header providers apply"); `--strict` does not escalate it, and
`report-module.sh` counts an INFO-only servlet as scanned (PASS). `[ev: retro r8-t9i-headers Δ2]`

**Commissioning header probe** (`toolbelt/commissioning-verify.sh` `servlet-response-headers` MANUAL step): after
deploying a `BWebServlet` module, confirm the live endpoint carries the station's headers:

```bash
curl -sI -H 'X-Requested-With: XMLHttpRequest' -u admin:pass \
     'http://<station>/<module>/api/equipment' \
  | grep -iE 'x-content-type-options|x-frame-options|content-security-policy'
```

A missing header means the station's provider is disabled or the servlet overrides it: fix the provider (operator)
or remove the override (module), not both. `[ev: retro our-dashboard-audit-deltas Δ5]`

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

---

## 9 · Credential digest strength — prefer SHA-256+ over MD5 [ev: retro wb-vendor-ux-wave3-vendor-drivers-deltas Δ13]

When a Niagara module stores a credential digest (e.g. a password hash, a PIN hash, or a device challenge-response seed) inside its own component slots or configuration files, use **SHA-256 or stronger**. MD5 is broken for collision resistance and must not be used for credential storage.

**Observed pattern to avoid (honAdvWirelessCfg, corpus B1104):** the WB Platform-tab credential storage computes an MD5 digest of the provisioning password and stores the hex digest in the BLE config it pushes to the controller:
```java
// WRONG — MD5 is not suitable for credential storage
MessageDigest md = MessageDigest.getInstance("MD5");
byte[] digest = md.digest(credential.getBytes(StandardCharsets.UTF_8));
// ... store hex(digest) in a slot
```

**Correct pattern — SHA-256 with a random salt:**
```java
// CORRECT — SHA-256 with a random 16-byte salt
SecureRandom rng = new SecureRandom();
byte[] salt = new byte[16];
rng.nextBytes(salt);
MessageDigest sha = MessageDigest.getInstance("SHA-256");
sha.update(salt);
byte[] digest = sha.digest(credential.getBytes(StandardCharsets.UTF_8));
// store Base64(salt) + ":" + Base64(digest) in the slot
```

If the protocol requires a specific digest algorithm (e.g. a device that mandates MD5 for its own challenge), isolate that to the comm layer and do NOT reuse MD5 for any in-station storage. Document the protocol constraint as a known limitation.

**Lint check implemented** (`toolbelt/lint-no-md5-credential-digest.sh`):
- `no-md5-credential-digest` — flags `MessageDigest.getInstance("MD5")` in a module that also contains `BPassword`, `BCredentials`, or slot names matching `*password*` / `*credential*` / `*pin*`.

**See also:** `types/security.md §3.2` (`BPassword` safe handling), `types/issues-and-gotchas.md §H1` (precision-consistency lint). [ev: corpus B1104]

### 9.1 FIPS-mode constraints `[ev: retro devguide415-r2-security-tags-analytics Δ1]`

A station licensed for FIPS 140-2 mode puts the FIPS providers first and removes most non-FIPS algorithms, so a
module must run unchanged in both modes (4.15.6.20 `Fips/` guide; corpus B1326 rows 1-3):
- **Never name a provider.** Call `Cipher.getInstance("AES/GCM/NoPadding")`, not `getInstance(alg, "BCFIPS")`:
  a named provider works in one mode only (`Fips/DevelopersNotes-51FBF6BE.txt:14-20`). The same holds for
  `MessageDigest`, `Signature`, `Mac` and `KeyStore`. [INFER: the guide shows `Cipher` only]
- **Legacy algorithms are decrypt-only.** Blowfish, MD5 and MD5withRSA remain only to read old data (decrypt
  Blowfish, never encrypt); FIPS mode removes most other non-FIPS algorithms from the JCA provider list, so a `getInstance` for one fails `[INFER]` (`Fips/Non-FIPS-compliantAlgorithms-520109A5.txt:7-23`).
  New data uses AES and SHA-256 or stronger (§ 9).
- **Secrets are 14 characters or more:** user passwords, certificate passwords and the system passphrase
  (`Fips/FIPS1402-51D87C8C.txt:13`). Size the module's machine users (oBIX, station links) for that and plan their
  change before the FIPS licence is installed (`Fips/UpdatingUserPasswordsFIPS_N4-1E65C4B9.txt:7`).
- No kit lint checks this yet: owed G80 `jca-named-provider` (`ledger.d/2026-10-23-r8-u8-sec-tags-analytics.md`).
  `[deferred: retro devguide415-r2-security-tags-analytics Δ1]`

---

---

## 10 · Outbound vs inbound security axes — do not conflate (PD-23) `[ev: retro wb-vendor-ux-rt-wb-pattern-deltas Δ23]`

Two orthogonal security axes govern a module that makes HTTP calls AND accepts inbound auth — they are separate configuration surfaces and must NEVER be conflated:

| Axis | Mechanism | What it controls | Module author action |
|------|-----------|-----------------|----------------------|
| **Outbound HTTP** | `httpClient-rt BHttpClientService.enableNonDriverClients` | Whether NON-DRIVER modules may use `BHttpClient` for outbound HTTP requests | Set `enableNonDriverClients=true` in the station's `httpClient-rt` service if a non-driver module needs outbound HTTP; raw `HttpURLConnection` / OkHttp bypass this gate entirely (see `types/issues-and-gotchas.md §H2`) |
| **Inbound auth** | `BAuthenticationScheme` SPI | How the station verifies incoming credentials (user/password, certificate, SAML, API key) | Implement `BAuthenticationScheme` to add a custom inbound auth mechanism; see `types/security.md §5` (licensing) for the service-anchor |

**Key rule:** the `enableNonDriverClients` gate is OUTBOUND-only. It has zero effect on who can connect TO the station. A module that sets `enableNonDriverClients=true` is opting into outbound HTTP, not opening inbound auth. Documenting both axes is mandatory in any module that touches either, so integrators do not mistake an outbound-gate change for an inbound auth weakening. `[ev: corpus B1075]`

**See also:** `types/cloud-connector.md §2` (transport dual model + HTTP transport), `types/issues-and-gotchas.md §H2` (`enableNonDriverClients` gotcha).

### 10.1 TLS uses the platform stores `[ev: retro devguide415-r2-security-tags-analytics Δ4]`

Certificates and trust live in the platform: the User Key Store, the System and User Trust Stores, the Allowed
Hosts exemptions and the `CertManagerService` (4.15.6.20 `StationSecurity/SSLAllowedHostsTab.txt:7-19`; corpus
B1326 row 9). A module never ships or loads its own keystore or truststore file and never installs a trust-all
`TrustManager`; the commissioning step is importing the peer's CA into the station's trust store. A
station-to-station connection with a bad certificate fails silently, with no approval prompt; the cause is the
connection's last failure cause (`SSLAllowedHostsTab.txt:17`), so the module reports that field, not a generic
"offline". No kit lint checks this yet: owed G81 `embedded-keystore`
(`ledger.d/2026-10-23-r8-u8-sec-tags-analytics.md`). `[deferred: retro devguide415-r2-security-tags-analytics Δ4]`

**See also:** `types/actions.md` §3 (action flags; OPERATOR vs admin invoke), `types/actions.md` §6
(`@NiagaraRpc` CSRF surface), `types/observability.md` (audit vs log decision table).
