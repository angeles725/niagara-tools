<!-- review-status: pending -->
# 2026-09-18 · kit · permissions-security-model-deltas

**Session**: corpus-mining for build-n4-module (operator: generate all possible kit deltas — permissions/security model).
**Delta count**: 8

## What happened

Mined B18 (module signing + module-permissions.xml + CSRF + header auth), B755 (BPermissions exact
bit values), B776 (action protection declarative recipe), B777 (security-module skeleton + permissions
inlining), B1004 (JACE-8000 QNX `moduleVerificationMode=medium`).

Cross-checked every finding against `types/dashboard.md`, `types/structure.md`, `types/logic-authoring.md`,
`types/wb-widgets.md`, `build-verify.md`, `corpus-index.md`, and retros
`dashboard-servlet-write-surface-and-reader-authority`, `ux-wb-writesurface-rbac-deltas`.

**Prior coverage confirmed (no duplicate proposed):**

- Servlet RBAC: `BPermissions.has(OPERATOR_WRITE)` (bit, not role name), DWS1/DWS2 five gates, CSRF
  `X-Requested-With` + `x-niagara-csrfToken` → `dashboard.md §RBAC + §DWS1/DWS2` ✓ [retro `ux-wb-writesurface-rbac-deltas.md` confirms these are folded]
- `@NiagaraAction(flags=Flags.OPERATOR)` operator-invoke (256), admin DEFAULT, `BComponent.canInvoke`,
  `doPrivileged` correct use vs AP-27 → `logic-authoring.md §action protection` ✓ [B776 folded]
- `requiredPermissions` on `@AgentOn` = view-visibility only, never security →
  `wb-widgets.md §rule 6` ✓
- Categories: author NOTHING, runtime assignment via `BCategoryService` →
  `logic-authoring.md:25` ✓
- Dev cert vs project CA, JACE signing check → `build-verify.md:110` ✓ (partial — JACE=medium gap is NEW)

Eight net-new deltas remain, all from the COMPONENT/JVM-sandbox permission layer (module-permissions.xml,
permission groups, verificationMode, signing bypasses) and the missing BPermissions complete bit table —
a distinct and orthogonal concern from the already-folded servlet RBAC surface.

## Evidence

- B18 §18.4.1 — `module-permissions.xml` dev-facing XML format: `<permissions>
  <niagara-permission-groups type="all|workbench|station"> / <req-permission> / <name>NETWORK_COMMUNICATION
  </name> / <purposeKey> / <parameters>`. Developer writes SEMANTIC group names, not Java class names.
  [ev: corpus B18 §18.4.1]

- B18 §18.4.2-18.4.3 + B777 §777.4 — Runtime transformation: the Gradle signing plugin (class
  `NiagaraPermissionGroupFactory`) expands high-level group names into concrete `<java-permission
  class="com.tridium.nre.security.NiagaraSocketPermission" ...>` / `<java-permission class=
  "java.lang.RuntimePermission" ...>` entries, then INLINES the expanded block into the built JAR's
  `META-INF/module.xml` `<permissions>` element. There is NO `module-permissions.xml` in the deployed
  artifact (find-zero across all corpus JARs per B777 §777.4). To inspect deployed grants:
  `unzip -p <mod>-rt.jar META-INF/module.xml | grep permissions`.
  [ev: corpus B18 §18.4.2-18.4.3; corpus B777 §777.4]

- B18 §18.4.4 + §18.3.1 — Three permission groups always require Honeywell code signing regardless of
  `verificationMode`: `ACCESS_CLASS` (exposes `sun.misc.*`/`sun.reflect.*`), `REFLECTION`
  (`ReflectPermission "suppressAccessChecks"`), `MBEAN_PERMISSION` (JMX operations). Verified via
  `ModuleClassLoader.verifyJarEntrySignature()` L88-96: `requiresSignature = module
  .getRequestedNiagaraPermissions().stream().anyMatch(NiagaraPermissionGroup::requiresSignature)`. A dev-
  signed or unsigned module that declares any of these will REFUSE to load even on a WIN supervisor set to
  LOW. [ev: corpus B18 §18.4.4, §18.3.1]

- B18 §18.4.7 — `type="station|workbench|all"` profile scoping: permissions are granted ONLY in the
  declared profile. `type="workbench"` grants apply only to the Workbench JVM; `type="station"` only to
  the NRE/daemon. Declaring `AWTPermission` under `type="all"` is harmless on the station (no AWT there)
  but signals an oversight; the real hazard is the inverse — granting station permissions only under
  `type="workbench"` and wondering why the daemon rejects them at runtime.
  [ev: corpus B18 §18.4.7]

