# mcp-n4 METHODOLOGY

How an agent reads and writes a Niagara N4 station through the `mcp-n4` server without
hurting it. Each rule names the code or test that enforces it, or is marked
**manual** (the agent and the human must do it). Evidence lives in
niagara-research blocks (section 6); this file does not restate it.

Paths below are relative to `mcp-n4-kit/`. Test names are in `tests/`.

## 1. Route ladder

Never answer "cannot". When a route is blocked, state the next one, its cost and
what it needs.

| Route | Use | Status |
|---|---|---|
| R-A BOX JSON | Default. Read, create, set, link, invoke, remove, save with no module on the station. | Implemented (`go/internal/mcpn4/box` + tools; the Python `mcp_n4/box.py` until S17). Live-certified in B1199. |
| R-B Java Fox sidecar | Fidelity cross-check, or fallback when BOX is disabled. Needs JRE 8 and the install jars. | Not built (B1197 phase P5). |
| R-F offline BOG-XML | Zero-risk dry run and diff against a `config.bog` snapshot: the niagara-research Go commands `bognav` and `bog-write`. | External tools. **manual** |
| R-E oBIX | Values only (read, write a value, invoke). No structure editing. | External. **manual** |
| R-D in-station module | Last resort, only where BOX is blocked and a module is acceptable. It adds attack surface. | Not built. |

Order of preference: R-A, then R-F to rehearse or diff, then R-B, R-E, R-D.
Rule enforced by: **manual**.

## 2. Safety layers L1-L8 (B1197 section 1197.4)

