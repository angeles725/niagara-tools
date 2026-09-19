# build-n4-module kit — Master improvement register (2026-09-19)

**What this is.** A consolidated, de-duplicated register of NET-NEW retro/delta candidates and structural gaps for the `build-n4-module` kit, produced from a multi-source research sweep (project memories, corpus B1–B1028, official Tridium devguide, real decompiled modules in `organized/`, the nmodsreflow module, and net-new vendor module sets Distech EC-Net 4.3 + PowerB 4.15). Nothing here is folded yet — every item is a proposal. Integration per item remains a user decision.

**Baseline.** As of the 2026-09-18 fold campaign the kit had **0 pending / 131 folded retros** and 10 `types/` docs. All items below are ADDITIONAL to that.

**Product scope (user-confirmed 2026-09-19).** Audience = internal + OEM. Module types = logic rt, dashboard ux, driver/network, service/tool (all). Deploy = Supervisor + JACE/-se. Security first-class = audit who-changed-what, RBAC+CSRF+secrets, licensing own modules, anti-injection.

---

## Summary — candidates by destination kit artifact

| Destination in kit | Type | Candidates | Priority |
|---|---|---|---|
| NEW `types/actions.md` | new doc | AC1–AC7 | HIGH |
| NEW `types/driver-authoring.md` | new doc | D1–D8, code-anatomy G3 (BProxyExt) | HIGH |
| NEW `types/security.md` + lints | new doc + 3 lints | SEC-01…SEC-11 | HIGH |
| NEW `types/module-wiring.md` | new doc + 1 lint | WIR-G1…G7 | HIGH |
| NEW `types/observability.md` + lint | new doc + `no-System.out` lint | OBS-1…OBS-7 | MED-HIGH |
| NEW `types/distribution.md` + build-verify | new doc | KD-1…KD-8 | MED-HIGH (OEM) |
| NEW `types/value-types.md` | new doc | TYP-G1…G8, anat G4, exemplar C5 | MED |
| NEW `types/theme.md`, `types/utility-lib.md`, `types/cloud-connector.md` | new docs | TH-01…03, UL-01/02, DB-01, CL-01…04 | MED (OEM) |
| Expand `types/moduleTest.md` | expand | TEST-G1…G8, R6 | MED-HIGH |
| Expand `types/logic-authoring.md` / `logic.md` | expand | anat G1/G2/G5/G6/G7, exemplars C1–C8, R2 | MED |
| Expand `types/structure.md` (-se, theme, lexicon) | expand | SE-C1…SE-C7 | HIGH (JACE) |
| Expand `types/issues-and-gotchas.md` | expand | R1, R2, R6, EC-Net-browser, GET-destructive | HIGH |
| Kit structure / tooling / SKILL.md | meta | AXIS-1…AXIS-8 + top-5 | HIGH |

Proposed NEW automated checks: `null-context-write`, `bql-string-concat`, `arbitrary-ord-input`, `no-System.out`, `agent-on-target-exists`, `se-must-not-use-display`, `uberjar-vs-api-dep-conflict`, `jasmine-ux-tests-present`.

---

## Part 1 — NEW `types/actions.md` (from actions lane AC1–AC7)

Kit today only mentions `@NiagaraAction` in scattered lines; no dedicated doc.

| ID | Topic | Evidence | Notes |
|---|---|---|---|
| AC1 | ASYNC threading model: normal actions run on the caller thread; `ASYNC` coalesces onto the shared engine thread → blocking there starves all timers | devguide `execution.txt`; `flags=Flags.ASYNC` on escalateAlarms/cleanup | REAL |
| AC2 | `doX()` dispatch contract: Baja calls `do`+Capitalize(name), plus a `(arg, Context)` overload; wrong name = silent no-op; Context overload gives invoking user | slot-o-matic `getMethod("do"+cap, ...)`; ackAlarm→doAckAlarm | REAL |
| AC3 | `Flags.CONFIRM_REQUIRED` (0x80) confirmation dialog — ideal for `faultReset`/destructive actions | Flags.java; resetStatistics/changeDeviceUuid | REAL, zero coverage |
| AC4 | `parameterType`/`returnType`/`defaultValue` on `@NiagaraAction`; missing `defaultValue` = slotomatic error; BStruct param → WB dialog | slot-o-matic.txt; ackAlarm(BAlarmRecord) | REAL |
| AC5 | Actions vs `@NiagaraTopic` (event-source, no storage; fire via `component.fire(topic,val)`); pick topic for many-listeners | execution.txt; B538; report-rt BReportSource | REAL — merges exemplar C1/C2 + anat G2 |
| AC6 | `Flags.NO_AUDIT` (0x800) on internal callback actions so they don't flood the audit trail | Flags.java; escalateAlarms/expire uses | REAL |
| AC7 | `@NiagaraRpc` vs `@NiagaraAction` boundary (bajaux `/rpc`, CSRF-gated vs oBIX/Fox) | B507 | merges with serving-recipes R7 |

