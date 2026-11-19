### Added — report-module `bundle-gzip`; static serving of a pre-compressed bundle; parallel SPA writer briefs

- `report-module.sh` adds a `bundle-gzip` row on every `-ux` part that has a `src/rc/dist/`: a `*.js` there over
  300 KB with no `<name>.gz` next to it is a WARN row (advisory, never fails the report), none is one PASS row,
  and a part with no `src/rc/dist/` gets no row (retro kit-gzip-bundle-writer-briefs Δ1).
- `types/dashboard.md` § Static serving: the bundle step writes deterministic `.gz` twins and the static handler
  serves them with `Content-Encoding: gzip` and `Vary: Accept-Encoding`; `types/frontend-standard.md` § 5 and
  § Enforcement point to it.
- `ORCHESTRATION.md` §3.c and the `BUILD-LOOP.md` § 2 parallel writer checklist: a brief names the module-map row
  that owns each path, and each writer runs `lint-module-map.sh` and the whole `npm test` before reporting done
  (retro kit-gzip-bundle-writer-briefs Δ3).
