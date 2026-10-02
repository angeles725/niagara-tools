# mcp-n4 METHODOLOGY

How an agent reads and writes a Niagara N4 station through `mcp_n4` without
hurting it. Each rule names the code or test that enforces it, or is marked
**manual** (the agent and the human must do it). Evidence lives in
niagara-research blocks (section 6); this file does not restate it.

Paths below are relative to `mcp-n4-kit/`. Test names are in `tests/`.

## 1. Route ladder

Never answer "cannot". When a route is blocked, state the next one, its cost and
what it needs.

| Route | Use | Status |
|---|---|---|
| R-A BOX JSON | Default. Read, create, set, link, invoke, remove, save with no module on the station. | Implemented (`mcp_n4/box.py` + tools). Live-certified in B1199. |
| R-B Java Fox sidecar | Fidelity cross-check, or fallback when BOX is disabled. Needs JRE 8 and the install jars. | Not built (B1197 phase P5). |
| R-F offline BOG-XML | Zero-risk dry run and diff against a `config.bog` snapshot: niagara-research `tools/bog-nav.py` and `tools/bog-write.py`. | External tools. **manual** |
| R-E oBIX | Values only (read, write a value, invoke). No structure editing. | External. **manual** |
| R-D in-station module | Last resort, only where BOX is blocked and a module is acceptable. It adds attack surface. | Not built. |

Order of preference: R-A, then R-F to rehearse or diff, then R-B, R-E, R-D.
Rule enforced by: **manual**.

## 2. Safety layers L1-L8 (B1197 section 1197.4)

| Layer | Rule | Enforced by |
|---|---|---|
| L1 identity | One dedicated station user for the agent; operator-level permissions on the target categories, never the super user. The server refuses a write unless the station's real `stationName` matched `expected_station`. | User: **manual**. Name check: `tools_write.py` identity gate; `test_write_on_an_unverified_session_is_refused_and_audited`, `test_expected_station_mismatch_closes_session_and_names_both` |
| L2 server-side enforcement | The station applies the user's permissions, slot limits and audit; the client never bypasses them. | The station. A 401/403 raises `AuthError`: `test_http_403_is_auth_error` |
| L3 dry run | `dry_run` defaults to true: the reply is the exact BOX ops and their inverse; nothing reaches the station. | `tools_write._process`; `test_dry_run_is_the_default_and_sends_no_mutation` |
| L4 confirmation token | Executing needs the single-use HMAC token from the dry run, bound to tool, arguments, plan hash and expiry. | `safety.ConfirmationTokens`; `test_a_token_is_single_use`, `test_args_changed_after_the_token_was_issued_are_refused`, `test_station_state_changed_since_the_dry_run_is_refused` |
| L5 write scope and limits | Every ORD must sit under a `--write-scope` prefix (slot boundaries); no prefix means no write; at most `--max-writes` per session; `n4_invoke_action` allows only `set`, `active`, `inactive`, `auto`. | `safety.WriteScope`; `test_no_scope_configured_refuses_every_write`, `test_prefix_boundary_is_respected`, `test_writes_beyond_the_session_budget_are_refused`, `test_actions_outside_the_allowlist_are_refused` |
| L6 write-ahead journal and rollback | An `intent` record is on disk before any op is sent; `n4_rollback(batch_id)` replays the recorded inverse as a new batch; an intent with no result is `in-doubt` and is never auto-rolled back. | `safety.Journal`; `test_the_intent_is_on_disk_before_the_station_receives_the_op`, `test_in_doubt_batch_is_refused_and_shows_the_intent`, `test_rollback_of_a_create_removes_the_component_as_a_linked_batch` |
| L7 audit | Every write call, dry, executed or refused, appends a redacted line to `audit.jsonl`. | `safety.AuditLog`, `redact_args`; `test_audit_records_planned_executed_and_refused_calls`, `test_audit_and_outputs_hold_no_secret_value` |
| L8 operator-controlled policy and secrets | Station URLs, credential env prefix and TLS policy are start flags (`--station`, `--credential-env`, `--insecure-tls`); the model chooses only a configured NAME. HTTPS only. Credentials come from the environment, never from tool arguments. A failed login is never retried. | `server.py`; `test_http_station_is_refused_at_start_without_the_test_flag`, `test_credential_env_is_not_a_tool_argument`, `test_unknown_station_name_is_refused_and_names_the_flag`, `test_tls_is_verified_unless_the_operator_marks_the_station_insecure`, `test_auth_failure_raises_once_without_retry` |