- B755 §755.3 — BPermissions full 6-bit table (source: `BPermissions.java:26-31`): `OPERATOR_READ=1`,
  `OPERATOR_WRITE=2`, `OPERATOR_INVOKE=4`, (bit 8 unused), `ADMIN_READ=16`, `ADMIN_WRITE=32`,
  `ADMIN_INVOKE=64`, (bit 128 unused). The kit mentions `OPERATOR_WRITE` by name in `dashboard.md:28`
  and `OPERATOR_INVOKE` in `logic-authoring.md:72` but never presents the complete table. A servlet
  that invokes actions (not writes) gates on `OPERATOR_INVOKE=4`; a config-read gate uses `ADMIN_READ=16`.
  [ev: corpus B755 §755.3]

- B1004 §1004.2-1004.3 — JACE-8000 QNX `defaults/system.properties` line 442 leaves the property
  commented out, leaving the N4 code default `medium` in effect. Win Supervisor line 442 actively sets
  `niagara.moduleVerificationMode=low` — an explicit override to the more permissive value. Comment in
  BOTH files says `low` "will be removed in a future release." A self-signed module loads on the Win
  Supervisor (LOW) but fails on a JACE (MEDIUM) unless the cert is in the JACE's user trust store.
  `build-verify.md:110` already notes "JACE enforces the project CA" but does not state why (MEDIUM
  default vs LOW override). [ev: corpus B1004 §1004.2-1004.3]

- B18 §18.3.2-18.3.3 — `skipModuleValidation` bypass is an AND gate: BOTH
  `-Dniagara.classLoader.skipModuleValidation=true` (JVM system property) AND
  `Feature.getb("skipModuleValidation", false)` (requires `Webs.license` with `skipModuleValidation=
  "true"` feature, verified expiry 2027-03-31) must be true. The property ALONE does nothing. This
  is the correct bypass for dev modules that request `ACCESS_CLASS`/`REFLECTION`/`MBEAN_PERMISSION`
  on the Win Supervisor. NOT available on a JACE without the `Webs.license`.
  [ev: corpus B18 §18.3.2-18.3.3]

- B18 §18.9 — `exemptions.tes` (binary file, `user/security/exemptions.tes`, ~16 KB): Workbench dialog
  "Module XYZ is not validly signed. Trust permanently?" adds a per-module exemption that persists across
  restarts. Scoped to the Workbench USER only (not the station daemon). The developer bypass path for
  Workbench inspection of unsigned modules — less invasive than JVM flags, no license required.
  Requires admin credentials to modify; Workbench audit-logs each addition.
  [ev: corpus B18 §18.9]

## Proposed kit deltas (propose-never-apply)

