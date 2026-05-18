# Smoke Checklist — ng-deploy.sh

Manual integration tests for station-connected deploy verification.
Run before and after each real station deploy.
Automated unit tests: `bats tests/ng-deploy.bats`

---

## Mode A — Full deploy (rt + ux jars)

Use when: Java changes in either rt or ux subproject, OR both.

### Prereqs
- [ ] `.env.local` populated in consumer module root (all 6 required vars + `EXPECTED_RT_TYPES` + `EXPECTED_UX_TYPES`)
- [ ] Station is running and accessible in Workbench
- [ ] Both `chihuahua-rt.jar` and `chihuahua-ux.jar` present in `STATION_MODULES_DIR`
- [ ] bats suite green: `bats tests/ng-deploy.bats`

### Run
```bash
cd /path/to/chihuahua
/path/to/niagara-tools/scripts/ng-deploy.sh --mode A
```

### Expected stdout
```
[ng-deploy] backup: _backups/chihuahua-pre-YYYYMMDD-HHMMSS.tar.gz
[ng-deploy] backup ok: _backups/chihuahua-pre-YYYYMMDD-HHMMSS.tar.gz
[ng-deploy] build: mode A — ...
[ng-deploy] build ok
[ng-deploy] copy ok
[ng-deploy] verify ok (chihuahua-rt.jar: 9/9)
[ng-deploy] verify ok (chihuahua-ux.jar: 2/2)
[ng-deploy] station restart required (Java classes deployed)
```

### Post-deploy verification
- [ ] Open Workbench → Platform → Station → Restart Station
- [ ] After restart, navigate to the module's component palette
- [ ] Confirm all expected component types load without errors
- [ ] Check `META-INF/module.xml` type count: `unzip -p chihuahua/chihuahua-rt/build/libs/chihuahua-rt.jar META-INF/module.xml | grep -c "<type"` == `EXPECTED_RT_TYPES`

### Rollback (if verify fails or station errors after restart)
```bash
# 1. Find the backup
ls -lt _backups/chihuahua-pre-*.tar.gz | head -3
# 2. Extract the backed-up jars
tar -xzf _backups/chihuahua-pre-YYYYMMDD-HHMMSS.tar.gz -C /tmp/rollback/
# 3. Copy old jars back
cp /tmp/rollback/modules/chihuahua-rt.jar "$STATION_MODULES_DIR/"
cp /tmp/rollback/modules/chihuahua-ux.jar "$STATION_MODULES_DIR/"
# 4. Restart station
```
See [docs/GOTCHAS.md](../docs/GOTCHAS.md) for root cause guidance.

---

## Mode B — UX-only deploy (ux jar only)

Use when: Only JS, CSS, HTML assets changed (no Java changes in any subproject).

### Prereqs
- [ ] `.env.local` populated (all required vars + `EXPECTED_UX_TYPES`)
- [ ] `chihuahua-ux.jar` present in `STATION_MODULES_DIR`
- [ ] Cache-buster bumped in `index.html` if JS/CSS changed (otherwise browser won't see new assets)

### Run
```bash
cd /path/to/chihuahua
/path/to/niagara-tools/scripts/ng-deploy.sh --mode B
# Optionally with BUILD_ID for cache-buster verification:
BUILD_ID=20260518a /path/to/niagara-tools/scripts/ng-deploy.sh --mode B
```

### Expected stdout
```
[ng-deploy] backup ok: _backups/chihuahua-pre-YYYYMMDD-HHMMSS.tar.gz
[ng-deploy] build ok
[ng-deploy] copy ok
[ng-deploy] verify ok (chihuahua-ux.jar: 2/2)
[ng-deploy] verify ok (cache-buster ?v=20260518a found)   ← only if BUILD_ID set
[ng-deploy] browser hard-reload only (no station restart needed)
```

### Post-deploy verification (browser cache)
- [ ] Open the module's web UI in the browser
- [ ] Open DevTools (F12) → Network tab → disable cache checkbox
- [ ] Hard-reload (Ctrl+Shift+R / Cmd+Shift+R)
- [ ] Confirm JS/CSS responses return HTTP **200** (not 304)
  - 304 = browser served cached old version — bump `?v=N` and redeploy
- [ ] Verify new UI changes are visible

### Rollback
```bash
# Same as Mode A but only the ux jar needs restoring
cp /tmp/rollback/modules/chihuahua-ux.jar "$STATION_MODULES_DIR/"
# No station restart needed — browser hard-reload sufficient
```

---

## Mode C — RT-only deploy (rt jar only)

Use when: Only Java BComponent changes in the `*-rt` subproject (no ux changes).

### Prereqs
- [ ] `.env.local` populated (all required vars + `EXPECTED_RT_TYPES`)
- [ ] `chihuahua-rt.jar` present in `STATION_MODULES_DIR`
- [ ] Station running

### Run
```bash
cd /path/to/chihuahua
/path/to/niagara-tools/scripts/ng-deploy.sh --mode C
```

### Expected stdout
```
[ng-deploy] backup ok: _backups/chihuahua-pre-YYYYMMDD-HHMMSS.tar.gz
[ng-deploy] build ok
[ng-deploy] copy ok
[ng-deploy] verify ok (chihuahua-rt.jar: 9/9)
[ng-deploy] station restart required (Java classes deployed)
```

### Post-deploy verification
- [ ] Restart station (Platform → Restart Station in Workbench)
- [ ] After restart, confirm the module version in the station's module list
  (Workbench → Platform → About → Modules → find `chihuahua-rt`)
- [ ] Run a quick BQL smoke query to confirm component types are accessible:
  ```
  SELECT * FROM control:BChiUp limit 1
  ```
- [ ] Verify type count in deployed jar: `unzip -p chihuahua/chihuahua-rt/build/libs/chihuahua-rt.jar META-INF/module.xml | grep -c "<type"` == `EXPECTED_RT_TYPES`

### Rollback
```bash
cp /tmp/rollback/modules/chihuahua-rt.jar "$STATION_MODULES_DIR/"
# Station restart required after rollback
```
