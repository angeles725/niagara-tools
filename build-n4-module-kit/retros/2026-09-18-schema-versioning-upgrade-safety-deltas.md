<!-- review-status: folded -->
# 2026-09-18 · kit · schema-versioning-upgrade-safety-deltas

**Session**: corpus-mining for build-n4-module (operator: generate all possible kit deltas).
**Delta count**: 6

## What happened

Corpus blocks B739, B754, B817, and B740 were mined for schema-evolution and versioning
content not yet in the kit core.  The existing retro `2026-09-03-slot-type-change-rompe-bog`
and `2026-09-05-campaign7-schema-risk.md` already fold the foundational rules
(never-retype, survival matrix, schema-risk.sh CSV table, cross-module-enum-as-double).
This session surfaces six gaps that survived the fold.

## Evidence

- B739 §739.1-3 — `ValueDocDecoder`/`ValueDocEncoder` source-read; the parse-desyncs-on-retype
  mechanism and the safe-vs-unsafe change table.  [ev: corpus B739 §739.2]
- B754 §754.1 — two distinct version types: runtime `javax.baja.util.Version` (dewey,
  longer-wins-tie `1.0.1 > 1.0`, `Version.compareTo:141-159`) vs install-layer
  `com.tridium.install.BVersion` (bit-flag `meetsVersionRequirement:135-142`).  [ev: corpus B754 §754.1]
- B754 §754.3 — `BModule` is `final`; grep of `com/tridium/sys/module/` and `javax/baja/sys/`
  found NO `upgrade()`/`migrate()`/`moduleStarted()` callback.  Whole-station conversion is the
  offline `migration-rt` (`BIFileMigrator` + converter SPIs).  [ev: corpus B754 §754.3]
- B754 §754.4 — `ValueDocEncoder.encodeType:1081-1088` records only the module name (string);
  no version, no hash.  The Slot-o-Matic type-hash `/*@ …(2979906276)… @*/` is a build-time
  staleness marker, never read at load.  [ev: corpus B754 §754.4]
- B754 §754.6 — `remove_or_rename_enum_tag` → `getRange().get(tag)` → `InvalidEnumException`
  unwrapped → OUTAGE; already in schema-risk.sh CSV, but `parse_slots()` only harvests
  `@NiagaraProperty` entries — a `@Range` tag change produces zero diff rows.  [ev: corpus B754 §754.6]
- B754 §754.6 row r5 + schema-risk.sh CSV note `fox-sync-unsafe` — `renumber_enum_ordinals`
  is SAFE for `.bog` (tag-based) but Fox encodes the ordinal as an integer; wrong ordinal
  decoded on the remote side without any error.  The CSV note exists but no WARN row is emitted.
  [ev: corpus B754 §754.6; schema-risk.sh CSV line `renumber_enum_ordinals,SAFE,…,fox-sync-unsafe`]
- B740 §740.2-3 — cross-module `@NiagaraProperty type=` on a sibling custom-module enum compiles
  with `compileOnly(files(...))` (does NOT reach plugin classpath) and passes `verify-module.sh`,
  yet fails at station load as `Missing class <mod>:<Enum>`.  [ev: corpus B740 §740.2]
- B754 §754.3 — self-migration seam: detect orphaned dynamic slot by name in `started()`,
  copy value to new slot, remove old slot.  Concept is in corpus-index retro description of B754
  but no concrete recipe exists in `METHODOLOGY.md` or `types/logic.md`.  [ev: corpus B754 §754.3]

## Proposed kit deltas (propose-never-apply)

