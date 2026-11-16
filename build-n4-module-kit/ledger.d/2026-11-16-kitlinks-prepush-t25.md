<!-- ledger: kit -->
<!-- date: 2026-11-16 -->
<!-- last-change-note: ci-local --changed (the pre-push replay) runs kit-links when a kit doc, a ledger.d or a changelog.d file changes -->
log: 2026-11-16 · ci-local T25 → `scripts/ci-local.sh --changed` runs `tests/kit-links.bats` on a snapshot of a change that touches a kit doc, a ledger.d or a changelog.d file, with the generated build state rebuilt in the snapshot, so a fragment naming a bare missing doc is refused at push instead of in CI (#706); the host-budget gate moved to `scripts/lib/ci-local-host.sh` unchanged. Retro kitlinks-prepush-t25 (Δ1-Δ2 applied, pending review). [ev: retro kitlinks-prepush-t25 Δ1] [ev: retro kitlinks-prepush-t25 Δ2]
