<!-- review-status: pending -->
<!-- module: kit -->
<!-- deltas: 3 -->
# 2026-11-16 · kit · kitlinks-advisories-t32

**Session**: row T32 of the 2026-11-15 coordination-stretch retro: the native-review advisories of niagara-tools #717,
which added the kit-links step to `scripts/ci-local.sh --changed`.
**Delta count**: 3

## What happened
Four advisories. A changelog.d-only change was never tested to run kit-links. The bats provisioning result was not
checked: `run_kit_links` runs in an errexit-ignored context and `provision_bats` ends on a printf, so a failed tar left
the bats already on PATH to run. The kit-state step ran the checkout's `toolbelt/kit-state.sh` while kit-links.bats in
the same snapshot runs the snapshot's own. The change detection had no failure path of its own: with `--range` a
failed `git diff` read as "changed" only through `!`, and the snapshot then reported a misleading "cannot snapshot";
without `--range` a failed `ls-files` read as "no kit change". Now a failed detection is one named line and exit 3, a
provisioning that fails or leaves another bats version first on PATH is one line and exit 3, and kit-state runs from
the snapshot (the launcher the push sends; go/ stays the checkout's either way).

## Evidence
- RED before GREEN: CL34 (stubbed provisioner returning 1), CL35 (provisioner leaving the wrong version), CL36 (bad
  range on `run_kit_links`) and CL37 (pushed launcher differs from the checkout's) failed on origin/main; CL33
  (changelog.d only) already passed, a coverage gap pinned by its mutation. `[ev: tests/ci-local-kitlinks.bats]`
- Through the CLI a bad `--range` stops in `run_changed` (exit 3) before kit-links, so CL36 pins the guard on
  `run_kit_links` itself. `[ev: scripts/lib/ci-local-changed.sh]`
- Mutations in a scratch copy, not committed: dropping changelog.d from the pathspec (CL33), the provisioner status
  check (CL34), the pinned-version check (CL35), the detection error path (CL36) and running the checkout's
  launcher (CL37) each turn their test red. `[ev: tests/ci-local-kitlinks.bats]`

## Proposed kit deltas (propose-never-apply)
| Δ | Delta | Target file / § | Token |
|---|---|---|---|
| Δ1 | APPLIED 2026-11-16. A kit-links bats that cannot be provisioned at the ci.yml pin is one stderr line and exit 3 | `scripts/lib/ci-local-kitlinks.sh` | `[ev: scripts/lib/ci-local-kitlinks.sh]` |
| Δ2 | APPLIED 2026-11-16. A git error while detecting a kit doc change is reported (exit 3), never read as no change | `scripts/lib/ci-local-kitlinks.sh` | `[ev: scripts/lib/ci-local-kitlinks.sh]` |
| Δ3 | APPLIED 2026-11-16. The snapshot's own kit-state launcher materializes its generated state | `scripts/lib/ci-local-kitlinks.sh` | `[ev: tests/ci-local-kitlinks.bats]` |
