<!-- review-status: pending -->
<!-- Marker lifecycle: the maintainer flips 'pending' above to 'folded <date> · kit <sha>' once the proposed deltas are reviewed and applied (or dismissed) in the kit. This retro only PROPOSES; kit changes are human-reviewed and human-committed. -->
# Retro — customer-remote-01 · 2026-10-02 · mcp-n4 session self-retrospective

> Written by hand from the session transcript, following `mcp_n4/templates/retro.template.md`. This session made
> no writes, so `audit.jsonl` / `journal.jsonl` contain no evidence and `n4_session_retro_draft` would have
> returned the honesty line. The deltas below are friction observed in the session itself. The reviewer keeps,
> edits or drops each one. Stage GitHub issues only by hand; nothing here is applied automatically.

## Session summary

First live use of the kit against a remote customer station: `customer-remote-01` (N4, remote VM behind a WireGuard
tunnel, about 103 ms RTT). The goal was a read-only inventory of every device and every point, and what each
point reads. Mode stayed `read-only` for the whole session; no write tool was called.

Timeline:

1. The kit was not runnable from the operator checkout: the shared `niagara-tools` checkout sat on an unrelated
   branch with no `mcp-n4-kit/`, and no MCP server was registered in the client. The kit was extracted from
   `origin/main` with `git archive` and driven over stdio by hand.
2. `n4_connect` returned `HTTP 401 from station` for one attempt. It was not retried (lockout rule). After the
   operator confirmed that the user had `HTTPBasicScheme`, the next attempt connected.
3. A recursive crawl of `station:|slot:/Drivers` was started: `n4_navigate` depth 1 per component,
   `n4_read_slots` on every Point, Device, Network or Ext, recursing up to 8 levels. After more than 16 minutes it
   had not finished, and its progress was invisible because stdout was piped through `tail`. It was killed.
4. The same inventory was taken with BQL over HTTP Basic: `GET /ord/<url-encoded ORD>` with
   `station:|slot:/Drivers/<Net>|bql:select … from control:ControlPoint|view:file:ITableToCsv`. Every query
   took 0.37–0.77 s. The full inventory (5 networks, 29 field devices plus the SNMP local agent, 507 points, with addresses, live values and
   facets) took 6 queries and about 3 s.
5. `n4_read_slots` was still needed once per driver to learn the proxy-extension slot names (`dataAddress`,
   `objectId` / `propertyId`, `objectIdentifier`) for the BQL projections.

## Proposed kit deltas

| # | Delta | Evidence | Proposed change | Cost |
|---|---|---|---|---|
| D1 | No bulk read tool: inventory work degrades into one round trip per component | The crawl ran more than 16 min unfinished; BQL returned all 507 points in 0.77 s on the same station and tunnel (about 1000× faster) | Add a read-only `n4_bql_query` tool: projected BQL against a subtree, results as rows; reject anything that is not a `select`; cap the row count; record it in the audit as a read | M |
| D2 | `load_tree` polls on a fixed 0.5 s delay with no latency awareness | `box.py` `load_tree(attempts=6, delay=0.5)`; on a ~100 ms RTT link each `n4_navigate` costs about 1–3 s even when the load op arrives early | Poll with a short first delay and backoff (for example 0.1 → 0.2 → 0.4 s); expose per-call timing in the result so slowness is visible | S |
| D3 | `n4_navigate` depth is capped at 3, and there is no "walk the subtree" guidance | Mapping needed depth ≥ 5 (Drivers → Network → Device → points → Point → proxyExt), so the client recursed at depth 1 | METHODOLOGY: for a station inventory use BQL (D1) first; use navigate only for targeted components. Optionally let `n4_navigate` take a `types` filter | S |
| D4 | `n4_read_slots` returns `value: None` for complex address slots | Modbus `dataAddress` / `absoluteAddress` (`modbusCore:FlexAddress`) and BACnet device `address` came back `None`; BQL rendered them as `Decimal:302` and `1:<ip>:47808` | Fall back to the slot's display string (`toString`) for complexes the reader does not decode | S |
| D5 | The 401 error is not actionable | Output was the bare `HTTP 401 from station`; the cause was the user's authentication scheme, which the operator had to fix by hand | Map 401 to a hint: "check the user's Authentication Scheme Name = HTTPBasicScheme and the password; do not retry (lockout)" | S |
| D6 | Kit availability depends on the operator checkout branch | The shared checkout was on `retro/2026-09-26-change-tier-time-budgets`, which has no `mcp-n4-kit/`; the skill launcher's default `KIT` path did not exist | Launcher: when `KIT` is missing, say so and offer `git worktree add … origin/main` (or an installed copy) instead of searching; README: register the server per project in `.mcp.json` once | S |
| D7 | Long-running reads give no progress | Crawl progress went to stderr and the client piped it through `tail`; the operator asked twice for status with nothing to show | Methodology rule: any read longer than about 1 min writes progress to a file and reports counts; a future bulk tool should stream or page | S |
| D8 | A device-type BQL query counts each network's built-in local device as a field device | `select … from driver:Device` on `SnmpNetwork` returned 3 rows; one was `localDevice`, the network's frozen `BSnmpAgent` slot (`snmp-rt` `BSnmpNetwork.java:190`), which is the station itself acting as an SNMP agent. The operator caught the miscount (2 real devices, not 3) | Inventory guidance and the future bulk tool (D1): exclude frozen local-device slots (`localDevice`, BACnet `localDevice`) or flag them as `local`; report counts with them separated | S |

## Already covered (dedupe)

- No retry after a failed login: METHODOLOGY section 3 and the skill's Hard Rules. It was followed: one attempt,
  then the cause was fixed before the second.
- Read-only by default: server mode `read-only` without `--allow-writes`. It held for the whole session.
- Only operator-configured stations are contacted: the station was passed by `--station` on the operator's
  explicit authorization of destination, operation and credential.

## Honest verdict

The kit's safety model worked as designed. Its read path is unfit for inventory work over a WAN, and that one
gap (D1, amplified by D2 and D3) cost the operator a 16+ minute wait that BQL resolved in seconds. Recommended
fold order: D1 → D2 → D4 → D5, with D3, D6, D7 and D8 as documentation-only changes.
