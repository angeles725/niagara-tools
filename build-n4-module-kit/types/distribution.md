# Distribution — OEM packaging, `.dist` bundles, install-time migration

How Niagara packages platform releases (`.dist`), what an OEM vendor overlay
contains, trust-cert deployment for production, BOG schema-safety on upgrade,
version stamping, station templates, and AX→N4 migration readiness.

This doc is the **packaging and field-delivery** companion.
For signing gates and `verify-module.sh`, see `build-verify.md`.

---

## 1 · Module JAR vs distribution `.dist` — two orthogonal artifacts

| Artifact | What it is | What it carries | Author |
|---|---|---|---|
| `<mod>-rt.jar` | a signed module archive | your types, code, resources | module author (you) |
| `<name>.dist` | a ZIP with `META-INF/dist.xml` | JRE, NRE, OS images, JACE/Atlas firmware — **not module code** | Tridium / OEM |

A `.dist` is the vehicle for **platform-level upgrades** pushed from the
Supervisor to a field controller. A module JAR never travels inside a `.dist`;
it is deployed separately through `modules/`. `[ev: corpus B1023]`

---

## 2 · `dist.xml` metadata contract

`META-INF/dist.xml` governs how the Supervisor installs the bundle.
Critical attributes — wrong values corrupt field devices:

| Attribute / Element | Type | Effect / Gotcha |
|---|---|---|
| `@name` | string | display name in the Supervisor distribution manager |
| `@version` | string | the version string presented to `installDist.query` |
| `@vendor` | string | vendor identity |
| `@reboot` | boolean | **GOTCHA:** `true` forces an immediate controller reboot after install; omit or `false` only for hotpatch-safe bundles |
| `@noStation` | boolean | **GOTCHA:** `true` = installs without stopping the station; wrong value on a bundle that mutates station files = data corruption |
| `@absoluteElementPaths` | boolean | `true` treats payload paths as absolute on the target; otherwise relative to install root |
| `@osInstall` | boolean | `true` = the bundle replaces OS-level firmware (JACE QNX, Atlas snap); installer validates compatibility |
| `@updateDaemonBinaries` | boolean | `true` = the daemon binary itself is replaced; forces a mandatory restart |
| `<dependencies>` | child | ordered list of prerequisite dists; Supervisor validates before install |
| `<provides>` | child | capabilities this dist advertises (used by dependent dists) |

`[ev: corpus B1023 §1023.7]`

---

## 3 · Supervisor-as-installer (push model)

The Supervisor is the install tool for field controllers (JACE, Atlas):

1. Upload the `.dist` to the Supervisor distribution manager.
2. The Supervisor sends `installDist.query` to the target controller.
3. The controller validates `dist.xml` (@version, @dependencies, @osInstall compatibility).
4. On success it pulls the payload and applies it; `@reboot`/`@noStation` govern restart.

**Gotcha:** a `.dist` whose `@noStation=false` and `@reboot=false` that still
touches station files silently corrupts the running station. Test the
`@noStation`/`@reboot` pair against the actual payload contents before shipping.
`[ev: corpus B1023 §1023.1]`

---

## 4 · OEM overlay anatomy — step-11 last-write-wins

An OEM distribution is assembled by merging an `overlay/` tree over the stock
Niagara install. Installer step 11 is the merge; every OEM file silently
**overwrites** the corresponding Tridium file with no diff or backup.

| Overlay path | Carries | Notes |
|---|---|---|
| `overlay/modules/` | OEM module JARs pre-installed | present on every fresh station |
| `overlay/etc/brand.properties` | vendor name, product name, version string | shown in Workbench About and splash |
| `overlay/security/certificates/` | OEM trust certs (see §5) | must match the signing cert used to sign the OEM modules |
| `overlay/defaults/workbench/newStations/` | station templates (`.ntpl`) for OEM commissioning (see §8) | one file = one new-station template in Workbench |
| `overlay/conversion/` | firmware-level AX↔N4 conversion `.dist` bundles | JACE/Atlas conversion; not a station-migration vehicle |
| `overlay/cleanDist/` | factory-reset `.dist` | restores the controller to a clean baseline; not a migration vehicle |

`[ev: corpus B1026 §1026.2-1026.3, B1025]`

---

## 5 · OEM vendor trust certificates for production

