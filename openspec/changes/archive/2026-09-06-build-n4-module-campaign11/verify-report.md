```yaml
schema: gentle-ai.verify-result/v1
evidence_revision: sha256:48e34eb35a1417e6387d64bf5eb8d4e66779efec492ef2c00b7526042b3a5ae7
verdict: pass_with_warnings
blockers: 0
critical_findings: 0
requirements: 56/56
scenarios: 32/32
test_command: bats tests/ (C9_CLIENT_ROOT, C9_CLIENT_REPO, C8_CLIENT_REPO unset; tests/lib/client-root.bash default)
test_exit_code: 0
test_output_hash: sha256:6e8749de7e69e903c3899c73b0fe1b917720d7a2ea813b959c2b6f24d8d75ba6
build_command: bash build-n4-module-kit/toolbelt/sweep-fold-audit.sh --strict build-n4-module-kit/retros/INDEX.md build-n4-module-kit
build_exit_code: 0
build_output_hash: sha256:c4dcf6eb53bdf02cdbedba73885c0b41f3205f34fdab1d3c0368d9bb091a4753
```

# Verification Report — build-n4-module-campaign11

**Kit**: v0.22.0 @ `66123a2` (tasks ticked `e55b369`) | **Range**: `dab0807..66123a2` | **Verdict**: PASS WITH WARNINGS — 0 CRITICAL, 1 WARNING, 2 SUGGESTIONS
**Note**: the executor could not persist this file because `gentle-ai sdd-verify-validate` 2.6.0 does not document the `evidence_revision` grammar (`sha256:<64 lowercase hex>`; integers for `blockers`/`critical_findings`); the lead built the admitted envelope above (`valid: true`) and recorded an occurrence on gentle-ai issue #4089.

## Run summary

| Command | Exit | Result |
|---|---|---|
| `bats tests/` (env unset; `tests/lib/client-root.bash` default) | 0 | 454/454 PASS |
| `C11_CLOSE=1 C11_CLOSE_COMMIT=66123a2 bats tests/c11-close.bats` | 1 | 12/13 — test 13 CLOSE-harness-run pending (C9 carry-over) |
| `sweep-fold-audit.sh --strict` | 0 | 93 folded / 93 cited / 0 uncited |
| `lint-guard-pins.sh --strict .` | 0 | 15 MATCH / 0 WARN over 10 lint scripts |
| `bats tests/kit-links.bats` | 0 | 8/8 |
| `shellcheck toolbelt/*.sh toolbelt/lib/*.sh scripts/*.sh` | 0 | 0 issues |
| `lint-write-path.sh --strict` main-ff1b659 CompPan-rt | 1 | 0 DRIFT / 5 STALE (unchanged) |
| `lint-write-path.sh --strict` main-00e7118 CompPan-rt | 0 | 0 DRIFT / 0 STALE |
| `lint-delays.sh` ColdRoomPan-rt @ ff1b659 | 0 | clean, BDefrostController absent |
| attribution trailers dab0807..66123a2 | — | 0 |
| conflict markers (repo-wide) | — | 0 |

## RED tip byte-identity

parser-oneliner `d88af78`, golden-parser `ed2088f`, concept-drift `77352a7`, client-root `54078f6`, guard-pins `452e7d4`, close `f1a8765` — all byte-identical to the merged bats.

## PR verdicts

| PR | Merged at | Second read | Verdict |
|---|---|---|---|
| PR1 T1 shared method-boundary parser (#99) | 127f25a | ad2121b69 PASS | PASS (3 OBSERVED flips; I2/I3/I5 defensive, B832-G3/G4 named seeds) |
| PR2 T3 concept-row DRIFT (#100) | 2aa32fd | 7e9669480 PASS | PASS |
| PR3 T2 client-root lib (#101) | 53efff7 | 654535f64 PASS | PASS (LD5 + c8-close SC1-smoke retargeted to the correct verdict) |
| PR4 T4 lint-guard-pins (#102) | 07d2fc7 | fdc87b4b0 PASS | PASS (K13 breach reverted 948afa8; T4-smoke re-pinned 452e7d4) |
| PR5 close (#103) | 66123a2 | 5fd3f6159 PASS | PASS / 1 PENDING (CLOSE-harness-run) |

Design validator eae61fb27 7/7; tasks read c620af37d 5/5. Spec compliance: 56/56 requirements, 32/32 scenarios (T1 17, T2 10, T3 11, T4 9, close 9). All 59 task items ticked.

## Issues

- **WARNING W1** — CLOSE-harness-run owed: the Windows niagaraTest session (CRA1/2/3-live, CPB5, R14 lockout + AuditEvent) has never run; `qa/c9-harness-run.md` does not exist. Out of C11 scope (proposal §2.2); owed by Cristian; gates W2 P1-P5.
- **SUGGESTION S1** — B832-G2 (`/* */` strip in Case-B) has no biting fixture; accepted at design D1h, pinned by the 3×3 baselines (C12 seed).
- **SUGGESTION S2** — legacy `NAMED MUTATION` prose in non-lint scripts (slot-coverage.sh, verify-module.sh) is outside T4's scope per D4b (C12 seed).

## Noted events (all resolved)

1. I2/I3/I5 fragment invariants are defensive with reachable-but-absent shapes (B832-G3/G4) — not OBSERVED mutations; tasks 1.10 and the PR1 retro record it.
2. K13 breaches reverted: a worker trimmed `tests/lint-timers.bats` for PR4 (restored 948afa8; QA re-pinned the smoke without a line number) and edited `tests/c11-close.bats` in the close commit (QA re-issued the hunk as f1a8765; branch rebuilt).
3. Fragment-merge slip on PR2's rebase pushed conflict markers for minutes (fixed 5beff22); real C8-archive markers found and resolved (7053907); CLOSE-no-conflict-markers added to the close gate.
4. lint-guard-pins lands 15 `# Mutation:` lines (design estimated 14): lint-write-path gained WP-drift-decoy after PR2.

## Pending (not blocking archive)

| Item | Owner |
|---|---|
| CLOSE-harness-run — Windows niagaraTest session | Cristian |
| Tunnel PRs #1-#3 merge; deploy chain (2.0.7/2.0.3/2.1.1 then C9 jars) | Cristian |
| P1-P5 product seeds | gated on the above + Cristian's answers |
| C12 seeds S1-S4 (niagara-research 850791f12, B833 7d6250b40; QA REDs 5503c14/881da39/7cfb3a7 left on origin) | on hold by Cristian's order |

## Key Learnings

1. The validator's grammar for `evidence_revision` is `sha256:<64 lowercase hex>` and integers for `blockers`/`critical_findings`; it is undocumented in 2.6.0 (gentle-ai #4089, fix PR #4171 unmerged).
2. A real-tree smoke that asserts a FAIL caused by a known bug goes green for the wrong reason once the bug is fixed; LD5 and c8-close SC1-smoke were the concrete cases.
3. Three parser copies were three awk invocation mechanisms; only a function-only fragment fits all of them.
4. I2/I3 are defensive under PEAK depth on the current corpus but have reachable shapes; they are named C12 seeds, not silent assumptions.
5. Cross-file line-number pins pressure workers into editing QA REDs; pin by name or compute the line.
