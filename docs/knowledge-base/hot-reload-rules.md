# Hot-Reload Rules — Niagara N4

Decision table for whether a deploy requires a full station restart
or only a browser hard-reload. Sourced from production observation (2026-05-17).

---

## Decision table

| Change type | Files changed | Hot-reload? | Action required |
|---|---|---|---|
| JS / CSS / HTML assets | `*-ux/src/rc/js/**`, `*-ux/src/rc/css/**`, `*-ux/src/rc/*.html` | Yes | Browser hard-reload only (with cache-buster bump) |
| Java BComponents (rt) | `*-rt/src/com/**/*.java` | No | **Station restart required** |
| Java servlets / handlers (ux) | `*-ux/src/com/**/*.java` | No | **Station restart required** |
| Both Java and assets | Any `.java` + any asset file | No | **Station restart required** |

**Rule of thumb**: the restart decision follows the FILE EXTENSION (`*.java` = restart),
not the subproject name or the build mode. Mode B (`ux-only`) can require a restart
when it includes servlet changes.

---

## Why `chihuahua-ux` is ambiguous

`chihuahua-ux` packages TWO distinct categories of source in a single jar:
- `chihuahua-ux/src/com/angeles/chihuahua/ux/*.java` — Java servlets and helpers
  (backend; classloader must restart to load new bytecode)
- `chihuahua-ux/src/rc/js/app/*.js` — ES5 frontend assets
  (hot-served; no classloader involved)

Saying "I only changed the ux subproject" does NOT tell you whether you need
a restart — you must check which FILES inside the subproject changed.

**Evidence (2026-05-17)**: A fix touching only `AlarmsPage.js` (one line) required
no restart. The same SDD's original apply included a new servlet handler
(`handleAlarmAckAll`) → restart required despite being in the ux subproject.

---

## Cache-buster rule (JS/CSS deploys)

After deploying a ux jar that changed JS or CSS assets, bump the `?v=N` query
string in `index.html` to invalidate the browser cache:
```html
<!-- Before -->
<script src="app/bundle.js?v=20260517"></script>
<!-- After bump -->
<script src="app/bundle.js?v=20260518"></script>
```
Without the bump, browsers serve the cached old version (HTTP 304).
`ng-deploy.sh` can verify this automatically when `BUILD_ID` is set in `.env.local`.

---

← Back to [GOTCHAS index](../GOTCHAS.md)