Default mode is read-only: write tools exist only with `--allow-writes`
(`test_read_only_server_has_no_write_tools_and_calling_one_is_32601`).

## 3. Live rules (B1199)

| Rule | Why | Enforced by |
|---|---|---|
| One op per `syncTo`. | Batched ops hide which one the station rejected. | `BoxClient.sync` raises on anything but one op dict (`box.py`). |
| Read back every write from a fresh load. | A reply is not proof. The write reply carries `requested/accepted/observed/verdict`. | `tools_write` read-back; `test_happy_path_is_verified_with_annotation`, `test_read_back_mismatch_is_reported_not_raised` |
| Write Status slots whole (value plus status). | A value-only write leaves the sibling `status` null and the output null. | `test_status_boolean_is_written_whole`, `test_value_or_status_child_path_is_refused_with_the_reason`, `test_set_slot_rejects_partial_status_by_default` |
| An absent value means the type default. | The station omits values equal to the type default. | `box.status_value`; `test_status_value_applies_type_defaults`, `test_omitted_value_takes_the_type_default` |
| Check dangling outputs after wiring. | A block whose `out` feeds nothing is usually a missed link. | `n4_find_dangling_outputs`; `test_reports_components_whose_out_is_not_a_link_source` |
| Lock-out is 5 failures in 30 s. | Retrying a bad login locks the shared account. | `AuthError` is never retried; `test_auth_failure_in_makessc_sends_no_cleanup_del` |
| A 401 says what to check. | The usual cause is the user's Authentication Scheme, not the password. The error names `HTTPBasicScheme`, the password and "do not retry". | `box.auth_message`; `test_http_401_names_the_auth_scheme_the_password_and_no_retry`, `test_a_401_on_the_bql_path_is_actionable_and_not_retried` |
| Complex slots show their display string. | A FlexAddress or BacnetAddress has no simple value; `None` hid the address. | `n4_read_slots` `value_display`; `test_complex_slot_without_a_value_falls_back_to_its_display_string` |
| No import side effects. | An import once ran a `save` by accident (B1199). | `test_import_makes_no_network_call`, `test_import_has_no_side_effects_in_fresh_process`, `test_import_prints_nothing` |
| Confirm a save on disk, not by the action's `null` reply. | `save` always answers `null`. | `n4_save_station` compares `config.bog` mtime and sha256 (`--station-home`); `test_the_reply_to_save_is_never_taken_as_proof`, `test_a_changed_config_bog_proves_persistence_with_evidence`. Without a station home: `persisted: unknown`. |
| Identity before writes. | Writing to the wrong station is the worst failure. | See L1. |
| Declare probe-only elements. | Rename, reorder, flags, facets and fire-topic are not live-certified (B1199 section 1199.7). | **manual**: say "unproven live" in the plan and run a scratch write first. |

## 4. Mandatory session checklist

1. `n4_describe_session`, then `n4_connect` to a station the operator configured.
   Confirm the reported `stationName` and the version tier (section 5).
2. `n4_navigate` and `n4_read_slots` the target subtree. **manual** For an inventory
   or any read wider than a few components, use `n4_bql_query` / `n4_inventory`
   first (section 7).
