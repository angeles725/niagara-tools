<!-- review-status: folded -->
<!-- module: kit -->
<!-- deltas: 10 -->
# 2026-10-15 · kit · devguide415-r2-security-tags-analytics

**Session**: research-sdd focus `dev-capabilities-kit-gaps` round 2 read the StationSecurity, Fips, Tagging, Hierarchies, Relations, ExportTags and Analytics (Guide, API, DeveloperAnalytics) user guides plus haystack source; this retro carries corpus B1326, B1329, B1338 and B1336 (flags on rebuild). Complements `2026-10-15-devguide415-nav-subscriptions-tags-ords.md` (**nsto** Δ9 tag-dictionary recipe) and `…-horizons-authoring-security.md` (**has** Δ5 Security Manager card).
**Delta count**: 10

## What happened
(1) The kit has no FIPS guidance: a named JCA provider (`getInstance(alg, "BCFIPS")`) works in one mode only, Blowfish/MD5 survive only for legacy decrypt, FIPS needs 14-character secrets. (2) Least-privilege rules: ancestors get automatic operator-read; histories need the operator flag; `UserService` slots use a per-own-account scheme; 4.15 adds absolute logoff (7 days). (3) Tags: the tags licence caps dictionaries at 2 (`n` takes one), so a module-shipped dictionary can push Haystack into fault — the kit presents shipping one as free; implied `hs:equipRef` never reaches non-`BControlPoint` components or NullProxy points (our logic/setpoint points); indexes and hierarchy caches are heap-only. (4) Analytics: the developer guide documents only data-processing blocks (the kit's `AnalyticDataSource.Provider` feed is undocumented), licensing is per `a:a`-tagged point and the framework stops when exceeded, the ORD schemes are `analyticValue:`/`analyticRollup:`/`analyticTrend:`/`alerts:`.

## Evidence
- FIPS `[ev: niagara-help 4.15.6.20 guides-clean/Fips/DevelopersNotes-51FBF6BE.txt:29-41]` `[ev: …/Fips/UpdatingUserPasswordsFIPS_N4-1E65C4B9.txt:13]`; RBAC `[ev: …/StationSecurity/role_AncestorPermissions.txt:29-38]` `[ev: …/StationSecurity/role_HistoryPermissions.txt:12]` `[ev: …/StationSecurity/StationAutoLogoff-4CBC23A4.txt:38]` `[ev: corpus B1326]`; kit grep `rg -i fips` outside retros → 3 fixture palette lines
- Tags `[ev: niagara-help 4.15.6.20 guides-clean/Tagging/LicenseRequirements-474FC1B4.txt:20]` `[ev: code organized/haystack/haystack-rt/decompiled/com/tridium/haystack/BContainmentRelation.java:65]` vs `[ev: code types/logic.md:359-366]` `[ev: corpus B1329]`
- Analytics `[ev: niagara-help 4.15.6.20 guides-clean/DeveloperAnalytics/buildingAlgorithmBlocks.txt:9,23]` `[ev: …/DeveloperAnalytics/ordSchemes.txt:15,25]` `[ev: …/AnalyticsGuide/AboutLicensing-2CF33B35.txt:14]` vs `[ev: code types/logic-authoring.md:31]` `[ev: corpus B1338]`

## Proposed kit deltas (propose-never-apply)
| Δ | Delta | Target file / § | Token |
|---|---|---|---|
| Δ1 | Add "FIPS-mode constraints": never name a provider in `getInstance`; no MD5/Blowfish/RC4/DES for new data; 14-char passwords/passphrases; plan the FIPS migration of machine users. Lint: yes — WARN on two-argument `getInstance(..., "<provider>")`; extend the MD5 lint. | `types/security.md` (new §) | `[ev: corpus B1326 rows 1-3]` |
| Δ2 | Least privilege: document the category + role a module's user needs; ancestors are automatic operator-read; history exts tested with an operator user (operator flag + `r`); self-service password needs `rw` on `UserService.Password`. Lint: no. | `types/security.md` §2 | `[ev: corpus B1326 rows 4-7]` |
| Δ3 | 4.15 absolute logoff (default 7 days) and session invalidation can end an HMI session — kiosk users get their own logoff settings or re-authenticate on 401. Lint: no. | `types/security.md` §3 + `types/dashboard.md` | `[ev: corpus B1326 row 8]` |
| Δ4 | TLS: use platform stores (User Key Store, Trust Store, Allowed Hosts, CertManagerService); never embed a keystore/truststore; failures show as the connection's last failure cause. Lint: partial — grep `KeyStore.getInstance`, `TrustManager`. | `types/security.md` §10 | `[ev: corpus B1326 row 9]` |
| Δ5 | Tag dictionary licence: 2 dictionaries by default (`Dictionary.limit`), `n` takes one; document the licence need or prefer direct tags / `hs` rules over shipping a dictionary. Lint: partial. | `types/logic.md` § Ship a tag dictionary + `types/cloud-connector.md:206-217` | `[ev: corpus B1329 §1329.2 C1]` |
| Δ6 | Equipment tagging: `hs:equip` + `hs:siteRef` on the equip; DIRECT `hs:equipRef` on NullProxy logic/setpoint points and non-point components (implied never reaches them). Lint: partial. | `types/logic.md` tag recipe | `[ev: corpus B1329 rows 2-3]` |
| Δ7 | Tag rules and hierarchies: `ScopedTagRule` over `HasAncestor`; every rule item needs a dictionary definition; Implied Tags Index off by default; indexes and hierarchy caches heap-only (recreate after edits); tags case-sensitive; no markers in `GroupLevelDef`; Spy "Tags Implied" to verify. Reinforces nsto Δ9. Lint: no. | `types/logic.md` + `types/logic-authoring.md:181` | `[ev: corpus B1329 rows 4-8]` |
| Δ8 | ExportTags `ComponentTag` makes a non-proxy COPY on the Supervisor (source edits do not propagate); prefer proxy points/oBIX/cloud for live data. Lint: no. | `types/distribution.md` | `[ev: corpus B1329 row 9]` |
| Δ9 | Analytics: mark `AnalyticDataSource.Provider` as undocumented (only data-processing blocks are documented); consume results through an analytic proxy ext; licence per `a:a` point (framework stops when exceeded; turn `AutoTagAnalyticPoint` off); data definition per tag; schemes `analyticValue:`/`analyticRollup:`/`analyticTrend:`/`alerts:` (no `analytic:`, polled); alerts default to no alarm. Lint: yes — grep `analytic:`. | `types/logic-authoring.md:31`, `:101` | `[ev: corpus B1338]` |
| Δ10 | Preserve author-set slot flags (hidden/operator) when a refresh action rebuilds slots — Tridium's proxy `forceUpdate` resets them to defaults. Lint: no. | slot-flags guidance (`types/value-types.md` or `types/logic-authoring.md`) | `[ev: corpus B1336 row 8]` |

## Lessons
- Licences shape module design (tag dictionaries, analytics points, MQTT devices); the kit should list each licence a module depends on.
- Implied semantics (tags, relations) have type and proxy preconditions that silently exclude our own logic points.
- A crypto call that names a provider is a FIPS-mode portability bug, not a style choice.

---
**Status**: FOLDED 2026-10-23 (branch `docs/r8-u8-security-tags-analytics`) — Δ1 `types/security.md` § 9.1; Δ2 § 2.6; Δ3 `types/dashboard.md` § Session lifetime (C31, earlier); Δ4 `types/security.md` § 10.1; Δ5 and Δ7 `types/logic.md` § Ship a tag dictionary (C21, earlier); Δ6 `types/logic.md` § Our-modules tag recipe; Δ8 `types/logic.md` § Ship a tag dictionary (target `types/distribution.md` was outside the unit surface); Δ9 `types/logic-authoring.md` analytics bullet + § ORD scheme typology; Δ10 `types/logic-authoring.md` § Programmatic slot-name rules, scope corrected (forceUpdate = R2 oBIX only, B1353). Owed Go lints G80 `jca-named-provider` (Δ1), G81 `embedded-keystore` (Δ4), G82 `analytic-ord-scheme` (Δ9) in `ledger.d/2026-10-23-r8-u8-sec-tags-analytics.md`. Δ1 dropped "RC4/DES" (not in the FIPS guide; covered by "any other non-FIPS algorithm throws"); Δ9 "data definition per tag" not folded (no source line re-verified).
