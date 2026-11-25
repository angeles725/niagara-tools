### Added — lint-subscribe-js: a `new baja.Subscriber()` a module .js file never unsubscribes (G76)

- `toolbelt/lint-subscribe-js.sh` (rule `go/internal/lint/rules/subscribeleakjs`): the JavaScript twin of
  `lint-subscribe-without-unsubscribe`. WARN (FAIL under `--strict`) a `new baja.Subscriber()` in a module `.js`
  file (`*.min.js`, `build/`, `node_modules/` skipped) whose file has no `.unsubscribeAll(` or `.unsubscribe(`
  call, and the first `.subscribe(` on a Subscriber-named receiver in a file with no teardown and no
  `new baja.Subscriber(` site. A promise finally, a detach/close hook, a widget `doDestroy` and a load of
  `bajaux/mixin/subscriberMixIn` (its destroy unsubscribes) all pair a site; `detach()` alone does not, it removes
  handlers only. Waive with `// lint-subscribe-js: <reason>`; a bare marker is a FAIL (`waiver-no-reason`).
  Evidence: BajaScript `Subscriber.js:30,336,632,758`, the getting-started tutorial's
  `unsubscribeAll(); detach();`, `Widget.js` `doDestroy`, `subscriberMixIn.js:304-317` (API survey 2026-11-24
  transports, row K6).
- `types/dashboard.md` (BajaScript subscribe bullet): the owed-lint note now points at the lint.