| Δ | Delta | Target file / § | Token |
|---|-------|-----------------|-------|
| Δ1 | Add rule: N4 has TWO distinct version concepts that must not be conflated. Runtime `javax.baja.util.Version` (dewey, longer-wins-tie) governs boot-time dep resolution — a missing dep aborts boot. Install-layer `com.tridium.install.BVersion` (bit-flag `meetsVersionRequirement`) governs which JARs the installer deploys before the station runs. Neither gates schema decode. The `.bog` is reconciled by name against whatever module version is installed now; bumping `vendorVersion` is audit trail only, not a decode gate. | `METHODOLOGY.md §Schema/upgrade safety` (new bullet, after S3) | `two-version-layers` |
| Δ2 | Add concrete recipe: when a slot's shape MUST change (add-new is insufficient), the decoder gives no convert hook. Use: (a) ADD the new slot; (b) in `started()` (or `atSteadyState()` if the station may have saved-data to migrate), detect the orphaned dynamic slot by name (`get(oldName) != null && slot is dynamic`), copy its value to the new slot, remove the old slot; (c) leave-old-deprecated until confirmed no station carries the old form. `BModule` is `final` with no `upgrade()` callback; whole-station migration is the offline `migration-rt` tool, not available in-component. | `types/logic.md §Schema-safe evolution` (new sub-bullet after existing "ADD-new" cross-ref) | `started-self-migration-recipe` |
| Δ3 | Add clarification: the Slot-o-Matic type-hash comment `/*@ …(hash)… @*/` in the generated AUTO region is a BUILD-TIME staleness marker only — it tells the annotation processor the region is current; `ValueDocEncoder.encodeType` never writes it to the `.bog`; the loader never reads it. A deployed schema change is invisible to the `.bog` decoder regardless of whether the hash changed. Prevents the false assumption that "the hash mismatch means Niagara re-validates the schema." | `METHODOLOGY.md §Schema/upgrade safety` (sentence appended to S2 matrix intro) | `slotomatic-hash-buildtime-only` |
| Δ4 | Add documented limit `L4` to schema-risk.sh header: `@Range` enum-tag changes are not auto-detected from `@NiagaraProperty` snapshots. `parse_slots()` harvests only `@NiagaraProperty` entries; a `BFrozenEnum` subtype with `@Range({"auto","hand"})` that removes `"hand"` produces zero diff rows — `remove_or_rename_enum_tag,OUTAGE` in the CSV is never triggered automatically. Manual pre-deploy check required: diff `@Range({…})` declarations across the before/after snapshot to detect tag removals. | `toolbelt/schema-risk.sh` (add `#   L4  @Range enum-tag changes (add/remove/rename tag on a BFrozenEnum @Range) are NOT auto-detected from snapshots; parse_slots harvests @NiagaraProperty only. Manually diff @Range declarations before deploy.` under the existing L1-L3 block) | `schema-risk-enum-tag-blind-spot` |
| Δ5 | Propose new `check_cross_module_type` WARN under `--src` in verify-module.sh: scan `@NiagaraProperty` annotations in the module source for `type=` values whose class package is neither `javax.baja.*`, `com.tridium.*`, `com.tridiumx.*`, `com.honeywell.*`, nor the current module's own `com.<vendor>.<Module>.*`. A foreign custom-module type compiles with `compileOnly(files(...))` (not on the plugin classpath) and passes the verify gate, but fails at station load as `Missing class <mod>:<Type>`. WARN severity (may be intentional with a correct runtime dep). Evidence: live JACE `Missing class ColdRoomPan:HoaMode` incident. | `toolbelt/verify-module.sh` (new `check_cross_module_type` function under `--src`; `WARN cross-module-type <jar> <detail>`) | `cross-module-type-warn` |
| Δ6 | Promote `renumber_enum_ordinals` in schema-risk.sh from silent SAFE to SAFE+emitted WARN: the CSV note `fox-sync-unsafe` exists but no warning row is printed to the report. Fox protocol encodes enum values as integer ordinals; a renumbered enum flowing over a Fox link (supervisor ↔ JACE) decodes the wrong tag on the consumer side without any error or exception. Add a `WARN fox-sync` row in the output when the diff classifier emits `renumber_enum_ordinals`. Also add one sentence to `METHODOLOGY.md §Schema/upgrade safety` S2: "renumbering enum ordinals is SAFE for `.bog` (tag-based) but UNSAFE for Fox/binary links (ordinal-encoded); avoid it if the value crosses a Fox link." | `toolbelt/schema-risk.sh` (emit WARN row for `renumber_enum_ordinals` kind); `METHODOLOGY.md §Schema/upgrade safety` (append to S2) | `fox-ordinal-renumber-warn` |

## Lessons

1. **The kit already captures the hard failures** (retype-outage, survival matrix, enum-as-double,
   schema-risk.sh CSV table) — the remaining gaps are subtler: the distinction between version
   concepts, the tool's blind spot for `@Range`, and the Fox ordinal edge case.

2. **The corpus-index retro description of B754 ("ADD-new + migrate-in-started()") is ahead of
   METHODOLOGY.md**: the concept appears as a parenthetical in the index description but was never
   promoted into a concrete recipe that a module author can follow.  Δ2 closes that gap.

3. **`renumber_enum_ordinals,SAFE` is a trap**: the CSV note `fox-sync-unsafe` is correct but silent.
   A schema-risk run on a module with renumbered enums exits 0 (SAFE) with no warning — the Fox
   hazard is invisible unless the author reads the CSV source.  Δ6 surfaces it.

4. **schema-risk.sh's L4 gap is a silent false-SAFE on enum-tag changes**: the CSV row exists, the
   evidence is in the corpus, but the tool can never reach it from `@NiagaraProperty` snapshots.
   Until `@Range` parsing is added, schema-risk gives a false clean bill on enum-tag removals.

---
**Status**: PENDING — INDEX row appended: `| 2026-09-18-schema-versioning-upgrade-safety-deltas.md | kit | 2026-09-18 | pending | 6 |`
