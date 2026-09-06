# qa/ — campaign-9 verify terrain (QA session, VERIFY authority)

| File | What |
|---|---|
| `c9-verify-runbooks.md` | Execute-only runbook per PR (PR1-PR13 + R14): fresh detached worktree, base/hygiene, gate, real-tree smokes with expected rows, named mutations, harness-only exclusions, version key, trailer grep, BLESS line, `tip == blessed`. |
| `c9-mutations.tsv` | Named-mutation table (one row per mutation, content-anchored). Input to `mutate.sh`. |
| `mutate.sh` | Applies a table row to a THROWAWAY detached worktree, runs the suite, prints OBSERVED / NO-FLIP / ANCHOR-MISSING / NO-OP-MUTATION, keeps the verbatim mutant output, restores. Self-tested by `tests/qa-mutate.bats`. |
| `junit-run.sh` | Compiles + runs one plain-JUnit client test class offline in WSL (Baja-free sources auto-selected). |
| `c9-harness-procedure.md` | The Windows `niagaraTest` run for the harness-only pins (CRA1/2/3 live, CPB5, R14 lockout + AuditEvent): mechanism, what it needs from Cristian, the three harness classes, run record format. |
| `c9-retro-pins.md` | Pre-staged retro pins (RP1-RP13) to fold at close. |

Rules: mutations only on detached throwaway worktrees (the harness refuses a branch HEAD); harness-only
pins never count as WSL-green; the RED file is the contract when prose disagrees.