**Format is Tridium-proprietary XML, NOT X.509.**

A production JACE or Atlas enforces `moduleVerificationMode=medium` (the
compile-time default) and requires a cert from the station trust store to
validate the module's signing chain.

### Cert file layout in `overlay/security/certificates/`

```
overlay/security/certificates/
  <vendorCert>.properties     # mandatory: description, version, enabled=true
  <vendorCert>.certificate    # Tridium-proprietary XML: DSA PUBLIC key only
```

The `.certificate` file is a **DSA public key** in Tridium's XML envelope — no
private key, no PEM, no X.509 DER. The signer holds the private key in the
build environment only. `[ev: corpus B1027 §1027.4]`

### Development path

During development, import the self-signed dev cert into the **Workbench user
trust store** (`<user_home>/security/cacerts.jceks`). The production cert goes
into the station's trust store through the overlay; it is never hand-imported
into a production JACE. `[ev: corpus B18 §18.9]`

### Hard signing gate (permission groups)

`ACCESS_CLASS`, `REFLECTION`, and `MBEAN_PERMISSION` force cert-chain
validation **regardless of `moduleVerificationMode`**. A local DEV cert is
insufficient for modules requesting these groups even on a Win Supervisor set
to LOW mode.

→ Full signing gate details and `skipModuleValidation` semantics are in
`build-verify.md §Signing per deploy target`. Do not duplicate here.

`[ev: corpus B1027 §1027.4, B18 §18.3-18.4]`

### Signing-profile build-triage

A missing or renamed signing alias does NOT fail the build uniformly — the outcome is
**profile-type-dependent** (bytecode-corroborated in `niagara-plugins-7.6.17.jar`):

| Profile type (`niagara.signing.profileType`) | Missing alias behavior | Build result | Station load result |
|---|---|---|---|
| `LocalSigningProfile` (dev default) | auto-generates a self-signed dev cert | **SUCCEEDS** — jar signed but untrusted | **FAILS** — `ValidationException` (BLD1 trust failure) |
| `RestrictedSigningProfile` (CI/release) | `IllegalArgumentException` | **FAILS** at build time | — |
| Any profile, profile FILE missing | `ProfileNotFoundException` | **FAILS** at build time | — |
| `requireSigning=false` (default), no aliases | UNSIGNED jar | SUCCEEDS | succeeds on DEV station, fails on production JACE |

**Deferred-failure trap:** a `LocalSigningProfile` with a missing alias ships a jar that the build
reports as GREEN but the station rejects at load time with an untrusted-cert `ValidationException`
(BLD1). The tell-tale symptom is "build signed OK, station refuses the module". Verify
`niagara.signing.profileType` in `~/.tridium/security/niagara.signing.xml` and confirm the intended
alias is present in the keystore before a release build.

`[ev: retro module-hardening-reqexec-closed-deltas Δ3]`

### Signature-failure triage pivot `[ev: retro module-hardening-failure-modes-deltas Δ10]`

A `ClassNotFoundException` during module load frequently has a ROOT cause of `ValidationException` deeper in the log. When you see a `ClassNotFoundException` on a module class, **grep the station log for `ValidationException`** FIRST before diagnosing a missing class:

| Log message | Root cause | Fix |
|---|---|---|
| `"No code signers found"` | jar is **unsigned** | sign the jar; check `requireSigning` in `niagara.signing.xml` |
| `"Error validating cert path"` | signer cert is **not trusted** by the station | import the cert into the station's trust store, or use Rebuild Module Signatures in Platform Software Manager |

**Triage pivot:** compare `moduleVerificationMode` across the failing station and a working station. If one is `low` (skip validation) and the other is `medium` or `high` (enforce trust chain), the failure is a trust mismatch, not a missing class. Fix the trust chain; do not lower the verification mode in production.

---

## 5b · `runtimeProfile` valid-set and corrupt-module trap `[ev: retro module-hardening-failure-modes-deltas Δ14]`

`runtimeProfile` in `module.xml` (generated from `<MOD>-rt.gradle.kts` by the plugin) is a **Java enum** — `RuntimeProfile.valueOf(string)` parses it case-sensitively. The EXACT valid values are:

| Value | Meaning |
|---|---|
| `rt` | real-time station profile |
| `ux` | browser/HMI profile |
| `wb` | Workbench profile |
| `se` | service edition profile |
| `doc` | documentation profile |

