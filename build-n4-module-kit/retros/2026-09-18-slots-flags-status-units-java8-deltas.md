<!-- review-status: folded -->
# 2026-09-18 · kit · slots-flags-status-units-java8-deltas

**Session**: corpus-mining for build-n4-module (operator: generate all possible kit deltas from the slot/property mechanics + bit-models + Java-8 discipline vein).
**Delta count**: 13

---

## What happened

Mined B4, B18, B734, B735, B736, B738, B745, B755, and B756 against the current kit (toolbelt/verify-module.sh, types/logic.md, types/logic-authoring.md, METHODOLOGY.md, build-verify.md, corpus-index.md, and all retros). Cross-checked the INDEX.md for already-folded items in this topic area to avoid duplication.

**Skipped as already folded** (present in types/logic.md, METHODOLOGY.md, or verify-module.sh):
- Java-8 bytecode major-52 check: `check_bytecode_major` in verify-module.sh.
- OPERATOR numeric without facets (WARN): `check_facet_presence`.
- Raw-number MIN/MAX facet: `check_raw_double_facets`.
- Slot flags (TRANSIENT, SUMMARY, READONLY, ASYNC, HIDDEN, DEFAULT_ON_CLONE, FAN_IN, OPERATOR) in logic.md §Flags and METHODOLOGY.md §Slot flags.
- `BUnit.getUnit("celsius")` + `BFacets.make(UNITS, …)` in METHODOLOGY.md §Facets.
- `getSlotFacets(Slot)` projection in logic.md.
- BStatus `isValid()` gate + `propagate` call in logic.md §Slots.
- `BIcon.make(BOrd.make(…))` with PNG in METHODOLOGY.md §Icon.
- `CONFIRM_REQUIRED` = UX-only confirm in dashboard.md.
- `LINK_TARGET` advisory in logic.md.
- `BPermissions.OPERATOR_WRITE` bit in dashboard.md.
- Permission source format + inlining into module.xml in METHODOLOGY.md §Build.

---

## Evidence

- BStatus.java:46-53 gives the 8 exact bit values; :51-140 the typed factory family; :287,299 `isValid`/`isOk`.
  [ev: corpus B736 §736.2-3]
- `makeOverridden` should mark a HOA-HAND or writable-override forced output so the HMI/subscriber can distinguish auto from forced. Currently the kit says "propagate aggregated status" but never says SET OVERRIDDEN.
  [ev: corpus B736 §736.4]
- `makeNull(s, true)` is the correct output when sensor is absent or value indeterminate; writing 0.0 with ok status is a lie.
  [ev: corpus B736 §736.4]
- `BFacets.makeNumeric(BUnit unit, BInteger precision, BNumber min, BNumber max)` packs UNITS+PRECISION+MIN+MAX in one builder. METHODOLOGY.md uses the two-key `BFacets.make(UNITS, …)` form; the `makeNumeric` overload is undocumented in the kit.
  [ev: corpus B745 §745.2]
- `BUnit.NULL` = no unit — the correct sentinel for dimensionless or intentionally unit-less slots when building `BFacets.makeNumeric(null, …)`.
  [ev: corpus B745 §745.1]
- Offset units (celsius, fahrenheit): `BUnit` handles the +273.15/+32 shift internally; hand-rolling `°C = raw + 273.15` breaks `UNIT_CONVERSION` and gives wrong engineering values.
  [ev: corpus B745 §745.3, BUnit.java:378-392]
- `BKitNumeric` declares `propagateFlags` (`BStatus`, SUMMARY|OPERATOR) as a user-configurable mask that ANDs which status bits flow through to the output. Custom components can adopt the same pattern; the kit's LC4 entry only documents CHECKING kitControl's existing propagateFlags, not adding one to a NEW component.
  [ev: corpus B738 §738.3, BKitNumeric.java:35]
- `BIUnlinkableSlotsContainer` lets a component list specific child slots that must never be a link source or target while remaining visible in the property sheet and writable by an operator. HIDDEN removes the slot from ALL UI; BIUnlinkableSlotsContainer keeps it visible but unlinked. Not in kit.
  [ev: corpus B735 §735.4]
- The TRANSIENT flag trap: TRANSIENT means NOT persisted to the .bog — accidentally combining TRANSIENT with a config/setpoint slot silently discards the operator's value on every restart. The kit says "TRANSIENT for computed outputs" but does not warn about the opposite misuse. A WARN check on TRANSIENT slots that are also OPERATOR (i.e. operator-tunable but ephemeral) would catch this.
  [ev: corpus B755 §755.5, B4 §4.1.2]