## Part 1b — NEW `types/driver-authoring.md` (drivers D1–D8 + anat G3)

Kit covers driver factory/discovery (folded) but has no comm-layer / health / point-SPI doc.

| ID | Topic | Evidence |
|---|---|---|
| D1 | `BTuningPolicy` map authoring (tuningPolicies/defaultPolicy, slots, per-point `tuningPolicyName`) | B872, B927, driverPoint.txt |
| D2 | `BBasicNetwork` comm layer: `makeComm()`, CommTransactionManager/Receiver/Transmitter, 3 worker queues | B517, basicDriver.txt |
| D3 | `doPing()` override + `configFail`/`configOk`/`configFatal` (fatal = no auto-recover) | B7, B810, driverFramework.txt |
| D4 | `BIPollable` + poll-bucket register on readSubscribed/readUnsubscribed; dibs first-poll | B872, B955, driverPoint.txt |
| D5 | `BHistoryDeviceExt` + `BHistoryPollScheduler` (pull field-device history) | B872, B33 |
| D6 | `BScheduleDeviceExt` makeExport/makeImportExt SPI | B927, B7 |
| D7 | `BVirtualComponent`/`BVirtualGateway` on-demand transient points (`virtual:` scheme) | B28, virtualComponents.txt |
| D8 | `BSerialNetwork`/`SerialComm` RS-485 pattern (BSerialHelper, interMessageDelay) | B517, B500 |
| anat-G3 | `BProxyExt` point SPI: `readSubscribed`/`readUnsubscribed`/`write` three overrides | BProxyExt.java |

## Part 1c — NEW `types/security.md` + lints (security SEC-01…SEC-11)

| ID | Concern | Topic | Kit artifact |
|---|---|---|---|
| SEC-01 | Audit | Null-Context write bypass: `set(prop,val,null)` skips audit AND grants `BPermissions.all` | NEW lint `null-context-write` + doc |
| SEC-02 | Audit | oBIX PUT attributes writes to a shared machine user (PANCCADIA gap) | doc §oBIX-attribution |
| SEC-03 | Audit | `BasicContext(user)` chain to attribute servlet writes (SessionManager→BUser→BasicContext) | doc |
| SEC-04 | CSRF | Replace X-Requested-With with `CsrfProtectedFilter` + `x-niagara-csrfToken` + `csrfUtil` | doc (deepens R11) |
| SEC-05 | RBAC | `BPermissions` 6-bit table + `Flags.isOperator` slot tier; ADMIN_INVOKE for destructive | doc |
| SEC-06 | Secrets | `BPassword`: never log `getValue()`; `doPrivileged`/`SecretChars`; toString masks | doc |
| SEC-07 | Licensing | `BILicensed.getLicenseFeature()` + `Sys.getLicenseManager().checkFeature()` + `check()` in serviceStarted | may host in `types/licensing.md` |
| SEC-08 | Licensing | HTTP phone-home anti-pattern (Reflow: plain HTTP + hostId leak) | doc anti-pattern |
| SEC-09 | Injection | BQL string-concat; use `BqlQuery.toBqlLiteral()`+`SlotPath.escape()` | NEW lint `bql-string-concat` + doc |
| SEC-10 | Injection | Arbitrary `BOrd.make(userInput)` from client (Reflow BQL gateway) | NEW lint `arbitrary-ord-input` |
| SEC-11 | Injection | `^` root file enumeration without permission check (Reflow FileTree/ImageList) | doc anti-pattern |

