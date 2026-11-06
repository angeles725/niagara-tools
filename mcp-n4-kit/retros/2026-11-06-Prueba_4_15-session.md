<!-- review-status: pending -->
<!-- Marker lifecycle: the maintainer flips 'pending' above to 'folded <date> · kit <sha>' once the proposed deltas are reviewed and applied (or dismissed) in the kit. This retro only PROPOSES; kit changes are human-reviewed. -->
# Retro — Prueba_4_15 · 2026-11-06 · mcp-n4 session self-retrospective

> Written by hand from the session transcript. The deltas below are friction observed while using the kit for
> a Supervisor sizing lab (research block B1432 in niagara-research). The reviewer keeps, edits or drops each
> one. Stage GitHub issues only by hand; nothing here is applied automatically.

## Session summary

Local lab station `Prueba_4_15` (Niagara 4.15.6.20, tier B), operator-authorized for any change. The goal was
to build measurement fixtures: a folder of history-import descriptors that read CSV files, and up to 20,000
`kitControl:SineWave` points (50 with an alarm extension). The kit was launched as a dedicated stdio server
with `--allow-writes`, write scopes under `/Drivers`, and `--allow-tier-b Prueba_4_15` after the scratch-folder
PoC the methodology requires (create, verify, remove).

Timeline:

1. Tier-B writes were refused until the PoC ran; after it, the opt-in worked as documented.
2. The import descriptors needed slots at their type default (omitted by the station) and an `int`/`Capacity`
   value (`capacity` = `1:52992`). `n4_set_slot` refused both, so the fixtures were written by editing
   `config.bog` offline with the station stopped.
3. Running an import needs the `execute` action. `n4_invoke_action` allows only `set`, `active`, `inactive`,
   `auto`, so each import was triggered with an oBIX `POST <obj/>` to `.../ImpNN/execute/`.
4. All load measurement (concurrent history reads, alarm reads, oBIX watches) was done with bash scripts,
   because the kit has no read-only load or latency probe.
5. `n4_save_station` returned `persisted: unknown` because no `--station-home` was configured. That is the
   documented behaviour; the save was proven from the station console instead.

## Proposed kit deltas

| # | Delta | Evidence | Proposed change | Cost |
|---|---|---|---|---|
| D1 | `n4_invoke_action` cannot run `execute`, so history import/export descriptors and other job-style components cannot be triggered | `go/internal/mcpn4/writeops/invoke.go:13` `AllowedActions = set, active, inactive, auto`; 50 imports had to go through oBIX `POST .../execute/` | Allow `execute` for a short allowlist of types (history import/export descriptors), with the same dry run, confirmation token and write-scope check, and a read-back of `lastSuccess`/`lastFailure` as the verdict | M |
| D2 | `n4_set_slot` has no integer or encoded simple types, so `int` capacities and `history:Capacity` cannot be set | `writeops/setslot.go:46` rejects unknown `value_type`; `alarm:FileAlarmDbConfig.capacity` is an int (500 default) and history `capacity` is a `BCapacity` encoded as `"1:52992"` | Add `baja:Integer` and a string-encoded path for a small set of `BSimple` types (`history:Capacity` first), each with a read-back comparison | M |
| D3 | The refusal for a default-valued (omitted) slot names only Workbench or an action as routes | `writeops/setslot.go:68-70`; on a lab station without Workbench the working route was an offline `config.bog` edit with the station stopped | Add the offline bog edit to the refusal text and to the METHODOLOGY route ladder, with its preconditions (station stopped, backup kept, xmllint, read-back after start) and its risk | S |
| D4 | No read-only latency or load probe, so sizing questions need ad-hoc scripts | B1432 load generators in `niagara-research/sources/probes/B1432-supervisor-sizing-lab/scripts/` | Either a read-only `n4_probe_latency` tool (repeat a BQL or history ORD read with K concurrent clients, report p50/p95/max, hard caps on K and duration) or a METHODOLOGY note that load testing is out of scope, pointing to the B1432 scripts | M |

## Already covered (dedupe)

- Tier-B opt-in only after a PoC: worked exactly as METHODOLOGY section 5 describes.
- `persisted: unknown` without `--station-home`: documented in the skill and README.
- The station omits default values: METHODOLOGY line 49 states it; D3 only adds a route.

## Honest verdict

The kit's safety model held: every refused write was refused for a documented reason. Its write surface
is too narrow for fixture building on a lab station (no `execute`, no integer or encoded simple types), so the
session left the kit for offline bog edits and raw oBIX. D1 and D2 would have kept the whole lab inside the
kit's audit trail.
