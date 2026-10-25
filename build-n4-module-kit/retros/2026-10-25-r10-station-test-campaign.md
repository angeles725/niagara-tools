<!-- review-status: folded -->
<!-- module: kit -->
<!-- deltas: 4 -->
# 2026-10-25 · kit · r10-station-test-campaign

**Session**: niagara-tools quality round 10, the client station-test campaign: client A's source-text tests
replaced by real `BTestNgStation` station tests run with `run-niagara-test run`, branch
`r10/station-campaign-retro` from origin/main 1551c6bb
**Delta count**: 4

## What happened
Six client A source-text tests (assertions on the Java source, not on behaviour) were replaced by real
`BTestNgStation` station tests, each run on a disposable copy of an install:
- a compressor-control logic module on a 4.14 install copy: 11/11;
- a cold-room logic module on a 4.15 install copy: 14/14;
- client A's dashboard facade module on a 4.15 install copy: 3/3, with a per-slot `checkLink` (a `LinkCheck` per
  slot) and real `BLink` propagation in the station;
- the fix of a latent defect the new tests exposed: 3/3.

Each test is pinned by a named production mutation. One boot with one mutation applied showed RED on the class
that mutation targets and GREEN on every other class, so each RED belongs to its class and the classes do not
lean on each other.

Costs measured on the way: the install copy about 5 s, the first boot of a copy about 4.5 min, one batched boot of
every class of a module 7-10 min. A test that checks an output once right after a synchronous action raced the
control cycle; it needed a sampling window (a `holds` helper that keeps checking over the window) to cover one
cycle. Station tests also needed a unique component name per test (the station outlives a single test) and waits
sized to the component's debounce plus one tick, about 20 s for one module.

One client gradle root hardcoded the niagara gradle plugin version, so it could not build against an install
that ships another one: the runner's `-PniagaraPluginVersion` had no effect. The root now reads
`-PniagaraPluginVersion`, keeping the old literal as its default; that fix stays on a local client branch.

The campaign also met the runner's known traps again. They are already in the kit docs and are cited below,
not restated as deltas.

## Already in the kit (cross-references, no new delta)
- The runner itself, `run-niagara-test run` (#556) `[ev: retro r10-run-niagara-test-run Δ1]`.
- **User-home trap and incident (D3).** `cmd /c "set "niagara_user_home=…" && …"` from WSL never reached the NRE
  (WSL re-escapes the inner quotes); the NRE reads `niagara_user_home` (`NreWin32::getNiagaraUserHome`,
  `_dupenv_s`) and otherwise falls back to `%USERPROFILE%\Niagara<maj>.<min>\<brand.id>`, so it rebuilt the real
  user home's registry and signers cache. Fix `WSLENV=niagara_user_home/w`, proven on a non-booting
  `nre.exe -version` probe; the gate snapshots the real homes before and after and requires `etc\` and
  `security\` in the scratch home (#557) `[ev: retro r10-rnt-run-live-fixes Δ1]` `[ev: retro r10-rnt-run-live-fixes Δ2]`.
- **D4.** A non-elevated caller reads an empty path and command line for LocalSystem service stations; they are
  attributed through `Win32_Service` `PathName`/`ProcessId`, and an unattributable one is refused (#557)
  `[ev: retro r10-rnt-run-live-fixes Δ3]`.
- **Verification mode.** 4.14 ships `niagara.moduleVerificationMode=low`, 4.15 leaves it commented out
  (`medium` applies), so a throwaway-signed test jar is `invalid test target(s)` with exit 0; the user trust store
  cannot be scripted (its password derives from the keyring), so `--module-verification low` on the disposable
  copy only, guarded by the install's sha256 check (#559) `[ev: retro r10-rnt-module-verification Δ1]`
  `[ev: retro r10-rnt-module-verification Δ2]`.
- **test.exe output.** The summary is on stderr, stdout holds only thread dumps; exit 0 green, -1 on any failure
  (the earlier "exits 0 on failures" was `%ERRORLEVEL%` parse-time expansion; use `/v:on` and `!ERRORLEVEL!`);
  `testng-results.xml` is the authority (#555) `[ev: retro r10-run-niagara-test-testng Δ1]`
  `[ev: retro r10-run-niagara-test-testng Δ3]`.
- **Fox.** The Windows NRE never binds Fox to loopback only, so a loopback-only station test turns Fox off and
  tolerates `Fox service did not start in time` (#555) `[ev: retro r10-run-niagara-test-testng Δ4]`.

## Evidence
- the station-test route and its traps as documented `[ev: types/moduleTest.md § Running from WSL]`
  `[ev: types/driver-authoring.md § 12.1]`
- the plugin-version property the runner passes `[ev: go/internal/niagaratest/run.go]`

## Proposed kit deltas (propose-never-apply)
| Δ | Delta | Target file / § | Token |
|---|---|---|---|
| Δ1 | Replace a source-text test with a real `BTestNgStation` station test (a facade's link checks through a per-slot `checkLink` and real `BLink` propagation); pin each test with a named production mutation, and run one boot per mutation: RED on the class it targets, GREEN on the rest | `types/moduleTest.md` § Running from WSL | `[ev: types/moduleTest.md § Running from WSL]` |
| Δ2 | A client gradle root that hardcodes the niagara gradle plugin version cannot build against an install that ships another one; read `-PniagaraPluginVersion` and keep the old literal as its default | `types/moduleTest.md` § Running from WSL | `[ev: go/internal/niagaratest/run.go]` |
| Δ3 | Budget a station-test run: copy about 5 s, first boot about 4.5 min, a batched boot of every class 7-10 min; a check of an output right after a synchronous action needs a sampling window that covers one control cycle | `types/moduleTest.md` § Running from WSL; § TEST-G2 | `[ev: types/moduleTest.md § Running from WSL]` |
| Δ4 | A station outlives a single test: give each test its own component names, and size every wait to the debounce plus one tick (about 20 s for one module) | `types/moduleTest.md` § Known gotchas | `[ev: types/moduleTest.md § Known gotchas]` |

## Lessons
- A test of the source text proves the text, not the behaviour; one mutation per boot proves which class a RED
  belongs to.
- A tool flag a build script hardcodes away is silently ignored: read the property, keep the old value as default.