- `BIcon.std(fileName)` resolves to `"module://icons/x16/" + fileName` (BIcon.java:69-71) — a shorthand for in-root raster icons. SVG via `BIcon.make("module://<mod>/icons/myicon.svg")` avoids maintaining x16/x32 raster pairs. METHODOLOGY.md documents only the PNG+BIcon.make(BOrd) form.
  [ev: corpus B738 §738.4]
- rt/ux profile bytecode is Java-8 (major 52) but also bound to **Compact 3** (the JRE subset shipped with the NRE) — no `java.awt`, no `javax.swing`, no `java.sql` (not in Compact 3). wb/se profiles run the full SE JRE. An rt class that imports `java.awt.*` compiles fine on a full JDK but crashes on the station with `NoClassDefFoundError`. The verify-module.sh bytecode check tests major version only, not Compact 3 compliance.
  [ev: corpus B756 §756.2]
- Three permission groups (`ACCESS_CLASS`, `REFLECTION`, `MBEAN_PERMISSION`) force `requiresSignature()=true` regardless of `niagara.moduleVerificationMode`. A module that requests any of these three will FAIL to load in LOW mode if unsigned. The current kit says "both jars must be signed" generally but does not flag this as an extra-hard signing gate.
  [ev: corpus B18 §18.3.1, §18.4.4]
- `BVersion.MEETS_MINIMUM` mask = 115 (LATER|SAME|EQUIVALENT|MORE_SPECIFIC|LESS_SPECIFIC, excluding EARLIER=4 and DIFFERENT=8). A `<dependency vendorVersion="4.14">` (minimum) is satisfied by 4.15 but NOT by 4.13 and NOT by a different vendor. This explains the behavior of the `--target-version` gate check.
  [ev: corpus B755 §755.4, BVersion.java:58-66]

---

## Proposed kit deltas (propose-never-apply)

