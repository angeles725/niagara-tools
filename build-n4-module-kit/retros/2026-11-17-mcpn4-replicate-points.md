<!-- review-status: pending -->
<!-- module: kit -->
<!-- deltas: 6 -->
# 2026-11-17 · kit · mcpn4-replicate-points

**Session**: a client B field job that replicated the same Modbus points onto many devices with
`n4_copy_component` and `n4_set_slot`, one confirmation token per write.
**Delta count**: 2

## What happened
The job needed the same points on every device of a trunk. `n4_copy_component` copies one child
under one parent, so each device and each address change was its own dry run and token: 630 calls
in 39 minutes for 112 points, and the process write budget ran out before the save. Two of the
replicated writable points carried a non-null fallback (a power point at 0 = OFF, a setpoint at
200): every copy would have commanded its device with it on first communication, and nothing in the
dry run said so.

## Evidence
- Field retro of 2026-11-16 (client B), rows U10 (630 calls / 39 min / 112 points, budget exhausted)
  and U18 (fallback values pending a decision).
- Lab proof on the 4.15.6.20 lab station: see the PR description (batch ids).

## Proposed kit deltas (propose-never-apply)
| Δ | Delta | Target file / § | Token |
|---|---|---|---|
| Δ1 | APPLIED 2026-11-17. `n4_replicate_points`: N x M adds under one token, per-target `copies[]` name/address overrides, one budget write per add, refused up front when over `write_budget.remaining` | `mcp-n4-kit/README.md` write tools; `mcp-n4-kit/skill/SKILL.md` bulk | `[ev: retro mcpn4-replicate-points Δ1]` |
| Δ2 | APPLIED 2026-11-17. Copy/replicate dry runs note writable points with a non-null fallback | `mcp-n4-kit/README.md` `n4_copy_component`; `mcp-n4-kit/skill/SKILL.md` | `[ev: retro mcpn4-replicate-points Δ2]` |
| Δ3 | APPLIED 2026-11-17. A replication keeps one write of `--max-writes` for its own `n4_rollback` (independent verifier of #772: one that spent the last write could be rolled back only after a restart) | `go/internal/mcpn4/writeops/replicate.go` | `[ev: code go/internal/mcpn4/writeops/replicate.go]` |
| Δ4 | APPLIED 2026-11-17. Targets that differ only by a trailing `/` are one target (verifier: both copies landed under one parent, verdict still verified) | `go/internal/mcpn4/writeops/replicate_args.go` | `[ev: code go/internal/mcpn4/writeops/replicate_args.go]` |
| Δ5 | APPLIED 2026-11-17 (`TestAnInDoubtReplicationRollsBackOnlyTheCopiesTheStationAccepted`, `TestAnInDoubtReplicationWithNoCopyAcceptedStillRefuses`, `TestAnInDoubtReplicationWithoutAResultRecordKeepsTheInDoubtRefusal`; review follow-up R1-001/R3-001/R3-002: `TestAnInDoubtReplicationKeepsACopyALaterBatchWroteUnder`, `TestAnInDoubtReplicationWithAnInconsistentFailedOpKeepsTheRefusal`). A mid-batch failure leaves up to 199 copies in doubt and `n4_rollback` refuses an in-doubt batch; offer a rollback of the copies the read-back finds (per-copy `unverified`, reviewer R4-002) | `go/internal/mcpn4/writeops/replicate_readback.go` | `[ev: code go/internal/mcpn4/writeops/replicate_readback.go]` |
| Δ6 | APPLIED 2026-11-17 (`TestAReplicationOfNAddsNeedsNWrites`, `TestARollbackRunsWithTheBudgetSpentAndSpendsNoWrite`, `TestARollbackRunsWithTheBudgetSpent`, blackbox `TestAReplicationOfNAddsNeedsNWritesAndItsRollbackNone`, `TestTheSaveAndTheRollbackRunAfterTheBudgetIsSpent`, `TestACopyAtTheRegisterOfAPointAlreadyOnTheTargetIsNoted`, `TestAReadOnlyPointAlreadyOnTheTargetDoesNotHoldTheRegister`). Rule: every `n4_rollback` is outside `--max-writes`, like `n4_save_station` (U7): the budget caps forward changes and recovery must always be reachable; each batch can be rolled back once, so it stays bounded. A replication of N adds needs N writes; this supersedes #777's one write kept for the rollback (charging the rollback N and keeping N would have halved the largest replication at the default budget to 100 adds, below the 120 adds (15 devices x 8 points) of a field job). The dry run reads each target once, four levels deep, and notes a copy at the register of a writable Modbus point already there. Rolling back an N-add replication is charged one write; the shared-register note does not read points already on a target device | `go/internal/mcpn4/plan/budget.go` | `[ev: code go/internal/mcpn4/plan/budget.go]` |
| Δ7 | APPLIED 2026-11-17. The register note took Hex:-1 (the address the station resets a duplicate to) as a held register and flagged copies at -1 (verifier probe G2 of #798); -1 is now never held or shared (`TestAnUnassignedAddressDoesNotHoldARegister`) | `go/internal/mcpn4/writeops/replicate_registers.go` | `[ev: code go/internal/mcpn4/writeops/replicate_registers.go]` |
| Δ8 | Open. Other register-note advisories of the #798 verifier: the same register in another address format is not matched (G3), a coil and a holding register at the same number are flagged (G4), points in subfolders of a target are not read (G5); and an in-doubt rollback removes a same-name same-type copy recreated by hand (P1: compare the handle from the add reply) | `go/internal/mcpn4/writeops/replicate_registers.go` · `rollback_replica.go` | `[ev: code go/internal/mcpn4/writeops/replicate_registers.go]` |
