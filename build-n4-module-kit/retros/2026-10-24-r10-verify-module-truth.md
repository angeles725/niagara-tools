<!-- review-status: pending -->
<!-- module: kit -->
<!-- deltas: 3 -->
# 2026-10-24 · kit · r10-verify-module-truth

**Session**: niagara-research quality round 10 (Ayudante1), branch `r10/verify-module-truth`
**Delta count**: 3

## What happened
Phase B of the verify-module Go port: the nine false results listed by phase A (retro go-verify-module) were
fixed, each in its own RED-first commit with a declared-difference golden, plus two found on the way (a decimal
`--plano` viewBox, an exact `types` class lookup). Every unreadable or unparseable input now ends the run with
exit 3 `cannot read <path>` and no verdict; module.xml/module-include.xml read through xmlsrc, gradle.kts through
ktssrc, Java through the javasrc stripped view and PropertyScan, and hits are matched on bytes.

## Proposed kit deltas (propose-never-apply)
- Δ1 A truth fix that tightens an operand must check the format's own grammar first: the first cut made a
  decimal SVG viewBox a usage error, which would have left real dashboards uncheckable; SVG allows decimals.
- Δ2 A bash gate's comparison carried no tolerance (exact integer cross-products); moving it to floats needs a
  tolerance chosen and pinned in a golden, never inherited.
- Δ3 The golden harness has no update flag: a declared-difference golden is copied from the `go test` "got"
  block, so each declared golden is reviewed line by line in its commit.
