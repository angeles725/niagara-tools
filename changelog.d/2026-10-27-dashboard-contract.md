### Added — dashboard scaffold: a route handler class and the ux-rt API contract

- A fresh dashboard's servlet delegates `GET /api/status` to `ux/route/StatusHandler` (router + one handler class
  per route group, DUX1).
- `<M>-ux/src/rc/js/contract.json` names each route's response fields; `StatusHandlerTest` (pure JUnit) checks
  the servlet side and `srcTest/rc/spec/<M>ContractSpec.js` (node, offline) checks that every field
  `services/api.js` reads is declared (types/dashboard.md DUX-CONTRACT1). The status component now prints the
  `status` field instead of the whole JSON body.