**Silent failure mode:** an invalid or misspelled value (e.g. `"RT"`, `"runtime"`, `"wb-rt"`) causes `RuntimeProfile.valueOf()` to throw `IllegalArgumentException`; the manifest parser swallows it into `BModuleStatus.corrupt` — the **only visible symptom at station boot is "module corrupt"** with the typo cause buried in the log. Always grep the module-load log for `IllegalArgumentException` before concluding that a module is mysteriously corrupt.

**Rule:** validate `runtimeProfile` in `module-include.xml` against the five exact lowercase values above before a production build. The Gradle plugin copies the string verbatim; it does NOT validate enum membership.

---

## 6 · BOG schema-safety matrix on module upgrade

The station decodes saved `.bog` data at boot using the **current** module JAR.
Class and slot changes to an already-deployed type have exact survival semantics.
Rule of thumb: **ADD, never retype.**

| Change | Survival | Field impact |
|---|---|---|
| ADD new slot | Safe | new slot gets default value; existing rows unaffected |
| REORDER slots | Safe | `.bog` binding is by NAME, not ordinal |
| RENAME slot | **Orphaned** | old name creates an orphan child; new slot gets default; data is NOT migrated automatically |
| RETYPE slot (different `BTypeSpec`) | **OUTAGE** | decoder throws; station fails to start until the data is repaired |
| REMOVE simple slot (leaf value) | **Lossy** | value is silently discarded at load; no error |
| REMOVE complex slot (sub-component) | **Shunt** | sub-component is detached and moved to an orphan folder; links break |
| Enum tag rename / remove | **OUTAGE** | stored ordinal maps to a missing label; decode throws |
| Class missing (module removed/renamed) | **OUTAGE** | `MissingClass` at boot for every instance of that type |

**Complex migration recipe (the safe path):**

```java
// In the NEW module:
//  1. ADD the replacement slot with a new name.
//  2. In started(), read the old slot (still present) and write the new one.
//  3. Deprecate the old slot with @deprecated in the lexicon; REMOVE in the NEXT release.
@Override public void started() throws Exception {
    super.started();
    if (get(oldSlot) != null && get(newSlot).isDefaultValue()) {
        set(newSlot, get(oldSlot));   // migrate once; old slot still decodes
    }
}
```

`[ev: corpus B754 §754.5-754.7]`

### Restore direction rule `[ev: retro module-hardening-failure-modes-deltas Δ9]`

`verifyDependencies` is a **FLOOR check**: it validates that the installed module version is ≥ the manifest's declared minimum, NOT that the versions exactly match.

| Restore direction | Result |
|---|---|
| OLDER `.dist` onto a NEWER station | **SAFE** — installed version ≥ manifest minimum; restore proceeds |
| NEWER `.dist` onto an OLDER station | **FAILS** — installed version < manifest minimum; dependency check rejects the restore |

**Rule:** deploy in the older-onto-newer direction only. Document the intended deploy direction in module release notes. Never validate restores in only one direction — a NEW `.dist` onto an OLD station is the typical field failure. `ignoreDependencies` bypasses the check (emergency use only; document why).

---

## 7 · Module version stamping for a suite

### Where to set the version

Set `defaultModuleVersion` in the **GROUP** `build.gradle.kts` (the one that
applies to all sub-projects). Never set it per-part `.gradle.kts` — the GROUP
setting propagates automatically. `[ev: corpus B755 §755.4]`

### 3-part floor, not 4-part build-stamp

```kotlin
// build.gradle.kts (GROUP root)
defaultModuleVersion("X.Y.Z")          // ✅ 3-part floor — the MINIMUM version
// vendorVersion="4.14.0.100"          // ❌ 4-part includes build stamp — fails L7
```

A 4-part build stamp (`4.14.0.100`) sets a specific version floor, not a
minimum. Any field controller running an older build stamp is refused.
The 3-part form (`4.14.0`) means "any 4.14 build at or above base 0".

### `BVersion.MEETS_MINIMUM`

Use `BVersion.MEETS_MINIMUM` (not `==`) for runtime version guards:

```java
BVersion actual = Sys.getModuleManager().getModule("mymod").getVersion();
if (!actual.meetsMinimum(new BVersion("1.3.0"))) {
    appFail("mymod 1.3.0 or later required");
}
```