## Part 1d — NEW `types/module-wiring.md` + lint (WIR-G1…G7)

| ID | Topic | Evidence |
|---|---|---|
| WIR-G1 | `uberjar()` first-party bundling (kit teaches Shadow; uberjar is simpler, built-in) | nmodsreflow gradle.kts |
| WIR-G2 | Gradle dep table: `nre`/`api`/`compileOnly`/`uberjar`/`moduleTestImplementation` → what each emits in module.xml | generated module.xml |
| WIR-G3 | Sibling-part dep `api(project(":mod-rt"))` → OEM vendor/version stamped | nmodsreflow-ux module.xml |
| WIR-G4 | `compileOnly(files($niagara_home/bin/ext/...))` for Niagara-private jars (jetty compact3) | gradle.kts |
| WIR-G5 | XML-first `<agent><on type="mod:Type"/>` registration without `@AgentOn` | module-include.xml |
| WIR-G6 | NEW lint `agent-on-target-exists`: every `<on type>` resolves to a real registered type | — |
| WIR-G7 | `uberjar` + `api()` double-declaration conflict note | classloader model |

## Part 1e — NEW `types/observability.md` + lint (OBS-1…OBS-7)

| ID | Topic | Evidence |
|---|---|---|
| OBS-1 | Full observability guide (logging + spy + fault + audit-vs-log) | multiple |
| OBS-2 | NEW lint `no-System.out` (bypasses BLogHistoryService) | B92/B106 |
| OBS-3 | Spy-page registration `Spy.ROOT.add(...)` + SpyDir/SpyWriter recipe | Nre.java, BModbusNetwork |
| OBS-4 | `appFail`/`configFail(msg)` fault surfacing + `faultCause` | BDevice.java |
| OBS-5 | Audit-vs-log decision table (audit is synchronous; only user-initiated servlet writes) | B167, B30/B31 |
| OBS-6 | Logger-name convention (module short name, not FQCN) | B20, B324 |
| OBS-7 | `BLogHistoryService` as BQL-alarmable sink | B33 |
| NOTE | `java.util.logging` is preferred; `javax.baja.log.Log` is `@Deprecated` | Log.txt |

## Part 1f — NEW `types/distribution.md` + build-verify (KD-1…KD-8, OEM)

| ID | Topic | Evidence |
|---|---|---|
| KD-1 | Full BOG schema-safety matrix (ADD/REORDER/RETYPE/RENAME/REMOVE → SAFE/LOSSY/OUTAGE) | B754 |
| KD-2 | OEM vendor trust cert deployment for production (Tridium XML, DSA; not X.509) | B1027, B18 |
| KD-3 | OEM overlay anatomy + step-11 last-write-wins | B1026 |
| KD-4 | `dist.xml` metadata (@noStation/@reboot/@osInstall/<provides>) | B1023 |
| KD-5 | Station template `.ntpl` as OEM commissioning vehicle | B1021, B1024 |
| KD-6 | Supervisor-as-installer (push .dist to JACE/Atlas) | B1023 |
| KD-7 | AX→N4 migration: N4 module must exist before migrate | B1024, B405 |
| KD-8 | Dependency floor (3-part) vs 4-part build-stamp as named gate | B755 |

## Part 1g — NEW `types/value-types.md` (type-system TYP-G1…G8 + anat G4 + exemplar C5)