| Δ | Delta | Target file / § | Token |
|---|---|---|---|
| Δ1 | Add the complete BPermissions 6-bit table: `OPERATOR_READ=1 / WRITE=2 / INVOKE=4`, `ADMIN_READ=16 / WRITE=32 / INVOKE=64` (bits 8 and 128 are unused gaps). Note that `OPERATOR_INVOKE=4` is what `canInvoke` checks for an `Flags.OPERATOR` action (B776), and what a servlet gates on when it proxies action invocations. Annotate the existing `OPERATOR_WRITE` line with its exact decimal value (2) so all 6 bits are co-located in one scannable reference. | `types/dashboard.md §RBAC` (after the existing `BPermissions.has(OPERATOR_WRITE)` bullet) | [ev: corpus B755 §755.3] |
| Δ2 | Add a `module-permissions.xml` source-format block: a module that needs JVM-level capabilities (network, reflection, JMX) declares them with SEMANTIC group names in `src/module-permissions.xml` using `<niagara-permission-groups type="…"> / <req-permission> / <name>NETWORK_COMMUNICATION</name> / <purposeKey>…</purposeKey> / <parameters>…`. This is a separate concern from the RBAC OPERATOR_WRITE bit check and from `module-include.xml`. Explain that this declares INTENT; the Gradle plugin expands to concrete Java permissions. | `types/structure.md §module-include.xml` (new sub-bullet after the existing module-include.xml vs META-INF/module.xml note) | [ev: corpus B18 §18.4.1] |
| Δ3 | Add source→runtime inlining fact: the Gradle signing plugin (`NiagaraPermissionGroupFactory`) expands the source `module-permissions.xml` group names into concrete `<java-permission class="…">` entries and INLINES them into the built JAR's `META-INF/module.xml`. There is no `module-permissions.xml` in the deployed artifact. To inspect what shipped: `unzip -p <mod>-rt.jar META-INF/module.xml \| grep -A20 permissions`. | `types/structure.md §module-include.xml` (alongside Δ2) | [ev: corpus B18 §18.4.2-18.4.3; B777 §777.4] |
| Δ4 | Add a signing-always groups rule to the signing section: three groups force Honeywell code signing regardless of `verificationMode` — `ACCESS_CLASS`, `REFLECTION`, `MBEAN_PERMISSION`. A module that declares any of these in `module-permissions.xml` MUST ship with the production Honeywell cert; self-signing or skipping signing will be rejected at load time even in LOW mode. Label this the "sign-or-refuse" group class. | `build-verify.md §signing` (after or alongside the existing dev-cert vs project-CA note) | [ev: corpus B18 §18.4.4, §18.3.1] |
| Δ5 | Add profile scoping rule: each permission group in `module-permissions.xml` is declared under `type="station"`, `"workbench"`, or `"all"`. A `type="workbench"` permission is never granted on the station daemon. A permission meant for the NRE must appear under `type="station"` or `"all"` or it will silently not apply. Verify by checking the decompiled `META-INF/module.xml` partition. | `types/structure.md §module-include.xml` (alongside Δ2/Δ3) | [ev: corpus B18 §18.4.7] |
| Δ6 | Add JACE=medium vs Win=LOW note: the Win Supervisor's `defaults/system.properties` explicitly sets `niagara.moduleVerificationMode=low`; the JACE-8000 QNX leaves the property commented out, falling back to the N4 code default `medium`. A dev-signed module that loads on the supervisor (LOW) will fail on a JACE (MEDIUM) unless the cert is imported into the JACE user trust store. The `build-verify.md:110` note on "JACE enforces the project CA" should name the reason. The `low` comment in both files says "will be removed in a future release" — treat `medium` as the stable target. | `build-verify.md §signing` | [ev: corpus B1004 §1004.2-1004.3] |
| Δ7 | Add `skipModuleValidation` AND-gate rule: the bypass for sign-always groups on the Win Supervisor requires BOTH the `-Dniagara.classLoader.skipModuleValidation=true` JVM property AND `Webs.license` carrying `skipModuleValidation="true"`. The property alone does nothing. Document the two-condition check and note it is NOT available on a JACE without a `Webs.license`. | `build-verify.md §signing` | [ev: corpus B18 §18.3.2-18.3.3] |
| Δ8 | Add `exemptions.tes` as the Workbench-user dev bypass: when a module needs Workbench inspection but can't use the production cert, the correct path is the Workbench "Trust permanently?" dialog → adds a per-module entry to `user/security/exemptions.tes`. Scoped to the Workbench user; does NOT affect the station daemon. Less invasive than JVM flags; requires admin credentials; Workbench audit-logs the addition. Note: this is for Workbench-local testing only — `skipModuleValidation` (Δ7) is needed for the station daemon. | `build-verify.md §signing` | [ev: corpus B18 §18.9] |

## Lessons

- The permissions surface splits cleanly into two orthogonal layers that are easy to conflate: (1) the
  **JVM security sandbox** (`module-permissions.xml`, permission groups, signing enforcement,
  `verificationMode`) — governs what the module's JVM code is ALLOWED to do; (2) the **application RBAC**
  (`BPermissions` bits, `BComponent.canInvoke`, the servlet `checkCanWrite` gate) — governs what a
  STATION USER is allowed to do with the module's components. The kit already covers (2) thoroughly
  (dashboard.md + logic-authoring.md); (1) was almost entirely absent.
- B776 (action protection) was the only component-level security item folded before this session; it
  lives in `logic-authoring.md §action protection` and covers the action-invoke half of layer (2).
- The JACE=medium vs Win=LOW asymmetry (B1004) is a concrete deploy trap: the existing build-verify note
  ("JACE enforces the project CA") is correct but opaque — the reason is that MEDIUM mode refuses self-
  signed unless the cert is in the trust store, while LOW only warns.
- `module-permissions.xml` source format and transformation have no prior representation in the kit;
  Δ2-Δ5 together close this gap.

---
**Status**: PENDING — INDEX row appended: `| 2026-09-18-permissions-security-model-deltas.md | kit | 2026-09-18 | pending | 8 |`
