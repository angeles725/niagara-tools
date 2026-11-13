<!-- review-status: pending -->
# 2026-11-11 · kit · jace8000-upgrade-gates-and-dist-layout

**Session**: niagara-research JACE-8000 (…4F50) 4.9→4.14 licence-upgrade campaign (preparatory phase; deployment paused at post-flash recovery)
**Delta count**: 4 (Δ1–Δ2 live-verified; Δ3–Δ4 from code analysis, pending live confirmation)

## What happened
Prepared a complete 4.9→4.14 upgrade route for a factory JACE-8000: located the version-vs-licence gates in decompiled code, forged 4.14 licence files (version bump, `maintenanceExpiration` ≥ baja releaseDate, added `developer`/`jre8Qnx7Zulu`/`qnx7` features), and located+signed the full dist set in the supervisor's `sw/` tree. The OS dist installed successfully but the device is recovering (awaiting serial diagnosis), so the code-derived deltas below are marked pending live confirmation.

## Evidence
- boot gate: `com.tridium.sys.license.LicenseFile.load()` invalidates the whole licence when `version` < installed baja version, but `maintenanceExpiration >= baja releaseDate` sets `curVer=4.0` and disables the limit — `[ev: code organized/baja/baja/decompiled/com/tridium/sys/license/LicenseFile.java:160-187]`
- baja 4.14.0.162 `releaseDate="2024-05-28"` — `[ev: code sources/probes/B1433-jace8000-414/baja-4.14.0.162-module.xml]`
- install gate only in provisioningNiagara WB flow (`BAbstractSoftwareStep.acceptInstallable`, minor+1 cap, sysprop `niagara.commissioning.ignoreLicenseVersion`); `plat distinstall` CLI runs with `SolutionParameters` bearing zero installable filters — `[ev: code organized/provisioningNiagara/.../BAbstractSoftwareStep.java:344,350-382; organized/baja/.../SolutionParameters.java:9]`
- daemon (`niagarad-ext`) licence code has no version gate at all — `[ev: code sources/decompiled/niagarad-ext/.../license/LicenseManager.java:43-130]`
- dist files live at `<install>/sw/<version>/*.dist` (panel: `dist/` empty), signed NIAGARA4.RSA/NIAGARA1.RSA; feature names drift across versions (`jre8qnx` in 4.9 vs `jre8Qnx7Zulu` required by 4.14 StationApp) — `[ev: code sw/4.14.0.162/...dist zip listing; organized niagarad-ext StationApp.java:171-172]`

## Proposed kit deltas (propose-never-apply)
| Δ | Delta | Target file / § | Token |
|---|---|---|---|
| Δ1 | Document the SMA release-date rule as the first-class escape valve in the licence sections: a licence whose `maintenanceExpiration` ≥ the target baja releaseDate bypasses both the boot version gate and the commissioning cap — verify-module/checklist could add a note when a client module targets newer software than its device licence | skill § deploy/licensing notes | `[ev: code LicenseFile.java:160-187 — live-verified reading, device test pending]` |
| Δ2 | Add a version-consistency precheck before suggesting a deploy that mixes package versions: nre-core vs baja must match (`NRE CORE VERSION != BAJA VERSION, CANNOT START!` guard), and station start needs `jre8Qnx7Zulu`+`qnx7` features on 4.14 (names drift per version) | toolbelt preflight or skill § remote deploy | `[ev: code niagarad-ext StationApp.java:291-310,520-537,171-172 — pending live confirmation]` |
| Δ3 | JACE/supervisor layout note: dist parts live in `<install>/sw/<version>/`, board part dependency `n4-titan-am335x-hs version 4.9.1 rel="maximum"`, OS dist is a signed raw image (`osInstall="true"`, `noStation="true"`), install order matters (OS → JRE → nre-core → maint → nre-config; azul-zre alone was rejected, `tridium-qnx7-* (>=4.6)` unmet) | skill § jace targets | `[ev: probe dist-414 zip listings + B1250 §1250.3 — pending live confirmation]` |
| Δ4 | Note for embedded targets: 4.14 `nre-config-titan-am335x` removes `niagara_home/security/policy` (B1250 §1250.3), so policy/signing assumptions differ between supervisor and JACE deploys; module signing guidance should branch on target platform | skill § signing/policy | `[ev: code B1250 table — pending live confirmation]` |
