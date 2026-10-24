<!-- review-status: folded -->
<!-- module: kit -->
<!-- deltas: 10 -->
# 2026-10-15 · kit · devguide415-r2-packaging-platform-signing

**Session**: research-sdd focus `dev-capabilities-kit-gaps` round 2 read the real 4.15.6.20 `.dist` manifests and the Templates, Provisioning, Lexicon, ModuleSign, SigningService and Platform user guides; this retro carries corpus B1318 (dist part), B1325, B1330, B1332 and B1341. Complements `2026-10-15-devguide415-horizons-authoring-security.md` (**has** Δ4 signing roster, Δ8 `se`) and `2026-10-15-n4-course-build-deps-permissions-signing.md` (**bdps** Δ4).
**Delta count**: 10

## What happened
Packaging and deployment rules the kit states are partly inverted and mostly missing. (1) In all 26 readable shipped `dist.xml` files `noStation` is the live alias of the devguide's `noRunningApp`, and `true` means the dist may NOT be installed while a station runs (22/26 Tridium dists) — the kit says `true` "installs without stopping the station". (2) Templates: a template module is a `.jar` of read-only `.ntpl`; manifest dependencies must list every module the logic/Px needs; upgrade/redeploy wipe local edits; changes never propagate; the kit conflates palette entries with `.ntpl` and links a non-existent `§Templates`. (3) Lexicon-only modules have naming and Brand-Pattern rules. (4) The Signing Service signs TLS/MQTT certificates, not modules; modes × statuses decide load/warn; the guide's "default Low" is stale. (5) `se` jars run on Windows hosts only — the kit lists JACE as an `se` host; a JACE has `-ux`/`-wb` jars only if the Configure Runtime Profiles wizard enabled them; a new module installs hot but any upgrade/downgrade stops the station and reboots the host.

## Evidence
- `noStation` `[ev: code organized/platform/platform-rt/vineflower/com/tridium/install/installable/DistributionManifest.java:87,130]` `[ev: niagara-help 4.15.6.20 devguide-clean/distribution.txt:112-118]` vs `[ev: code types/distribution.md:37,40]` `[ev: corpus B1318 §1318.1]` (B1023:284 already correct)
- Templates `[ev: niagara-help 4.15.6.20 guides-clean/Templates/UpgradingADeployedTemplateTemplates-9A17381D.txt:25]` `[ev: …/Templates/DeployingTemplates-054F175F.txt:33]` vs `[ev: code types/logic.md:185]` `[ev: code types/structure.md:133]` `[ev: corpus B1325]`
- Lexicon modules `[ev: niagara-help 4.15.6.20 guides-clean/Lexicon/devkit-LexiconModuleBuilder.txt:14-43]` `[ev: corpus B1330]`
- Signing `[ev: niagara-help 4.15.6.20 guides-clean/SigningService/signingService-SigningService.txt:16]` `[ev: …/ModuleSign/VerificationModesModuleSigning-182D0E86.txt:25-36]` `[ev: …/ModuleSign/SignatureStatusesModuleSigning-1F0024B2.txt:22-45]` `[ev: corpus B1332]`
- Platform `[ev: niagara-help 4.15.6.20 guides-clean/Platform/aPlat4PlatformAdminConfigureRuntimeProfiles.txt:49-51]` vs `[ev: code types/structure.md:300]` `[ev: corpus B1341]`

