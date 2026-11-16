<!-- ledger: kit -->
<!-- date: 2026-11-15 -->
<!-- last-change-note: retro of a build-kit bench session: 21 proposed deltas from a scaffold-to-deploy run of every scaffold type -->
log: 2026-11-15 · build-kit bench session → `retros/2026-11-15-build-kit-bench-session.md` records a newcomer run of the kit (scaffold, `toolbelt/build.sh`, gate, report, `scripts/ng-deploy.sh`) for the `logic`, `dashboard` and `driver` types against a mirror of a 4.15 installation, plus one deploy into a laboratory installation; 21 deltas proposed, none applied
open_issue: retro build-kit-bench-session Δ1-Δ21 pending (deploy writes the build home before its checks; deploy gate without the expected signer; false version-bump refusal on an unchanged rebuild; logic scaffold fails the report; no `-wb` deploy mode; signing-profile and plugin-version first-build traps)
