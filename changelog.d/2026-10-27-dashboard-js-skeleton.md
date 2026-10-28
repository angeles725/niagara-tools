### Changed — the dashboard scaffold emits a split rc/js skeleton

- `scaffold-module.sh --type dashboard` emits `rc/js/config.js`, `js/services/api.js` (the only `fetch`, with the
  XHR header and an abort timeout), `js/store.js` (poll loop, state, subscribe, reload after `reloadAfterMs`
  without a good read), `js/components/status.js`, `js/pages/home.js` and `js/main.js`; `index.html` loads them in
  order and holds no inline script. The module map lists them in `frontend-data` and `frontend` rows. A fresh
  dashboard no longer WARNs `rc-scan` `fetch-no-signal` or `lint-spa-poll-no-recovery`.
