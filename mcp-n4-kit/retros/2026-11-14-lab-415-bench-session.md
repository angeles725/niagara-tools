<!-- review-status: pending -->
<!-- Marker lifecycle: the maintainer flips 'pending' above to 'folded <date> · kit <sha>' once the proposed deltas are reviewed and applied (or dismissed) in the kit. Deltas 1-5 and 8 below were applied in the same pass under the operator's standing instruction for this bench; 7 followed in the next pass, 6 in part (its runtime half stays proposed); 9 was applied later against the in-repo fake station only and is not yet proven on a live station. -->
# Retro — lab-415 · 2026-11-14 · mcp-n4 bench session on a laboratory station

> Written by hand from the session's `audit.jsonl` and `journal.jsonl` (20 audit entries, 10 verified batches)
> and from the saved `config.bog`. The mechanical draft (`retro-draft --write --dry-run`) found "no new deltas":
> every batch said `verified`, including the one that lost configuration (delta 5), which is itself delta 6.

## Session summary

- Station: a laboratory station on Niagara 4.15.6.20 (tier B), started for the bench; the operator authorized
  writes, creates and deletes on it.
- `live-smoke --apply`: 22 steps verified, the probe-only rollback of a nested subtree included (3 links restored).
- Copy of a point whose frozen slots carry integrator flags: flags lost before the fix, kept after it; both
  outcomes read from the saved `config.bog`.
- The station was left as found: every scratch component removed, saved, `persisted: true`.

## Proposed kit deltas

| # | Proposed change | Target (file · section) | Evidence (batch_id / tool call / audit line) | Type | Priority |
|---|---|---|---|---|---|
| 1 | `live-smoke` passes the operator's `--allow-tier-b NAME` / `--allow-tier-c NAME` to the server; without them it could not run on any station that is not tier A. APPLIED 576ee1cc | `go/internal/mcpn4/livesmoke` · `README.md · Live smoke test` | first run: `create Folder McpSmoke` failed, detail "version tier refuses writes: station ... is tier B (version 4.15.6.20)" | tool | high |
| 2 | A step that ran and did not verify prints its reason on the console under its line; before, the reason was only in the report. APPLIED 576ee1cc | `go/internal/mcpn4/livesmoke/reason.go` | the same run printed `failed    create Folder McpSmoke` and 20 `skipped` lines, nothing else | tool | high |
| 3 | `--report PATH` is checked before the server starts: a missing directory is exit 2 naming it. Before, the whole scenario ran and the report was lost with a temp-file name in the message. APPLIED 576ee1cc | `go/internal/mcpn4/livesmoke/run.go` | `live-smoke: open .../nodir/.report.json-<id>.tmp: no such file or directory` after the run | tool | medium |
| 4 | `n4call`, a bench client: runs the tool calls of a JSON-lines file against a server started over stdio, a write as dry run, token, execute; one JSON line per request, credentials masked. APPLIED 59cb2521 | `go/cmd/mcp-n4/n4call` · `go/internal/mcpn4/benchcall` · `README.md` | the bench had no way to call one tool on a station that is not registered in the agent's MCP configuration | tool | medium |
| 5 | A copy keeps the flags of a frozen slot at its default value: such a slot arrives as name plus `f` and was dropped as name-only before its flags were carried. APPLIED 8881222d | `go/internal/mcpn4/writeops/copy_strip.go` · `copy_tree.go` | batch fe844fb1...: source `facets f="h"`, `fallback f="o"`; copy had neither, verdict `verified`. After the fix the dry-run op carries `{"nm":"p","n":"facets","f":"h"}` and the saved bog shows both flags on the copy | defect | high |
| 6 | The copy verdict compares the copy with the SOURCE configuration, not only with the planned add op: the read-back already compares type, value and flags (`diffTree`), but a slot the plan itself dropped is missing from both sides, so the loss of delta 5 could not surface. A source child that is neither in the add op nor in the plan notes is a difference. PARTIAL: applied as a plan invariant (`copy_accounting_test.go`: every source child is copied, listed in the notes or bare; it fails when the flag guard of delta 5 is removed); the runtime verdict still compares the read-back with the plan, so comparing it with the source stays proposed | `go/internal/mcpn4/writeops/copy_readback.go` · `METHODOLOGY.md · section 2` | the batch of delta 5 reported `"differences": []` | defect | high |
| 7 | The rollback of a removed nested subtree stops being probe-only in `live-smoke`: it gates the exit code, and the plan text no longer says "not yet proven live". APPLIED | `go/internal/mcpn4/livesmoke` (scenario, plan) · `README.md · Live smoke test` | step `rollback folder removal`: verified, `relinks: {restored: 3, skipped: 0}` on 4.15.6.20 | doctrine | medium |
| 8 | Section 5 says what the tier B PoC is in practice (a full `live-smoke --apply` on a scratch or laboratory station of the same build) and lists the builds it has passed on, starting with 4.15.6.20. APPLIED in this pass | `METHODOLOGY.md · section 5` | report of the run: `ok: true`, 22 verified | doctrine | medium |
| 9 | A flags tool (`n4_set_flags`, dry run and token like every write) so an integrator flag can be set, proven and repaired without Workbench; today a flag specimen needs an offline edit of `config.bog` with the station stopped. APPLIED (`n4_set_flags`: plan with both flag sets, exact inverse, read-back verdict, rollback). Live on 4.15.6.20 (2026-11-15, scratch folder `Flags648`, removed afterwards): dynamic slot `nw` `""` to `h`, frozen property `in10` `ts` to `ths`, frozen action `set` `o` to `ho`, all `verified`; after `n4_save_station` the `config.bog` held `<p n="nw" f="h"`, `<p n="in10" f="ths"/>`, `<a n="set" f="ho"/>`; the three rollbacks read back `verified` and the next save left none of the three `f` attributes | `go/internal/mcpn4/writeops/setflags.go` · `README.md · Writing safely` | the specimen for delta 5 was made by stopping the station and editing the bog | tool | low |
| 10 | `n4_set_flags` reads a listed slot without `f` that no type contract declares as a dynamic slot with no flags, and journals `""` as the inverse. That is right for a dynamic slot, but a contract walk that ends at a type the registry channel answers with `{}` cannot tell "no such frozen slot" from "type not described". The tool should say in the plan note when the walk ended at an undescribed type, and the kit should list which types a live station leaves undescribed. OPEN. Live on 4.15.6.20 (2026-11-15, dry runs only): `stationName` of the station root read `ro` from the `baja:Station` contract, `Services` and `UserService/lockOutEnabled` read none | `go/internal/mcpn4/writeops/setflags_current.go:39-40` · `go/internal/mcpn4/box/registry.go:11` | independent verifier of PR #648: a rig with `acme:Widget -> baja:Station -> {}` plans the inverse `""` with no refusal | tool | low on the lab station, unknown elsewhere |

## Already covered (dedupe)

- A frozen slot DECLARED transient (a writable point's `in16`, flags `tsL`) is left out of a copy and listed in the
  plan notes: `copy_strip.go` already documents it; the first specimen tried was this case and proved nothing.
- A tier B station refuses an execute and keeps dry runs: METHODOLOGY section 5.

## Honest verdict

Eight deltas applied (9 proven live on 2026-11-15) and one applied in part (6). The one that matters most is 5 with 6: the tool reported `verified`
over a copy that had lost configuration, and nothing in the audit could show it.

INDEX row: | 2026-11-14-lab-415-bench-session.md | lab-415 | 2026-11-14 | pending | 9 |