3. Call the write tool with its default `dry_run=true`.
4. Show the plan (ops, inverse, scope) to the human and wait for approval. **manual**
5. Repeat the same call with `dry_run=false` and the `confirmation_token`.
6. Read the `verdict`. Anything but `verified` (`mismatch`, `failed`, `unverified`)
   is reported as it is and not retried blindly. An `in-doubt` batch is inspected first.
7. `n4_find_dangling_outputs` over the touched subtree.
8. `n4_save_station` and report `persisted` with its `evidence`; `unknown` is not success.
9. Keep every `batch_id` in the report so the human can ask for `n4_rollback`.
10. Write the session retro: call `n4_session_retro_draft` (or run
    `python3 tools/new_retro.py --station NAME --state-dir DIR`, which writes
    `retros/YYYY-MM-DD-<station>-session.md` and its `pending` INDEX row; `--dry-run`
    prints only). Review the evidence-backed candidates, keep or edit the deltas and
    drop the noise; with no friction the honest result is `no new deltas; ...`.
    Propose, never apply: the kit is changed by a human, and GitHub issues are staged
    only by hand. Enforced by `TestRetroStepIsDocumented` and the `test_retro.py` /
    `test_new_retro.py` suites.

Never put credentials in tool arguments or chat.

## 5. Version tiers (B1197 section 1197.6)

- **A** 4.13 and 4.14: BOX 2.3 with `checkLinks`; full support.
- **B** 4.15 and 4.3: start read-only until a PoC matches (4.15 has no `apd`; 4.3 has
  no server `checkLink`).
- **C** any other build: enable writes only after `reg.loadContract`, `loadRoot` and
  a harmless scratch write succeed.

Print the detected version and tier at connect. The tier gate is **manual**;
the server does not yet branch on it.

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
| For a station inventory, call `n4_inventory` (networks, devices and points under `/Drivers`) or `n4_bql_query` FIRST. Use `n4_navigate` / `n4_read_slots` only for targeted components, e.g. to learn a driver's proxy-extension slot names (`dataAddress`, `objectId`) once per driver, then project them in BQL as `proxyExt.<slot>`. | Tool descriptions; `test_query_returns_rows_and_columns_from_the_path_form_ord`, `test_inventory_counts_devices_and_points_per_network_with_local_apart` |
| `n4_bql_query` is read-only: `select` only, no `|`, path-form `GET /ord/<ORD>` (the `/ord?` form answers 400), capped rows (default 5000, `truncated`), a timeout, and control characters stripped. | `mcp_n4/bql.py`; `test_non_select_and_pipe_injection_send_nothing`, `test_row_cap_and_timeout_are_passed_through`, `test_parse_csv_strips_bom_and_control_chars_and_caps_rows` |
| A read is recorded: a session observation always, and an `audit.jsonl` line with outcome `read` when the write machinery is active. | `test_the_read_is_recorded_as_a_session_observation`, `test_the_read_is_audited_with_outcome_read` |
| Any read expected to take more than about 1 minute writes progress to a file and reports counts. Never leave the human waiting on a piped, invisible stream. `n4_inventory` returns `progress` (rows per step) and appends the same lines to `--progress-file`. A hand-run crawl writes its own progress file. | `test_progress_lines_go_to_the_operator_progress_file`; **manual** for hand-run scripts |
| Count devices without the network's built-in local device (`localDevice` on SnmpNetwork, the station acting as an agent). `n4_inventory` flags it `local: true` and reports `local_devices` apart from `field_devices`. | `bql.is_local_device`; `test_local_devices_are_flagged_and_counted_apart` |
| Load latency is visible: `n4_navigate` and `n4_read_slots` return `elapsed_ms`. Load polling starts at 0.1 s and backs off (0.2, 0.4, 0.8 s cap), with a bounded total wait. | `box.poll_delays`; `test_load_tree_polls_with_a_short_first_delay_and_backoff`, `test_load_tree_total_wait_is_bounded_and_each_delay_capped`, `test_read_slots_and_navigate_report_elapsed_ms` |