| Layer | Rule | Enforced by |
|---|---|---|
| L1 identity | One dedicated station user for the agent; operator-level permissions on the target categories, never the super user. The server refuses a write unless the station's real `stationName` matched `expected_station`. | User: **manual**. Name check: the `writeops`/`plan` identity gate; `plan.TestWriteOnAnUnverifiedSessionIsRefusedAndAudited`, `session.TestExpectedStationMismatchClosesSessionAndNamesBoth` |
| L2 server-side enforcement | The station applies the user's permissions, slot limits and audit; the client never bypasses them. | The station. A 401/403 raises `AuthError`: `box.TestReplayRecordedFrames/http_403_is_auth_error` |
| L3 dry run | `dry_run` defaults to true: the reply is the exact BOX ops and their inverse; nothing reaches the station. | `tools_write._process`; `blackbox.TestDryRunIsTheDefaultAndSendsNoMutation` |
| L4 confirmation token | Executing needs the single-use HMAC token from the dry run, bound to tool, arguments, plan hash and expiry; the plan names the station and the connect session, so a token never executes after a reconnect. | `safety.ConfirmationTokens`; `blackbox.TestATokenFromStationACannotExecuteOnStationB`, `blackbox.TestATokenIsSingleUse`, `blackbox.TestArgsChangedAfterTheTokenWasIssuedAreRefused`, `blackbox.TestStationStateChangedSinceTheDryRunIsRefused` |
| L5 write scope and limits | Every ORD must be a plain slot ORD (no `|` hop after `station:|slot:`) under a `--write-scope` prefix (slot boundaries); no prefix means no write; a create may make the scope root itself under its parent unless it already exists (the same scope can roll that create back while nothing was written inside it, but cannot remove the root with n4_remove_component: a wider scope or a manual step does); a link is scoped on its target only (the source must still be plain), and its refusal names the end; at most `--max-writes` per server process (a batch is one write, `n4_replicate_points` one per add and `n4_rollback` one per re-created component and link; a reconnect does not reset it; `n4_save_station` is outside the budget and every write reply reports `write_budget`); `n4_invoke_action` allows only `set`, `active`, `inactive`, `auto`. | `safety.WriteScope`; `ord.TestAChainedOrdAfterTheSlotPathIsRefused`, `blackbox.TestNoScopeConfiguredRefusesEveryWrite`, `blackbox.TestPrefixBoundaryIsRespected`, `blackbox.TestScopeRootCanBeCreatedUnderItsParent`, `blackbox.TestScopeRootThatExistsIsNotCreatedAgain`, `blackbox.TestLinkFromASourceOutsideTheScopeIsPlanned`, `blackbox.TestLinkRefusalNamesTheTargetEnd`, `blackbox.TestWritesBeyondTheSessionBudgetAreRefused`, `blackbox.TestReconnectingKeepsTheProcessBudget`, `blackbox.TestSaveRunsAfterTheBudgetIsSpent`, `blackbox.TestTheSaveDoesNotSpendTheBudget`, `blackbox.TestEveryWriteReplyReportsTheBudget`, `blackbox.TestActionsOutsideTheAllowlistAreRefused` |
| L6 write-ahead journal and rollback | An `intent` record is on disk before any op is sent; `n4_rollback(batch_id)` replays the recorded inverse as a new batch; an intent with no result is `in-doubt` and is never auto-rolled back; a journal line that cannot be read (not UTF-8, or a batch id that cannot key a batch) is listed by `n4_list_batches` as `unreadable journal line N` and refuses the rollback of every batch it may belong to (any batch when its batch id is unknown; otherwise its own batch and the batch it names in `rollback_of`); every failure after the intent carries its `batch_id` (audit outcome `error`), and a multi-op batch that fails at op k journals an in-doubt result naming the ops sent and their inverse; a slot or fallback that changed since the batch wrote it is never overwritten (compare before restore); the rollback of a create is refused when a later batch not rolled back wrote under the created component or the component holds a link or child its create did not make (bench retro 2026-11-15 delta 1: a folder's rollback removed 10 blocks and 6 links later batches had added). | `journal.Journal`; `blackbox.TestASlotChangedAfterTheBatchIsNotOverwrittenByItsRollback`, `blackbox.TestTheIntentIsOnDiskBeforeTheStationReceivesTheOp`, `blackbox.TestInDoubtBatchIsRefusedAndShowsTheIntent`, `blackbox.TestRollbackOfACreateRemovesTheComponentAsALinkedBatch`, `blackbox.TestRollbackOfACreateRefusesWhenLaterBatchesWorkedUnderIt`, `blackbox.TestRollbackOfACreateRefusesWhenItHoldsAChildNoBatchMade`, `blackbox.TestAMultiOpBatchThatFailsMidWayNamesTheSentOpsAndTheirInverse`, `plan.TestAnUnexpectedFailureAfterTheIntentKeepsTheBatchID`, `writeops.TestAComponentOrRelinkIntentThatCannotBeWrittenStopsInDoubt`, `journal.TestAJournalLineWhoseBatchIsUnknownRefusesEveryRollback` (any batch: three batches refused), `blackbox.TestAJournalLineWhoseBatchIsUnknownRefusesEveryRollback` (the same refusal through the tool), `blackbox.TestAnUnreadableLineOfAnotherBatchDoesNotBlockARollback` |
| L7 audit | Every write call, dry, executed or refused, appends a redacted line to `audit.jsonl`. | `safety.AuditLog`, `redact_args`; `blackbox.TestAuditRecordsPlannedExecutedAndRefusedCalls`, `plan.TestAuditAndOutputsHoldNoSecretValue` |
| L8 operator-controlled policy and secrets | Station URLs, credential env prefix and TLS policy are start flags (`--station`, `--credential-env`, `--insecure-tls`); the model chooses only a configured NAME. HTTPS only. Credentials come from the environment, never from tool arguments. A failed login is never retried. | `go/internal/mcpn4/cli` (start flags); `blackbox.TestHTTPStationIsRefusedAtStartWithoutTheTestFlag`, `blackbox.TestCredentialEnvIsNotAToolArgument`, `blackbox.TestUnknownStationNameIsRefusedAndNamesTheFlag`, `session.TestTlsIsVerifiedUnlessTheOperatorMarksTheStationInsecure`, `box.TestAuthFailureRaisesOnceWithoutRetry` |

Default mode is read-only: write tools exist only with `--allow-writes`
(`blackbox.TestReadOnlyServerHasNoWriteToolsAndCallingOneIs32601`).

## 3. Live rules (B1199)

| Rule | Why | Enforced by |
|---|---|---|
| One op per `syncTo`. | Batched ops hide which one the station rejected. | `Client.Sync` refuses anything but one op dict (`go/internal/mcpn4/box/session.go`). |
| Read back every write from a fresh load. | A reply is not proof. The write reply carries `requested/accepted/observed/verdict`. | `tools_write` read-back; `blackbox.TestHappyPathIsVerifiedWithAnnotation`, `blackbox.TestCreateComponentReadBackMismatchIsReportedNotRaised`, `blackbox.TestSetSlotReadBackMismatchIsReportedNotRaised` |
| Write Status slots whole (value plus status). | A value-only write leaves the sibling `status` null and the output null. | `blackbox.TestStatusBooleanIsWrittenWhole`, `blackbox.TestValueOrStatusChildPathIsRefusedWithTheReason`, `writeops.TestPlanAndReadbackDifferentialPython` |
| An absent value means the type default. | The station omits values equal to the type default. | `box.status_value`; `box.TestChildrenAndStatusValue`, `blackbox.TestOmittedValueTakesTheTypeDefault` |
| Check dangling outputs after wiring. | A block whose `out` feeds nothing is usually a missed link. | `n4_find_dangling_outputs`; `blackbox.TestReportsComponentsWhoseOutIsNotALinkSource` |
| Lock-out is 5 failures in 30 s. | Retrying a bad login locks the shared account. | `AuthError` is never retried, and any `AuthError` pauses every station call for `--auth-cooldown` (>= 30 s) unless other credentials reconnect; `box.TestReplayRecordedFrames/auth_failure_in_makessc_sends_no_cleanup_del`, `session.TestARejectedLoginPausesTheNextConnectWithTheSameCredentials`, `session.TestARejectedBoxCallPausesEveryLaterStationRead` |
| A 401 says what to check. | The usual cause is the user's Authentication Scheme, not the password. The error names `HTTPBasicScheme`, the password and "do not retry". | `box.auth_message`; `box.TestReplayRecordedFrames/auth_failure_raises_once_without_retry`, `bql.TestA401OnTheBqlPathIsActionableAndNotRetried` |
| Complex slots show their display string. | A FlexAddress or BacnetAddress has no simple value; `None` hid the address. | `n4_read_slots` `value_display`; `readtools.TestComplexSlotWithoutAValueFallsBackToItsDisplayString` |
| A status slot names its flags. | A slot typed `baja:Status` carries its bits as its own value; read as a Status complex it came back `status: "0"`, `status_ok: true` while the station showed `{fault}`. | `n4_read_slots` `status_flags`; `readtools.TestAStatusValueSlotNamesItsFlagsAndIsNotOkWhenAnyIsSet`, `box.TestStatusFlagsNamesEveryBitAndIsEmptyOnlyWhenOk` |
| A type filter keeps subtypes. | `types: ["history:HistoryExt"]` kept nothing while the concrete extension type kept 12: an exact-spec filter makes the caller know every concrete type first. | `n4_navigate` reads the ancestry from the type registry (`types_note` when it cannot); `blackbox.TestTypesFilterMatchesASubtypeOfAGivenType`, `blackbox.TestTypesFilterFallsBackToExactTypesWhenTheRegistryCannotBeRead` |
| No import side effects. | An import once ran a `save` by accident (B1199). | `cli.TestEofEndsTheServerWithExitZero` (the Go binary has no import: its start prints nothing and contacts no station) |
| Confirm a save on disk, not by the action's `null` reply. | `save` always answers `null`. | `n4_save_station` compares `config.bog` mtime and sha256 (`--station-home`); `writeops.TestTheReplyToSaveIsNeverTakenAsProof`, `blackbox.TestAChangedConfigBogProvesPersistenceWithEvidence`. Without a station home: `persisted: unknown`. |
| A scripted client executes a write only on an explicit step, never a wrapper default. | A wrapper's dry-run-only flag did not reach `n4call` and an `n4_remove_component` executed on the lab station (restored with `n4_rollback`; its outside links re-created by hand). The server was right: the token came from a dry run of that plan in that session; the client chained the dry run into the execution because executing was its default. Evidence: niagara-research retro of 2026-11-16 (client A overamps) row M17, lab proof 3 incident. | `n4call` dry-runs every write unless `--execute`; a destructive write also needs `--allow-destructive` and its line's own `"execute": true`: `benchcall.TestAWriteWithoutExecuteIsADryRunAndSaysSo`, `benchcall.TestALostExecuteFlagLeavesAConfirmedDestructiveLineADryRun`, `benchcall.TestADestructiveLineWithoutItsOwnExecuteIsRefusedEvenWithBothFlags`. Another script that drives writes keeps the same shape (dry run by default, `dry_run: true` visible in the sent JSON before any destructive call): **manual**. |
| A Spyder download is done only when its state reads back good. | On a client B field job a Spyder Full Download failed without a word in the station and cost 43 min; the IOComp configuration was then checked by hand per Spyder. Good after a Full Download: `deviceStatus=1`, `compileStatusFlag=0`, `sErrorStatus` empty, `enabled` true (a first download that fails leaves the Spyder disabled). | `n4_commissioning_report` section `spyder` (with `spyder_spec`, the IOComp against the caller's table); `blackbox.TestCommissioningSpyderFlagsADeviceLeftDisabled`, `blackbox.TestCommissioningSpyderFlagsADeviceStatusOtherThanOne`, `blackbox.TestCommissioningSpyderComparesTheIOCompWithTheSpecTable`. Offline fixtures only: the lab station has no Spyder, so the IOComp query is not live-certified. |
| A token from an earlier station session is refused with the way to re-plan. | A dropped tunnel mid-batch left 10 history extensions half-configured and every pending token dead after the next `n4_connect`; a bare mismatch gave no way out. The refusal names the cause and says to repeat the same call with `dry_run=true`, check the new plan and execute with its new token; nothing is sent and no token is issued by the refusal. | `plan.State.offerReplan`; `blackbox.TestATokenFromAnEarlierSessionIsRefusedWithTheWayToReplan`, `blackbox.TestAStaleStateRefusalDoesNotClaimASessionChange` |
| Identity before writes. | Writing to the wrong station is the worst failure. | See L1. |
| Surface a permission-classifier block; never route around it. | When the harness permission classifier blocks a live write, even one sent by a scoped writer, the block is the operator's decision. Reaching the same write through another tool, a subagent or a peer session defeats it. Stop, name the blocked call and its reason, and wait for the operator. Evidence: 2026-10-02 session (`retros/2026-10-02-customer-remote-01-session.md`, `odd/tasks/mcp-n4-issue179-hygiene.md`). | **manual** |
| A rollback is `verified` only when it is faithful. | A re-created subtree can come back without configuration the kit cannot write: a frozen child's settings (e.g. a writable's `proxyExt`, which is never re-added) and the value of an input whose link is not re-created. The read-back compares each frozen child with the snapshot, and the confirmed remove journals each link-driven input's value as seen at execution (kept out of the confirmation hash, since a link keeps changing it: PR #187 blocking review R3-001); anything not brought back is listed (`frozen_config_not_restored`, `link_inputs_not_restored`, with the captured value) and the verdict is `partial`. A slot the station omits counts as its default: a scalar is equal when the captured value is the type default, a struct slot (e.g. a StatusNumeric `readValue`) when every captured child is (a child whose default the kit does not know always differs). A frozen child the read-back could not load is listed with `readback_error` and no `missing`, and the verdict is `unverified`, not `partial`: `partial` asserts what was not restored, a read error establishes nothing (v0.28.0 advisory review R4-001/R4-002). Verdict precedence: `mismatch` > `unverified` > `partial` > `verified`. Restore those by hand (`n4_set_slot`) or decide to accept them. Evidence: issue #179 R4-002/R3-002, B1200-G1/G2. | `blackbox.TestFrozenChildConfigNotRestoredDowngradesTheVerdictToPartial`, `blackbox.TestAFrozenChildWhoseConfigMatchesKeepsTheVerdictVerified`, `blackbox.TestAnInputLinkedFromOutsideTheSubtreeIsReportedNotLost`, `blackbox.TestASkippedRelinkReportsTheCapturedInputValue`, `blackbox.TestALinkDrivenValueThatChangesAfterTheDryRunKeepsTheTokenValid`, `writeops.TestTheFrozenCheckTreatsOmittedDefaultsAsEqualAndAReadErrorAsUnread`, `writeops.TestAFrozenCheckReadErrorMakesTheRollbackUnverifiedAndIsScrubbed` |
| A copy of a writable Modbus point needs its own register address. | A Modbus client device allows one writable point per register: the station resets a second writable proxy at the same address to `Hex:-1` and logs "Duplicate Writable point for register address" (`BModbusClientProxyExt.started` and `changed`), also when the address is set again afterwards. A field copy of a writable point (a field JACE, 4.14) and the 4.15.6.20 lab station both showed it; the same body without `regType`, or with `regType` first, failed the same way, while a distinct address (40010) stuck and the source's address stuck on the copy once the source was removed. A read-only `control:NumericPoint` copy keeps its address. | `n4_copy_component` `copies[].data_address` sets each copy's address in the same add (lab: two writable copies at 40010 and 40011 `verified`); the plan notes a copy that would share a register and the read-back names the cause. `writeops.TestACopyOverrideSetsEachCopysOwnAddressAndNotesTheDuplicates`, `writeops.TestACopyReadBackNamesTheDuplicateRegisterCause`, `blackbox.TestCopyAddressOverridesAreCheckedBeforeTheStation` |
| Replicate points onto many devices in one batch, sized against the budget. | A client B field job copied the same points onto many devices one `n4_copy_component`/`n4_set_slot` call at a time: 630 calls in 39 minutes for 112 points, and the `--max-writes` budget ran out before the save (field retro 2026-11-16, row U10). | `n4_replicate_points`: N x M adds under one token, per-target `copies[]` name/address overrides, one budget write per add, refused before any token when the plan exceeds `write_budget.remaining`; `blackbox.TestOneTokenReplicatesEveryPointOntoEveryTarget`, `blackbox.TestAReplicationLargerThanTheRemainingBudgetIsRefusedUpFront`, `writeops.TestEachTargetGetsItsOwnAddressOverride`. Live-certified on 4.15.6.20 (lab, 2026-11-17): 2 Modbus points (writable at 40002 with fallback 200, read-only at 30001) onto 2 devices, one with its own address 40010, batch `2efb76264f27481fa3adf6c6a11b69e5` `verified` (4 writes), rolled back by `7e7c43a1b878406d9c8c7888d017a819` (`verified`); with `--max-writes 3` the dry run was refused ("needs 4 write(s) but only 3 remain"). |
| A copy of a writable point with a non-null fallback commands every target device. | The fallback is a writable's output when no priority input is set, so each copy writes it to its device on first communication (field retro 2026-11-16, row U18: a power point at 0 = OFF and a setpoint at 200). On 4.15.6.20 a null fallback is absent from the copy body and a set one carries its value. | `n4_copy_component` and `n4_replicate_points` note each such point (never refuse); `writeops.TestACopyNotesAWritablePointWithANonNullFallback`, `writeops.TestACopyDoesNotNoteANullFallback`, `writeops.TestAReplicationNotesANonNullFallbackWithoutRefusing`; observed live in the dry runs of the lab proof above. |
| Declare probe-only elements. | Rename, reorder, facets and fire-topic are not live-certified (B1199 section 1199.7). Flags are, through `n4_set_flags` on 4.15.6.20 (`retros/2026-11-14-lab-415-bench-session.md` delta 9). | **manual**: say "unproven live" in the plan and run a scratch write first. |
| A template deploy carries the component tree, never the template's handle references. | Inside a `.ntpl` the configuration bindings, links and ORD values name components by template handle (`h:758`); sent to a station those handles name unrelated components. | `n4_deploy_template` leaves them out with a plan note each (`writeops.TestADeployAddsTheTemplateTreeWithItsSettings`). Live-certified on 4.15.6.20 (lab, 2026-11-17): deploy batch `243620bf270e4509b2c6caea799dbfcc` read back `mismatch` on a facets value the station re-encodes (fixture corrected), rolled back by `0bdf3c01820b4ca68b6f3ee15f303b6a` (`verified`); deploy batch `54f4d46a0c114811ae72c6fddeb3d900` `verified`, rolled back by `8b34ec7ff42742769b378b4b6a34a573` (`verified`); refusals (range, required, unknown setting, station template, missing module, existing name) observed live. |
| On a lab station, a module swapped under the same jar name must change the jar before the start. | Removing a module jar and re-adding one with the same name can leave the station registry "Up-to-date" without the module's types: the module runs, but the registry channel answers its types as errors, so the `n4_create_component` pre-check refuses them as "not known to the station" (a binary older than that pre-check, #659, sent the add and the station created the component). Module-swap step: stop the station, copy the jar into `modules/`, `touch` it, start, and check that the start log has a `[sys.registry] Out-of-date: Module …` line naming the module, then `Rebuilding registry...` and `Rebuilt: N types` (a start that logs the registry `Up-to-date` after a swap missed it); reconnect with `n4_connect` after the start. Evidence: lab proof 2 on the 4.15.6.20 lab station, niagara-research retro of 2026-11-16 (client A overamps) rows M12 and M14: the untouched swap left the client module's types unknown; the touched one logged "Module changed", then the rebuild. | The refusal names this route; `blackbox.TestAnUnknownTypeRefusalNamesTheStaleRegistryRoute`. The swap itself is **manual**. |
| A type-registry call that gets no answer is asked again once. | Right after a station starts or a module first loads, the first set of a slot still at its declared default failed once on the registry lookup and the retry worked (client A retro 2026-11-16 row M13). `box.Client.LoadTypes` retries an HTTP error, transport failure or malformed reply once after 0.5 s, so `n4_set_slot`, the create, link, alarm, history and flags prechecks and the listings inherit it; an answer (the station's own error, an `isErr` unknown-type entry, an AuthError) is never retried. Two failures refuse with "the station's type registry did not answer (it may have just started): retry". | `blackbox.TestASetOfADeclaredDefaultSurvivesOneFailedRegistryCall`, `TestTwoFailedRegistryCallsRefuseWithARetryHint`, `TestAnUnknownTypeAnswerIsNotRetried` |
| Alarm `msgText` does not survive the return to normal. | The toNormal transition rewrites the same record's `msgText` with `toNormalText` (empty by default), and alarm texts resolve against the record, never the extension's parent. A value that must outlive the event lives in a persisted slot, reaches both `toOffnormalText` and `toNormalText` through an `alarmData` key, and is read from the slot or a history. Detail and source lines: `build-n4-module-kit/types/logic.md` LC9; evidence `build-n4-module-kit/retros/2026-11-16-alarm-msgtext-tonormal.md`. | **mechanical**: `n4_alarm_text_check` (live) and `bog-nav <bog> alarm-text` (offline) flag `parent-path` and `empty-tonormal`; run one before and after writing an alarm text. |
| Niagara decodes $xx / $uhhhh escapes only inside a BQL body (BqlQuery.make); ORD slot paths are matched escaped. Validate operator text that goes into BQL on its DECODED form; never decode ORD paths before a scope check. | Live 4.15.6: `alarm_class=x$27$20or$20alarmClass$20like$20$27$2a` matched 27 of 27 alarm records, and `select uuid$20from$20control:NumericPoint` passed the no-from rule of `alarm:`; decoding an ORD path would compare a name the station never sees (bql-rt BqlQuery.java:24, baja SlotPath.java:229-306, EscUtil.java:86-117). | `bql.DecodeForBql` in `checkQueryText`, `validateTableQuery`, `AlarmWhere` and `tags.ValidateSelect`; `readtools.TestReadAlarmsRefusesAFilterThatBqlDecodesToAQuote`, `TestBqlQueryChecksTheDecodedQueryText`, `TestNeqlSelectChecksTheDecodedText`, `TestReadHistoryNeverDecodesTheIdAndNamesTheCallersText`. |

## 4. Mandatory session checklist

1. `n4_describe_session`, then `n4_connect` to a station the operator configured.
   Confirm the reported `stationName`, `version` and `tier` (section 5); with
   `tier_writes: refused` plan with dry runs only and follow the refusal's routes.
   After rebuilding the mcp-n4 binary, restart the MCP server (the client keeps the
   old process); `n4_describe_session`'s stale-binary warning shows it; never
   conclude a tool is missing before checking `server.build`.
2. For an inventory or any read wider than a few components, `n4_inventory` /
   `n4_bql_query` FIRST (section 7); then `n4_navigate` (optional `types` filter) and
   `n4_read_slots` only for the targeted components. **manual**
3. Call the write tool with its default `dry_run=true`.
4. Show the plan (ops, inverse, scope) to the human and wait for approval. **manual**
5. Repeat the same call with `dry_run=false` and the `confirmation_token`.
6. Read the `verdict`. Anything but `verified` (`partial`, `mismatch`, `failed`,
   `unverified`) is reported as it is and not retried blindly. `partial` (rollback only)
   names what was not restored (section 3); a rollback is `unverified` when a frozen child
   could not be read back (`readback_error` in its gap): read it again before deciding. An `in-doubt` batch is inspected first.
7. `n4_find_dangling_outputs` over the touched subtree. After a history rollout, `n4_read_history`
   (last N records of a few of the new histories) proves records arrive; the configuration read-back alone does not.
8. `n4_save_station` and report `persisted` with its `evidence`; `unknown` is not success.
9. Keep every `batch_id` in the report so the human can ask for `n4_rollback`.
10. Write the session retro: call `n4_session_retro_draft` (or run
    `go/bin/retro-draft --write --station NAME --state-dir DIR`, which writes
    `retros/YYYY-MM-DD-<station>-session.md` and its `pending` INDEX row; `--dry-run`
    prints only). Review the evidence-backed candidates, keep or edit the deltas and
    drop the noise; with no friction the honest result is `no new deltas; ...`.
    Propose, never apply: the kit is changed by a human, and GitHub issues are staged
    only by hand. Enforced by `TestRetroStepIsDocumented`, `blackbox.TestRetroToolReturnsMarkdownAndCandidatesForTheServerStateDir` and
    the Go tests of `go/cmd/mcp-n4/retro-draft`.

Never put credentials in tool arguments or chat.

## 5. Version tiers (B1197 section 1197.6)

- **A** 4.13 and 4.14: BOX 2.3 with `checkLinks`; full support.
- **B** 4.15 and 4.3: start read-only until a PoC matches (4.15 has no `apd`; 4.3 has
  no server `checkLink`).
- **C** any other build: enable writes only after `reg.loadContract`, `loadRoot` and
  a harmless scratch write succeed.

The gate is **enforced** (v0.5.0). `n4_connect` reads the oBIX `productVersion` from
`/obix/about/` (one GET, never retried) and reports `version`, `version_source`, `tier`
and `tier_writes` (`allowed`, `allowed-by-opt-in`, `refused`); `n4_describe_session`
repeats them. A version that cannot be read (no oBIX, no permission) is tier C, so a
failed detection only makes the server stricter; the reply then carries
`version_error`. A 401/403 there may count as a failed login toward the lock-out:
fix the user's oBIX permission before reconnecting. Every mutating tool then branches on
the tier, after the identity check and before the budget, scope and token:

- Tier A: unchanged.
- Tier B: an execution is refused unless the operator started the server with
  `--allow-tier-b NAME` for that station, after a PoC matched the build. The PoC is a full
  `live-smoke --apply --allow-tier-b NAME` on a scratch or laboratory station of the SAME
  build, never on the station the work is for: every required step `verified` and the scratch
  folder gone. Builds it has passed on: 4.15.6.20 (2026-11-14, 22 steps, the rollback of a
  nested subtree included).
- Tier C: an execution is refused unless the operator started the server with
  `--allow-tier-c NAME`, after `reg.loadContract`, `loadRoot` and a harmless scratch
  write succeeded. The server has no probe tool; the opt-in records that the probe ran.
- A dry run is always allowed; when the tier would block the execution its reply
  carries `tier_gate` (`tier`, `version`, `would_block`, `reason`), outside the plan
  hash. Reads are never gated.

The refusal names its routes (review the dry run now, use Workbench, run the PoC or
probe and restart with the opt-in) and starts with `safety.REASON_TIER`, which the
session retro classifies. Enforced by `tiers.TestTierTable`,
`blackbox.TestExecutionIsRefusedWithoutOptIn`,
`blackbox.TestTierBOperatorOptInAllowsWrites`, `blackbox.TestTierCOperatorOptInAllowsWrites`,
`blackbox.TestExecutionIsRefusedWithTheProbeSteps`,
`blackbox.TestTierAWritesWithoutAnnotation`,
`blackbox.TestDryRunIsAllowedAndSaysTheTierWouldBlock`,
`blackbox.TestEveryWriteToolIsGatedByTheTier` and `blackbox.TestReadsAreUnaffected`.

## 6. Evidence index (niagara-research)

- B1177: BOX recipe and wire encodings.
- B1179: auth, session and lock-out.
- B1192: decode rules and Status.
- B1197: MCP synthesis, route ladder, safety layers, tiers, build plan.
- B1199: live PoC on station `LLM` (create, link, kitControl, save, rollback).

## 7. Inventory and long reads (retro 2026-10-02)

A navigate/read crawl costs one BOX round trip per component, several seconds
each over a WAN. A first remote inventory crawl ran more than 16 minutes and did not
finish. The same inventory took 6 BQL queries and about 3 s.

| Rule | Enforced by |
|---|---|
| For a station inventory, call `n4_inventory` (networks, devices and points under `/Drivers`) or `n4_bql_query` FIRST. Use `n4_navigate` / `n4_read_slots` only for targeted components, e.g. to learn a driver's proxy-extension slot names (`dataAddress`, `objectId`) once per driver, then project them in BQL as `proxyExt.<slot>`. | Tool descriptions; `bql.TestQueryReturnsRowsAndColumnsFromThePathFormOrd`, `bql.TestInventoryCountsDevicesAndPointsPerNetworkWithLocalApart` |
| `n4_bql_query` is read-only: `select` only, no `|`, path-form `GET /ord/<ORD>` (the `/ord?` form answers 400), capped rows (default 5000, `truncated`), a timeout, and control characters stripped. | `go/internal/mcpn4/bql`; `bql.TestNonSelectAndPipeInjectionSendNothing`, `bql.TestRowCapAndTimeoutArePassedThrough`, `bql.TestParseCsvStripsBomAndControlCharsAndCapsRows` |
| A read is recorded: a session observation always, and an `audit.jsonl` line with outcome `read` when the write machinery is active. | `bql.TestTheReadIsRecordedAsASessionObservation`, `bql.TestTheReadIsAuditedWithOutcomeRead` |
| Any read expected to take more than about 1 minute writes progress to a file and reports counts. Never leave the human waiting on a piped, invisible stream. `n4_inventory` returns `progress` (rows per step) and appends the same lines to `--progress-file`. A hand-run crawl writes its own progress file. | `bql.TestProgressLinesGoToTheOperatorProgressFile`; **manual** for hand-run scripts |
| A long-running client (script, crawl, smoke run) takes a progress file (e.g. `--progress-file`) and writes each step there. It never relies on stdout or stderr piped through `tail`: the pipe shows nothing until the process ends. Evidence: retro 2026-10-02 D7 (`retros/2026-10-02-customer-remote-01-session.md`). | **manual** |
| Count devices without the network's built-in local device (`localDevice` on SnmpNetwork, the station acting as an agent). `n4_inventory` flags it `local: true` and reports `local_devices` apart from `field_devices`. | `bql.is_local_device`; `bql.TestLocalDevicesAreFlaggedAndCountedApart` |
| A network is found by its type, not by where it sits. `n4_inventory` takes its networks from the `driver:DeviceNetwork` query and gives each device and point to the deepest network above it, so a base above the driver container (the station root, a site folder) lists every network, named by its path below the base, and no plain folder. | `bql.TestInventoryFindsEachNetworkByItsRowUnderABaseAboveTheDriverContainer`, `bql.TestInventoryAtTheDriverContainerKeepsNetworkNames` |
| Load latency is visible: `n4_navigate` and `n4_read_slots` return `elapsed_ms`. Load polling starts at 0.1 s and backs off (0.2, 0.4, 0.8 s cap), with a bounded total wait. | `box.poll_delays`; `box.TestReplayRecordedFrames/load_tree_polls_with_a_short_first_delay_and_backoff`, `box.TestReplayRecordedFrames/load_tree_total_wait_is_bounded_and_each_delay_capped`, `readtools.TestReadSlotsAndNavigateReportElapsedMs` |

## 8. Kit development: delivery and review (retro 2026-10-03)

These rules govern changes to the kit itself, not station sessions.

| Rule | Enforced by |
|---|---|
| Each work unit merges as its own PR from the latest `origin/main`. Under that delivery, `gentle-ai review assess --committed-only` reports a medium-risk slice as `review_due: false` with `under_budget`: the slice never reaches the review budget because it is merged first. So at every PR, start the native review explicitly (`gentle-ai review status ... --next-transition`, then the returned START) for any medium or high candidate, whatever `review_due` says. Only a `passive` candidate skips the review. Evidence: retro 2026-10-03 D3 (H1, H5, H6 assessed `under_budget`). | **manual** |
| A mutation check runs both the mutated and the restored test with `go test -count=1`, so a cached result is never read as RED or GREEN, and before reading GREEN confirms the mutation is really gone with `git diff --exit-code` on the mutated file. This replaces the Python bytecode rule (`__pycache__`, retired with the Python suite in T9). Evidence: retro 2026-10-03 D1. | **manual** |
| Before each minor release, run the untested-branch audit (README "Run the tests", `go test -coverprofile`) and give every never-run error-return or refusal branch in the write path a test or a stated reason. Evidence: retro 2026-10-03 D2 (the H4 characterization tests found a swallowed error reason). | **manual** |

## 9. Reading Px views and station files

Px graphics are files (`file:^px/<name>.px`), not components: BOX and BQL reads do not show
them. A live station served them over two read-only HTTP routes, with the same HTTP Basic
credentials (still taken only from the operator's environment, never from tool arguments):

1. `GET /ord/<url-encoded station:|slot:/|bql:select slotPath, displayName, pxFile, media
   from baja:PxView|view:file:ITableToCsv>` lists the views. `slotPath` came back `NULL`:
   `BPxView` extends `BAbstractPxView`, a `BStruct` held as a property of its owning
   component, so it has no slot path of its own.
2. `GET /file/<path>` returns the file (`/file/px/General.px` answered the raw Px XML; an
   absent `/file/nav/NavFile.nav` answered 404).

Source notes (N4 web-rt): `web.xml` maps the `file` servlet (`FileServlet`) to `/file/*`;
`file:^a/b` corresponds to `/file/a/b` (`WebSnoopHtmlWriter.java:121-122`);
`FileServlet.canRead` requires read permission and refuses directories and `.bog` /
`.bog.gz` (`FileServlet.java:418-426`); `BPxView.pxFile` is a `BOrd` property.

| Rule | Enforced by |
|---|---|
| List views with `n4_list_px_views`, then read one with `n4_px_summary` (structure, bound ORDs, images, includes) or `n4_read_file` (raw text). A `NULL` slot path is reported as `slot_path: null` / `owner_path: null` with `owner_note`, never guessed. | `blackbox.TestPxViewsAreListedWithTheirFileAndMedia` |
| File reads are read-only `GET /file/<path>`; each path segment is URL-encoded. `..`, `|`, absolute or home paths, drive letters, backslashes, empty segments, directories (trailing `/`), `:` and segments ending in `.` or space (Windows aliases such as `config.bog.` or `config.bog::$DATA`) and `.bog` / `.bog.gz` are refused before any request. | `go/internal/mcpn4/files`; `files.TestUnsafePathsAreRefusedBeforeAnyRequest`, `files.TestBogFilesAreRefusedLikeTheFileServlet`, `files.TestWindowsPathAliasesOfRefusedFilesAreRefused`, `blackbox.TestUnsafePathsAndBogFilesSendNothing`, `files.TestTheUrlPathEncodesEachSegment` |
| Reads are capped (`max_bytes`, default 2 MB, at most 16 MB); a cut read says `truncated: true` and has no `sha256`. A binary file returns size, `sha256` and content type, never its bytes in the reply; `output_file` saves complete bytes only. | `blackbox.TestTheSizeCapTruncatesAndSaysSo`, `blackbox.TestABinaryFileReturnsSizeAndDigestWithoutContent`, `blackbox.TestATruncatedReadIsNotSavedAndAPathNameIsRefused` |
| A 401 on `/file/` latches station calls and is never retried (on a live station one user got 401 where another succeeded: check that user's Authentication Scheme and password). A 403 (no read permission) or 404 is a typed `status`, not an authentication failure, and does not latch. | `blackbox.TestA401OnTheFilePathLatchesAndIsNotRetried`, `files.TestMissingAndForbiddenFilesAreTypedResultsWithoutALatch` |
| The Px summary parses with the standard library only and refuses a `DOCTYPE` / `ENTITY` declaration (no entity expansion). Grouping bound ORDs by component strips a trailing value slot (`/out`, `/in1`..`/in16`, `|slot:out`, ...): a heuristic, the station is not asked. Every list is capped (`max_items`). | `px.TestADoctypeOrEntityDeclarationIsRefused`, `px.TestBoundOrdsAreGroupedByComponentWithTheTrailingSlotStripped`, `px.TestLabelTextsAreListedAndBounded` |
