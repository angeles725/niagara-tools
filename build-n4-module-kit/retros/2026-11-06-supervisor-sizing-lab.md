<!-- review-status: pending -->
<!-- module: kit -->
<!-- deltas: 5 -->
# 2026-11-06 · kit · supervisor-sizing-lab

**Session**: niagara-research, Supervisor sizing lab on `Prueba_4_15` (Niagara 4.15.6.20), research block
B1432 (`niagara-research/niagara-mental-model-bloque1432.md`, branch `docs/b1432-supervisor-sizing-lab`)
**Delta count**: 5

## What happened
A client needed a Supervisor sized for 53,750 points logged every 5 minutes for 184 days in the native history
database. The lab measured storage, import cost, trend-query latency and concurrency, live oBIX dashboards,
the alarm console, sensor scaling from 500 to 20,000 points, and a heap change. The kit already holds the
three defaults that break a 6-month design: history capacity 500 (`types/logic-authoring.md:776`), file alarm
DB 500 / 250,000 (`BUILD-LOOP.md:662-663`), and where the heap is set (`types/platform.md:46`). The deltas
below add what the lab measured on top of them.

## Evidence
- Raw results and scripts: `niagara-research/sources/probes/B1432-supervisor-sizing-lab/` `[ev: B1432]`
- Week queries reach ~100 req/s; 6-month raw queries plateau at ~4.5 req/s and ~4.6 cores from 8 to 64
  clients `[ev: B1432 §1432.4]`
- `-Xmx4G` instead of `-Xmx1G`: no gain for 6-month queries; heap used never above ~340 MB `[ev: B1432 §1432.7]`
- 500 → 20,000 one-second points: idle 0.012 → 0.064 cores; 50 dashboards × 100 points 1.2 → 2.1 cores;
  boot 17 s → 49 s `[ev: B1432 §1432.5]`
- A CSV history import kept only 500 records until `configOverrides` carried `capacity` = `h:Capacity`
  `"1:52992"` `[ev: B1432 §1432.7]`

## Proposed kit deltas (propose-never-apply)
| Δ | Delta | Target file / § | Token |
|---|---|---|---|
| Δ1 | Raising `-Xmx` does not speed long trend queries; at 20,000 points the heap used stays near 300 MB. Raise heap for drivers and object count, not for chart speed | `types/platform.md` § heap (line 46) | `[ev: B1432 §1432.7]` |
| Δ2 | Alarm retention arithmetic: 250,000 ÷ retention days = maximum alarm records per day (1,358/day for 184 days); past that the only documented archive (`OrionArchiveAlarmProvider`, Supervisor, 4.11+) needs an RDBMS | `BUILD-LOOP.md` step 7, after "Size the store" | `[ev: B1432 §1432.6]` |
| Δ3 | Dashboards serve long ranges (months) from rollups and fetch raw samples on demand: a raw 6-month chart costs ~0.4–0.6 s CPU and the raw path saturates at ~4.5 req/s | `types/dashboard.md` § history/chart data | `[ev: B1432 §1432.4]` |
| Δ4 | Supervisor sizing reference: ingest ≈ 7.6 µs CPU per record; 50 live dashboards ≈ 2 cores; boot time grows with the configuration (49 s at 20,000 objects), so a failover standby must run hot | `types/platform.md`, new § Supervisor sizing (links B1432) | `[ev: B1432 §1432.3, §1432.5]` |
| Δ5 | A history import created from the palette keeps 500 records; set `capacity` through the descriptor's `configOverrides` (`h:Capacity`, `"<restrictBy>:<max>"`, `"1:52992"`; `"0:0"` = unlimited) before the first execute | `types/logic-authoring.md` § Logging a point to history (after line 776) | `[ev: B1432 §1432.7]` |

## Lessons
- `file:^` resolves to the station `shared/` directory; a CSV under the station root next to it is not found.
- `alarm:|view:file:ITableToCsv` is not a valid export ORD; use `alarm:|bql:select *|view:file:ITableToCsv`.
- An elevated console station cannot be killed from WSL (`taskkill` access denied); start it from WSL with
  its console on a FIFO and stop it with `save` + `quit` written to the FIFO.

---
**Status**: PENDING — INDEX row generated from the header: `| 2026-11-06-supervisor-sizing-lab.md | kit | 2026-11-06 | pending | 5 |`