`[ev: corpus B755 §755.4]`

---

## 8 · Station template `.ntpl` for OEM commissioning

A `.ntpl` file is a ZIP containing a pre-configured station BOG
(`config.bog`) plus a `template-manifest.xml` that declares the template's
display name and description.

### Placement

```
overlay/defaults/workbench/newStations/
  MyProductStation.ntpl     # appears in Workbench: File › New Station › …
```

Workbench scans `<niagara_home>/defaults/workbench/newStations/` at launch
and lists every `.ntpl` it finds in the New Station wizard. An OEM `.ntpl`
pre-configures the driver network, alarm classes, history extensions, and
service slots an OEM product needs — eliminating per-unit manual wiring.

`[ev: corpus B1021, B1024 §1024.4]`

---

## 9 · AX→N4 migration readiness

The migration tool is an **offline tool operation**, not a runtime hook. Key rules:

| Rule | Detail |
|---|---|
| N4 module must exist first | Run `install` or overlay-deploy the N4 JAR **before** running migrate; the migration tool resolves the new type at startup |
| `BModuleRemovalConverter` handles removals | Handles cases where a module is removed entirely and not replaced; does NOT handle type replacements (that requires a custom `BConverter`) |
| Replacements need `BConverter` | Write a `BConverter` and register it in `module-include.xml` to remap old-AX type names to new-N4 type names |
| Migration is offline | The NRE must be stopped; migration mutates `.bog` files directly |
| What migrates | Logic programs, alarm classes, history extensions, px view refs (limited); complex graphics and AX drivers do NOT migrate |
| What does NOT migrate | Driver points, schedule DBs, alarm routing full, graphics/PX bound to AX-only widgets |

`[ev: corpus B1024 §1024.3-1024.5, B405]`

---

---

## 10 · Station stuck at boot — Platform Software Manager recovery

When a station refuses to boot because of a bad or wrong-version module (unsigned/untrusted per
BLD1/BLD7, bad manifest, incompatible version), recovery is a **Platform-level operation** performed
while the **station is DOWN** (the platform daemon must be running):

### Recovery procedure

1. Connect Workbench to the **Platform node** (not the station node).
2. Open **Platform > Software Manager**.
3. Identify the offending module — it appears as:
   - **"Bad Target"** — bad manifest or otherwise unusable (incompatible/corrupt).
   - **"Out of Date"** — module version does not match the installed platform family.
4. Select the module and choose a remediation action:

| Action | When to use |
|---|---|
| **Downgrade** | roll back a newer-than-expected module to a known-good lower version |
| **Uninstall** | remove a module with no dependents; the station will boot without it |
| **Import + Re-Install** | install the correct build via a fresh JAR upload |
| **Rebuild Module Signatures** | repair the BLD1/BLD7 trust failure (signs the existing JAR against the station's current cert chain); **station must be not-running** |

5. Click **Commit** after selecting the action; the platform applies the change.
6. Restart the station.

**Critical gotcha — NEVER use the File Transfer Client for modules:** the Niagara File Transfer
Client copies raw bytes only; it does NOT apply runtime profiles or signing chains. A module
deployed this way bypasses the Software Manager's installation pipeline and will fail the station's
module-verification check on next boot, even if the JAR bytes are correct.
`[ev: corpus B1139]`

`[ev: retro module-hardening-reqexec-closed-deltas Δ7]`

---

---

## 11 · Firmware/file OTA via chunked Base64 action invocations [ev: retro honeywell-wb-rt-wb-deltas Δ5]

When an rt action can accept only small strings and you need to push a firmware image or large file from the WB to a device, use ≈5 000-character Base64 segments sent as repeated action invocations, terminated by a sentinel string `"END"`. The rt side accumulates segments in a `StringBuilder`, Base64-decodes on the sentinel, then writes the binary. This avoids any single-payload size limit without a custom Fox file channel.

See `types/logic-authoring.md §Chunked Base64 transfer for firmware/file OTA` for the full WB + rt code recipe. `[ev: corpus B1080]`

---

**See also:** `build-verify.md` (signing gate, `verify-module.sh`, JACE vs Win Supervisor
verification modes), `types/structure.md` (module-include.xml, permission groups),
`types/logic-authoring.md` (started() lifecycle seam for migrations).
