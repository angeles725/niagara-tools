<!-- review-status: pending -->
<!-- module: kit -->
<!-- deltas: 6 -->
# 2026-11-10 · kit · mcpn4-history-batch-tools

**Session**: client A live history rollout (35 interval history extensions on a 4.14 JACE) plus the client A dashboard module 2.0 signed build
**Delta count**: 6

## What happened
Configuring 35 history extensions through mcp-n4 took about 8 tool calls per point (create, three `n4_set_slot` writes, each a dry run plus an execute), and a dropped tunnel mid-batch killed every pending token and left 10 extensions half-configured. `n4_add_history` now creates each extension already configured, one dry run and one token for up to 50 points. The same session's signed module build ended at `build.sh` exit 53 (report INCOMPLETE: no schema-risk baseline for an already-deployed module) after schema-risk had been run by hand against the baseline commit, and signing took several rounds because the exported certificate was the CA and the Workbench private key export (PBES2 + AES-256-GCM) is unreadable by OpenSSL 3.

## Evidence
- `n4_add_history` with tests `writeops.TestAddHistoryPlansOneConfiguredAddPerPointInOneBatch`, `writeops.TestAddHistoryRefusals`, `writeops.TestAddHistoryReadbackComparesEveryExtensionAndUsesTheAssignedName`, `writeops.TestResolveHistoryName`; guard-drop mutations of the repeat refusal and the mismatch verdict turned them red `[ev: 3d37c6d2]`
- `n4_set_slot` omitted-default typing and `baja:Format` / `baja:RelTime` `[ev: 67df67c7]`
- Live: 10 extensions verified enabled at 5 min through `n4_set_slot`; the tunnel dropped mid-batch (`Connection refused`), the next `n4_connect` made a new session and the old tokens were dead `[ev: 67df67c7]`
- Signed build: jars signed, verify-module ALL PASS, then `build.sh` exit 53, report-module INCOMPLETE `[ev: 67df67c7]`

## Proposed kit deltas (propose-never-apply)
| Δ | Delta | Target file / § | Token |
|---|---|---|---|
| Δ1 | `n4_add_history` also sets capacity (`historyConfig/capacity`) so a dashboard's 7 d / 30 d views are covered; today the default 500 records is about 41 h at 5 min | `mcp-n4-kit/README.md` § `Writing safely` | `[ev: 3d37c6d2]` |
| Δ2 | Add `n4_read_history` (last N records of a history) so a rollout proves records arrive, not only configuration | `mcp-n4-kit/METHODOLOGY.md` § `4. Mandatory session checklist` | `[ev: 3d37c6d2]` |
| Δ3 | After a server-session loss, offer to re-plan the pending dry runs (same args, new token) instead of only refusing the old token | `mcp-n4-kit/METHODOLOGY.md` § `3. Live rules (B1199)` | `[ev: 67df67c7]` |
| Δ4 | APPLIED 2026-11-17 (tests `TestSchemaBaselineRefFeedsTheReport`, `TestSchemaBaselineRefIsRemovedWhateverTheExit`, `TestSchemaBaselineRefRefusals`, `TestSchemaBaselineRefSkipsAnArtifactNewSinceTheRef`, `TestNoSchemaBaselineRefLeavesTheTreeAlone`, `TestSchemaBaselineRefReplacesADanglingLeftover`, `TestSchemaBaselineRefKeepsOperatorLinks`, `TestSchemaBaselineRefToolFailures` in `go/internal/buildmodule`). `build.sh` accepts a schema-risk baseline from a git ref (`--schema-baseline-ref <commit>`) so an already-deployed module whose deployed jars are not at hand does not end at exit 53 after a manual SAFE run | `build-n4-module-kit/build-verify.md` § `Schema-risk deploy workflow` | `[ev: 67df67c7]` |
| Δ5 | A signing doctor refuses a CA certificate (CA:TRUE, no code-signing usage) before the build and converts the Workbench PBES2/AES-GCM key export with Java, not OpenSSL | `build-n4-module-kit/build-verify.md` § `Signing per deploy target` | `[ev: 67df67c7]` |
| Δ6 | A servlet write template checks CSRF and the user's permissions on the target component (never on the user itself), with a lint rule for `getPermissionsFor` on a non-target | `build-n4-module-kit/BUILD-LOOP.md` § `7. Retro + close (HARD close gate — not optional)` | `[ev: 67df67c7]` |

## Lessons
- A recipe that needs N ordered writes per object belongs in one tool with one plan, one token and one read-back, not in N separate confirmations.
- A batch over a flaky link must leave a state that one query can read back (BQL over the extensions), so recovery starts from facts.
- A history extension created through BOX starts disabled with defaults; the add op can carry its configured slots so it is never half-configured.

---
**Status**: PENDING — INDEX row generated from the header: `| 2026-11-10-mcpn4-history-batch-tools.md | kit | 2026-11-10 | pending | 6 |`
