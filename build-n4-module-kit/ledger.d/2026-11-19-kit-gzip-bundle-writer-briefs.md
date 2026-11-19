<!-- ledger: kit -->
<!-- date: 2026-11-19 -->
<!-- last-change-note: report-module bundle-gzip row; static serving of a gzip twin; parallel SPA writer briefs -->
log: 2026-11-19 · report-module: a `-ux` part whose `src/rc/dist/*.js` is over 300 KB with no `.gz` twin is a WARN `bundle-gzip` row (no `dist/`, no row). `types/dashboard.md` § Static serving documents the deterministic twin and its `Content-Encoding` + `Vary` serving. `ORCHESTRATION.md` §3.c and the `BUILD-LOOP.md` § 2 parallel writer checklist: each brief names the module-map row of every path; each writer runs `lint-module-map.sh` and the whole `npm test` before done. Retro kit-gzip-bundle-writer-briefs (Δ1-Δ3 applied). [ev: retro kit-gzip-bundle-writer-briefs Δ1]
