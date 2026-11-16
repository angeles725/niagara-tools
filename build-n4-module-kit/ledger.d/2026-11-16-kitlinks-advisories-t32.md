<!-- ledger: kit -->
<!-- date: 2026-11-16 -->
<!-- last-change-note: ci-local --changed kit-links fails loudly on a bats it cannot provision or a git error in its change detection -->
log: 2026-11-16 · ci-local T32 → the kit-links step of `scripts/ci-local.sh --changed` exits 3 with one stderr line when bats cannot be provisioned at the version ci.yml pins or git fails while detecting a kit doc change, and runs the snapshot's own kit-state launcher; a changelog.d-only change is pinned to run kit-links (#717 review advisories). Retro kitlinks-advisories-t32 (Δ1-Δ3 applied, pending review). [ev: retro kitlinks-advisories-t32 Δ1] [ev: retro kitlinks-advisories-t32 Δ2] [ev: retro kitlinks-advisories-t32 Δ3]