| ID | Topic | Evidence |
|---|---|---|
| TYP-G1 | `@NiagaraEnum(range={@Range(value,ordinal)})` + `getRange().get()` make() (kit's switch form diverges) | BAaPhpDataTypesEnum |
| TYP-G2 | `BDynamicEnum` + `BEnumRange.make(int[],String[])` at discovery; attach via `BFacets.makeEnum` | AaPhpAttributeConversion |
| TYP-G3 | `BFacets.makeEnum` + merge `make(a,b)` + `makeRemove` | BEnumPoint, BBacnetActionCommand |
| TYP-G4 | `makeBoolean(trueText,falseText)` + `FIELD_EDITOR` key | BAaPhpSendFileParam |
| TYP-G5 | `BSimple.intern()` in make() for identity dedup | BHistoryId |
| TYP-G6 | `BUnit.convertTo` + `BUnit.make` custom units | B745 |
| TYP-G7 | BStruct as action param + discovery learn-entry bag examples | BAddInputSlotArgs, B955 |
| TYP-G8 | `BFacets.DEFAULT` sentinel + 5-arg dynamic `add(...,facets,cx)` | B4, B1007 |

## Part 1h — Expand `types/moduleTest.md` (testing TEST-G1…G8 + R6)

| ID | Topic |
|---|---|
| TEST-G1 | `BTestNgStation` full-services fixture (Role/User/Alarm/History/Fox/Web) |
| TEST-G2 | `TestHelper.waitFor` / `assertWillBeTrue` (async, avoid Thread.sleep) |
| TEST-G3 | `BTridiumTestNg.toDataProviderArray` + `DataProviderResults` |
| TEST-G4 | `NRetryAnalyzer` (flaky-test retry) |
| TEST-G5 | `@Requires(os=...)` OS-conditional skip |
| TEST-G6 | `StationRunner` out-of-process integration tests |
| TEST-G7 | `TestHelper` private-field reflection / runWithAlternateValue |
| TEST-G8 | `loadPaletteItem`/`loadPaletteFor` palette assertions |
| R6 | moduleTest plugin 7.6.17 bug + ux modules lack Jasmine |

## Part 1i — Expand `types/logic-authoring.md` / `logic.md` (code-anatomy + exemplars + R2)

| ID | Topic |
|---|---|
| anat-G1 | `doXxx` dispatch general law (not one example) |
| anat-G2 | `@NiagaraTopic` + `fire()` in a plain BComponent |
| anat-G5 | Logging pattern (Log vs Logger, package-level static helper) |
| anat-G6 | `BSingleton`+`@NiagaraSingleton`+`@AgentOn` general agent pattern |
| anat-G7 | `changed()`/`added()`/`removed()` guard contract (super order + isRunning + try/catch) |
| C1/C2 | `@NiagaraTopic` + BReportSource/BReportRecipient scheduled pipeline |
| C3 | `BProgram`/`BRobotCode`/`ProgramBase` scripting SPI + `BProgramService.runRobot` |
| C4 | `BBatchRoutine` mass-edit SPI (targets BOrdList, run(component)) |
| C6 | `BEmailService.send(BEmail)` wiring + BOutgoingAccount SMTP |
| C7 | `BIRestrictedComponent` placement guard |
| C7b | Abstract provider child inside a BAbstractService + CoalesceQueue |
| C8 | `BSimpleJob` + Fox file-channel streaming (BFileChannel/BFoxFileStore) |
| R2 | rt lifecycle seam `cancelRunTickets` vs `cancelTicket` (untestable → readback) |

## Part 1j — Expand `types/structure.md` (-se profile SE-C1…SE-C7)

| ID | Topic |
|---|---|
| SE-C1 | `-se` = Java SE dependency (AWT/Swing/JDBC); loads on JACE via `-rp:rt,se` |
| SE-C2 | JACE headless AWT trap: display classes in `-se` fail at runtime (NEW lint `se-must-not-use-display`) |
| SE-C3 | JACE install mechanics: stop-all, overwrite-in-place, no backup/rollback, NIAGARA_USER_HOME fallback |
| SE-C4 | `-se → all` dep trap: wb deps compile but ClassNotFound on headless daemon |
| SE-C5 | Multi-part module: adding an `-se` sibling |
| SE-C6 | JACE resource constraints (1 GB, QNX ARM, no heavy DB/JNI) |
| SE-C7 | `runtimeProfile.set(se)` gradle + `niagara-module.xml` runtimeProfiles |
| — | Also: theme module (`types/theme.md`) and lexicon-only/`-doc` profiles are uncovered types |

## Part 1k — Tier-A field-proven gotchas → `issues-and-gotchas.md` / lints

| ID | Topic |
|---|---|
| R1 | `Clock.schedule` rejects `time<=0`; floor at 1 ms (NEW lint rule in `lint-timers.sh`) |
| R5 | Post-deploy class-resolution gate (`SEVERE Missing class`) |
| EC-Net | EC-Net 4.3/Hx browser `set(value)` fails → HOA BooleanWritable pattern |
| GET-destructive | Reflow: destructive operations via HTTP GET + CSRF `<img>` vector |

## Part 1l — NEW vendor-class docs (from vendor-module mining: Distech EC-Net + PowerB)

Net-new module classes the kit does not cover, mined from real jars via javap + XML resources.

### NEW `types/theme.md` (Distech `themeDistech-ux`)
| ID | Topic | Evidence |
|---|---|---|
| TH-01 | Theme module = zero-Java ux jar; sole registration is `<defs><def name='themeName' value='...'/></defs>`; platform scans ux modules for it | themeDistech-ux module.xml |
| TH-02 | `imageOverrides/` mirrors each module's Java package path to swap icons per type context (833 files) | imageOverrides/bacnet/com/tridium/... |
| TH-03 | NSS syntax (`#define`, `lineargradient(stop()... angle())`) + ux/hx/fx theme.css + fonts; gotcha: hx css copy/paste font-URL bug | nss/theme.nss |

### NEW `types/cloud-connector.md` (PowerB `cloudLink*`)
| ID | Topic | Evidence |
|---|---|---|
| CL-01 | `BCloudConnectionService extends BAbstractService` + 3 static SPI factory maps (msg handler / channel config / file upload) | BCloudConnectionService |
| CL-02 | Transport dual model: `BHttpTransport` (okhttp, from platform) vs `BAmqpTransport` (qpid-proton 534 cls fat-jarred); store-and-forward queue, GZIP, retries | BAmqpTransport |
| CL-03 | Auth SPI `BAbstractClientAuthenticator` + `BUsernameAndPassword`/token; credentials in platform **KeyRing** (not component props) + per-backend `KeyRingPermission` | auth/*, cloudLinkAzure module.xml |
| CL-04 | Backend plugin pattern: per-backend HandlerFactory + ChannelConfigFactory registered into core maps; dep chain core→Azure→Forge→{Ncs,HonSbp}; `nc` SmartTagDictionary | cloudLinkForge/Ncs/HonSbp module.xml |

### NEW `types/utility-lib.md` (PowerB `clUtils*`, `jtds`)
| ID | Topic | Evidence |
|---|---|---|
| UL-01 | Utility-lib: registered types are agents/abstracts (`<agent><on type='driver:Device'/>`), NO palette; static `registerHelper()` self-registration | clUtils-rt module.xml, BDeviceInfoHelper |
| UL-02 | Protocol-split convention `<base>-rt` → `<base><Protocol>-rt`; base carries no driver dep, each specialization narrows the agent `<on type>` | clUtilsBacnet/Niagara |
| DB-01 | Pure library bundle (jtds/JDBC): empty `<types/>`, `autoload='true'`, only `baja` dep, `getClassLoader` + socket permissions; consumers `Class.forName(...)` | jtds-rt module.xml |

### `types/driver-authoring.md` additions (KNX/Z-Wave twists)
| ID | Topic | Evidence |
|---|---|---|
| DR-01 | KNX: `BEibnetIpNetwork extends BLoadableNetwork`; addressing by `groupAddresses` (not object/property); LinkLayer+TunnelLayer; ETS import; per-datatype ProxyExt family | eibnetIp-rt |
| DR-02 | Z-Wave: `BZWaveNetwork extends BBasicNetwork`; inclusion lifecycle as persistent `BInclusionMonitor` BComponent; command-class capability tree; firmware `.hex` + device templates bundled in jar | zwave-rt |

---

## Part 2 — Proposed NEW automated checks/lints (net-new)

1. `null-context-write` (SEC-01) — flag `set(slot,val,null)`/`invoke(action,val,null)` outside framework-internal sites.
2. `bql-string-concat` (SEC-09) — flag `BqlQuery.make("..."+x)` / `BOrd.make("...bql:"+x)` without `SlotPath.escape`.
3. `arbitrary-ord-input` (SEC-10) — flag `BOrd.make(clientInput)` without allowlist prefix.
4. `no-System.out` (OBS-2) — flag `System.out.print*` in `*.java`.
5. `agent-on-target-exists` (WIR-G6) — every `<on type="mod:Type">` resolves.
6. `se-must-not-use-display` (SE-C2) — `-se` source importing `JFrame/JDialog`-class display code.
7. `uberjar-vs-api-dep-conflict` (WIR-G7) — a lib both `api()`'d and `uberjar()`'d.
8. `jasmine-ux-tests-present` (AXIS-7) — `-ux` with `rc/` but no `srcTest/rc/spec/` → WARN.
9. `clock-schedule-zero-floor` (R1) — timer scheduled with a `<=0` computed delay.

---

## Part 3 — Path to a "perfect" kit (structural audit, 8 axes)

- **AXIS-1 Module-type coverage**: missing `types/driver.md`, multi-profile `-rt+-ux` scaffold/fixture, `-se`, theme, lexicon-only, service-module route in SKILL decision table.
- **AXIS-2 Lifecycle**: commissioning-verify tooling is the kit's own named #1 gap (`BUILD-LOOP §6.b`) — no live facade↔rt / config-sanity auditor.
- **AXIS-3 Checks**: gotchas without checks (grant-all permissions, class-resolution, timer zero-floor, actions, jasmine).
- **AXIS-4 Exemplars/fixtures**: no `-ux`/`-wb` fixture; exemplars live in client repos, not the kit; no canonical excerpt.
- **AXIS-5 Skill quality**: installed `SKILL.md` is desynced from kit (missing `orient-guard.sh` step); no `model:` frontmatter; narrow trigger (won't fire for driver/service/tool).
- **AXIS-6 Onboarding**: no quick-start decision tree; README doesn't list toolbelt; corpus-index stops at B1028.
- **AXIS-7 Testing**: no Jasmine ux enforcement; moduleTest WARN-only; bats gaps (bog-audit, Wave-3 lints).
- **AXIS-8 CI/verification**: no `.github/workflows` CI; Wave-3 lints missing `# Mutation:` guard pins; sweep doesn't verify module_root paths.

**Top-5 to reach near-perfect:** (1) commissioning-verify tool, (2) multi-profile scaffold + fixture, (3) `types/driver-authoring.md`, (4) SKILL sync gate + trigger expansion, (5) Jasmine enforcement + Wave-3 guard pins.

---

## Part 4 — Recommended execution order

1. **Tier A field-proven** (R1, R2, R5, R6, SEC-01/03/09/10/11, EC-Net) — highest recurrence risk; several are automatable checks.
2. **New core docs** (`actions.md`, `driver-authoring.md`, `security.md`, `module-wiring.md`) — biggest coverage gaps for the confirmed module types.
3. **-se + distribution** (SE-C1…C7, KD-1…8) — required by Supervisor+JACE+OEM scope.
4. **observability + value-types + moduleTest expansion**.
5. **Structural/kit-perfect** (commissioning-verify tool, scaffold, SKILL sync, CI).
6. **Vendor-class docs** (`theme.md`, `utility-lib.md`, `cloud-connector.md`) — pending the vendor-module mining lane.

---

## Part 5 — Separate deliverables

- **nmodsreflow documentation draft** — full `-rt`/`-ux`/contract reference + 7 patterns + 7 gotchas + delta vs B50 (from the nmodsreflow lane). Candidate to become a corpus block or kit reference; held for review.
- **Vendor net-new corpus** — Distech EC-Net + PowerB net-new modules brought into `organized/` with vineflower + CFR + procyon (best-of-3) + Maven `-sources.jar` for embedded OSS. Substantive uniques mined: cloudLink* (cloud connectors → CL-01…04), themeDistech (theme → TH-01…03), clUtils*/jtds (utility libs → UL-01/02, DB-01), eibnetIp/zwave (KNX/Z-Wave → DR-01/02). Candidates folded into Part 1l above.

---

## Source lanes (for traceability)

memories · corpus B1–B760 · docs+niagara-help · actions · drivers · exemplars · kit/skill audit · code-anatomy · module-wiring · testing · type-system · nmodsreflow · -se/controller · security · OEM-distribution · observability · vendor-modules (in progress).