## Proposed kit deltas (propose-never-apply)
| Δ | Delta | Target file / § | Token |
|---|---|---|---|
| Δ1 | Fix the dist attribute table: `@noStation` is the shipped alias of `noRunningApp`; `true` = install only with every station stopped (22/26 Tridium dists); devguide defaults `reboot`/`noRunningApp` true, code reads false — always write both; `updateDaemonBinaries` does not itself reboot; add `description`, `buildDate`, `buildHost`, `releaseDate`, `<exclusions>`. The `:57-59` gotcha stays (it reads the flag correctly). Lint: yes — FAIL on "installs without stopping"; WARN on a dist.xml without explicit `reboot`/`noStation`. | `types/distribution.md:35-45` | `[ev: corpus B1318 §1318.3 C1-C2]` |
| Δ2 | Add "Shipping a template": template module via Make Module (read-only `.ntpl`); manifest `<dependencies>` lists every `-rt`/`-ux` module; decide "Use Minor Version On Deployment" explicitly. Lint: partial — unzip `.ntpl` manifest vs `module.xml`. | `types/logic-authoring.md:208-211` | `[ev: corpus B1325 rows 1-3]` |
| Δ3 | Add "What breaks on module upgrade": template changes never propagate; upgrade/redeploy remove-then-install and wipe local edits; app templates untracked (record their version); a renamed slot breaks exposed `BConfigBinding`s. Lint: partial. | `METHODOLOGY.md:44-52` | `[ev: corpus B1325 rows 4-6]` |
| Δ4 | Fleet install via Provisioning (`!sw`, explicit version, auto-included registry deps, `provisioningNiagara` licence); fix the dangling `§Templates` link and say a palette entry is not a `.ntpl` (reinforces afl Δ3). Lint: yes — dangling `§` link check. | `types/distribution.md` + `types/logic.md:185` + `types/structure.md:133` | `[ev: corpus B1325 rows 8-9, C1-C2]` |
| Δ5 | Lexicon-only modules: names `customLexicon<Xxxx>`/`<Vendor>Lexicon<Xxxx>`, preferred symbol 1-8 letters, Brand Pattern gating, no classes; customer overrides in `!lexicon/<lang>` apply after restart; a slot Display Name beats the lexicon. Lint: partial. | `types/structure.md` lexicon § + `types/dashboard.md` i18n | `[ev: corpus B1330 §1330.2]` |
| Δ6 | Signing: the Signing Service issues TLS/MQTT certificates, never module signatures; add the mode × status table (invalid signature fails everywhere; untimestamped fails after expiry; self-signed only in Medium from the user trust store; unsigned only in Low); read `niagara.moduleVerificationMode` from `system.properties` — the guide's "default Low" is stale. Extends has Δ4 / bdps Δ4. Lint: no. | `build-verify.md` § Signing + `types/distribution.md:100` | `[ev: corpus B1332 §1332.2-3]` |
| Δ7 | Program objects/robots must be signed (Program Recompile) though enforcement is off; gradle profile RSA 3072; "Unable to load" startup causes (non-standard install, cert removed, expired+untimestamped); Jar Signer Tool for legacy jars. Lint: partial. | `types/logic-authoring.md:917` + BLD1 triage | `[ev: corpus B1332 rows 5-8]` |
| Δ8 | Correct `se` hosts: "Supervisor (Windows) only — a JACE never loads `-se`; keep JACE logic in `-rt`". Extends has Δ8. Lint: partial — WARN on `runtimeProfile=se` in a JACE-targeted module. | `types/structure.md:300` | `[ev: corpus B1341 §1341.2 C1]` |
| Δ9 | Add the runtime-profile and install rules: the Configure Runtime Profiles wizard decides whether a JACE has `-ux`/`-wb` jars (restarts the station); new module = hot install, upgrade/downgrade = station stop + host reboot; deps pulled in on install, block uninstall; Station Copier aborts on core/QNX needs. Lint: no. | `types/distribution.md` Software Manager § + `BUILD-LOOP.md:207` | `[ev: corpus B1341 rows 2-6]` |
| Δ10 | Budget `-rt` memory against fixed NRE pools (heap/meta/code cache via Configure NRE Memory, B1284); JACE alarm/history writes go to flash. Lint: no. | `types/math-control-libraries.md:43` | `[ev: corpus B1341 row 7]` |

## Lessons
- A boolean whose name reads one way (`noStation`) and acts the other needs the shipped manifests as the tie-breaker, not prose.
- "Upgrade" is a different operation per artifact (module, template, app template); the kit must say which state survives each.
- The same word ("signing") names two unrelated services; disambiguate in the kit.

---
**Status**: FOLDED 2026-10-23 (branch `docs/r8-u10-platform-templates`) — Δ1, Δ4 `types/distribution.md`; Δ5 `types/lexicon.md` (lint owed in ledger.d `2026-10-23-r8-u6-c54-c57.md`); Δ6, Δ7 `build-verify.md`; Δ8 `types/structure.md`; Δ2, Δ3 `types/logic-authoring.md` § Templates (Δ2 lint owed: G92 in ledger.d `2026-10-23-r8-u10-platform-templates.md`); Δ9, Δ10 new `types/platform.md` (landed there, not in `BUILD-LOOP.md:207` / `types/math-control-libraries.md:43`, which are outside the unit's surface; pointer lines owed).