| Δ | Delta | Target file / § | Token |
|---|-------|-----------------|-------|
| Δ1 | Add the 8-bit **BStatus reference table** (DISABLED=0x01…UNACKED_ALARM=0x80, ok=0) sourced from BStatus.java:46-53; add the typed factory names (`makeFault`, `makeDown`, `makeNull`, `makeStale`, `makeOverridden`, `makeDisabled`(orig, state)) to the §Slots block so builders know which factory to call | `types/logic.md` §Slots | `bstatus-table` |
| Δ2 | Add authoring rule: **set `makeOverridden(s, true)` on an output when it is forced by a HOA HAND or writable override** so the HMI and any subscriber can distinguish a forced value from an auto-computed one; the kit currently documents reading OVERRIDDEN but not producing it | `types/logic.md` §Slots | `overridden-produce` |
| Δ3 | Add authoring rule: **use `BStatus.makeNull(s, true)` when a sensor is absent or the computation is indeterminate** — write a NULL-status output, not a 0.0 with ok-status (a 0.0/ok lie can drive a false control decision downstream) | `types/logic.md` §Slots | `null-produce` |
| Δ4 | Document **`BFacets.makeNumeric(BUnit unit, BInteger precision, BNumber min, BNumber max)`** as the canonical one-call builder (packs UNITS+PRECISION+MIN+MAX); update the §Facets recipe in METHODOLOGY.md to show this overload alongside the existing `BFacets.make(UNITS, …)` two-key form | `METHODOLOGY.md` §Facets | `makeNumeric-overload` |
| Δ5 | Add **`BUnit.NULL`** note: when building `makeNumeric(null, …)` or a slot that is explicitly dimensionless, pass `BUnit.NULL` (not a string `"null"` nor omit the arg) — avoids a NullPointerException in the units renderer | `METHODOLOGY.md` §Facets | `bunit-null` |
| Δ6 | Add **offset-units trap**: never hand-roll `°C = raw − 273.15` or `°F = raw × 1.8 + 32` for display or conversion; set the `UNITS` facet to `BUnit.getUnit("celsius")` / `"fahrenheit"` and let `BFacets.UNIT_CONVERSION` and the field editor handle offsets — `BUnit` special-cases them at BUnit.java:378-392, manual conversion breaks the framework | `METHODOLOGY.md` §Facets | `offset-units-trap` |
| Δ7 | Add authoring recipe: **add a `propagateFlags BStatus` slot (SUMMARY\|OPERATOR, default `BStatus.ok`) to custom control components** to give operators operator-side control of which status bits (fault/stale/down/null) propagate to outputs — mirror the `BKitNumeric` pattern; when propagating input status, mask by `getPropagateFlags()` before `out.setStatus(…)` | `types/logic.md` §Propagation | `propagateflags-custom` |
| Δ8 | Document **`BIUnlinkableSlotsContainer`**: implement it to exclude specific child slots from the link picker without hiding them from the property sheet. Use case: a setpoint or config slot that must remain operator-writable but should never be driven by a link (e.g., an HOA override that must be set by hand, not by a wire-sheet link) | `types/logic.md` §Links | `biunlinkable-container` |
| Δ9 | Add **TRANSIENT trap** WARN check: a slot that carries `OPERATOR` (operator-writable → intentionally persisted config) and ALSO `TRANSIENT` (not persisted) is almost certainly a bug — the operator sets a setpoint, the station restarts, the setpoint reverts to default silently; add a `check_transient_operator` WARN rule in `verify-module.sh` that flags any source slot combining `TRANSIENT` with `OPERATOR` | `toolbelt/verify-module.sh` + `types/logic.md` §Flags pitfalls | `transient-operator-warn` |
| Δ10 | Add **SVG icon recipe**: `BIcon.make("module://<mod>/icons/myicon.svg")` via `BIcon.make(String ordList)` (BIcon.java:40-56) serves a single scalable file instead of x16+x32 PNG pairs; `BIcon.std(fileName)` is the shorthand for in-root raster at `module://icons/x16/<fileName>`; cache in a `static final BIcon` field — never build per-call | `METHODOLOGY.md` §Icon | `svg-icon-recipe` |
| Δ11 | Add **rt/ux = Compact 3** profile constraint to build-verify.md: the NRE ships the Compact 3 JRE subset — `java.awt`, `javax.swing`, `java.sql`, `javax.xml` (DOM) are absent; an rt or ux class that imports these compiles under a full JDK 8 but throws `NoClassDefFoundError` on the station; propose a `check_compact3_imports` WARN rule (grep for `import java.awt\|import javax.swing\|import java.sql` in rt/ux profile source) | `build-verify.md` §Profile constraints + `toolbelt/verify-module.sh` | `compact3-imports` |
| Δ12 | Add to build-verify.md §Permissions: **ACCESS_CLASS, REFLECTION, and MBEAN_PERMISSION always require a validly-signed module regardless of `moduleVerificationMode`** — `requiresSignature()=true` for these three groups forces cert-chain validation even in LOW mode; a module requesting any of them loads with `ValidationException` if unsigned; call this out as a hard signing gate beyond the general "both jars must be signed" rule | `build-verify.md` §Permissions | `hard-sign-gate` |
| Δ13 | Add to build-verify.md §Versioning: **`BVersion.MEETS_MINIMUM` mask = 115** (`LATER\|SAME\|EQUIVALENT\|MORE_SPECIFIC\|LESS_SPECIFIC`; excludes EARLIER=4 and DIFFERENT=8) — this is why `<dependency vendorVersion="4.14">` (minimum) is satisfied by 4.15 but not by 4.13 or a different vendor, and why the `--target-version` gate FAILs on a jar stamped with a version above the target; cite BVersion.java:58-66 | `build-verify.md` §Versioning | `meets-minimum-mask` |

---

## Lessons

1. **The kit's propagation story is read-side only.** logic.md documents `isValid()` gating and `propagate()` but never the write-side: which factory to call (`makeNull`, `makeOverridden`, `makeFault`) and when. B736 closes that gap with exact method names and use cases.

2. **`BFacets.makeNumeric` is a better default than the two-key form.** The kit teaches `BFacets.make(UNITS, u)` but makeNumeric packs UNITS+PRECISION+MIN+MAX in one call and is what Tridium's own `BSequence` uses — it prevents a common "facet with no precision" oversight.

3. **Compact 3 is the silent build gap.** The verify-module.sh bytecode check confirms Java-8 compilation but can't see Compact 3 exclusions; a single `import java.awt.Color` in an rt class sails through the gate and crashes live. A grep-based WARN is cheap to add.

4. **ACCESS_CLASS/REFLECTION/MBEAN are hard signing gates.** Most module authors know "sign your jars" from the general rule, but the three permission groups that override LOW mode are not called out — a module gets signed, deployed, and still fails to load because someone added a `REFLECTION` permission group without knowing it now requires Honeywell-CA trust.

5. **propagateFlags on custom components** closes an operator-UX gap: kitControl blocks expose this mask; our refrigeration components currently hardcode their propagation (any fault/stale input faults the output). Adopting the BKitNumeric pattern lets a commissioning engineer tune it per site without a code change.

---

**Status**: PENDING — INDEX row appended: `| 2026-09-18-slots-flags-status-units-java8-deltas.md | kit | 2026-09-18 | pending | 13 |`
