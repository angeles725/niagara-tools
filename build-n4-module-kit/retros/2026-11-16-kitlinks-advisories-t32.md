<!-- review-status: folded -->
<!-- module: kit -->
<!-- deltas: 4 -->
# 2026-11-16 · kit · kitlinks-advisories-t32

**Session**: row T32 of the 2026-11-15 coordination-stretch retro: the native-review advisories of niagara-tools #717,
which added the kit-links step to `scripts/ci-local.sh --changed`.
**Delta count**: 4

## What happened
Four advisories. A changelog.d-only change was never tested to run kit-links. The bats provisioning result was not
checked: `run_kit_links` runs in an errexit-ignored context and `provision_bats` ends on a printf, so a failed tar left
the bats already on PATH to run. The kit-state step ran the checkout's `toolbelt/kit-state.sh` while kit-links.bats in
the same snapshot runs the snapshot's own. The change detection had no failure path of its own: with `--range` a
failed `git diff` read as "changed" only through `!`, and the snapshot then reported a misleading "cannot snapshot";
without `--range` a failed `ls-files` read as "no kit change". Now a failed detection is one named line and exit 3, a
provisioning that fails exits 3 (the download or sha256 error of `fetch_pinned`, or one line when it returns an error
or leaves another bats version first on PATH), and kit-state runs from the snapshot (the launcher the push sends; go/
stays the checkout's either way). The independent verifier then saw a pinned-tarball download on a blackholed network
still running after 99 s (curl's default connect timeout and `--retry 3` allow ~20 minutes); `fetch_pinned` now
bounds each try with `--connect-timeout 10 --max-time 120`.

## Evidence
- RED before GREEN: CL34 (stubbed provisioner returning 1), CL35 (provisioner leaving the wrong version), CL36 (bad
  range on `run_kit_links`) and CL37 (pushed launcher differs from the checkout's) failed on origin/main; CL33
  (changelog.d only) already passed, a coverage gap pinned by its mutation. `[ev: tests/ci-local-kitlinks.bats]`
- Through the CLI a bad `--range` stops in `run_changed` (exit 3) before kit-links, so CL36 pins the guard on
  `run_kit_links` itself. `[ev: scripts/lib/ci-local-changed.sh]`
- CL24 (tests/ci-local-pins.bats) runs a stub curl that records its arguments and exits 28: RED without the two
  options, GREEN with them; no network. `[ev: tests/ci-local-pins.bats]`
- Mutations in a scratch copy, not committed: dropping changelog.d from the pathspec (CL33), the provisioner status
  check (CL34), the pinned-version check (CL35), the detection error path (CL36) and running the checkout's
  launcher (CL37) each turn their test red. `[ev: tests/ci-local-kitlinks.bats]`

## Proposed kit deltas (propose-never-apply)
| Δ | Delta | Target file / § | Token |
|---|---|---|---|
| Δ1 | APPLIED 2026-11-16. A kit-links bats that cannot be provisioned at the ci.yml pin exits 3 (the download error, or one line when it is not the pinned version) | `scripts/lib/ci-local-kitlinks.sh` | `[ev: scripts/lib/ci-local-kitlinks.sh]` |
| Δ2 | APPLIED 2026-11-16. A git error while detecting a kit doc change is reported (exit 3), never read as no change | `scripts/lib/ci-local-kitlinks.sh` | `[ev: scripts/lib/ci-local-kitlinks.sh]` |
| Δ3 | APPLIED 2026-11-16. The snapshot's own kit-state launcher materializes its generated state | `scripts/lib/ci-local-kitlinks.sh` | `[ev: tests/ci-local-kitlinks.bats]` |
| Δ4 | APPLIED 2026-11-16. A pinned-tarball download is bounded: curl `--connect-timeout 10 --max-time 120` | `scripts/lib/ci-local-tools.sh` | `[ev: tests/ci-local-pins.bats]` |
