<!-- ledger: kit -->
<!-- date: 2026-10-24 -->
<!-- last-change-note: quality round 10: build.sh refuses a real niagara_home unless --install-into-home -->
log: 2026-10-24 · quality round 10 → toolbelt/build.sh exits 10 before gradle when niagara_home lacks the .niagara-mirror marker (mirror-niagara-home.sh writes it): the plugin's :jar step replaces <niagara_home>/modules jars and the drift gate's backup is a temp dir removed on success, so a client build pointed at a real install lost its jars silently. --install-into-home opts in. Pins: tests/build-sh.bats BS-home-guard (RED, nothing written to modules/), BS-home-guard-optin (positive control), BS-home-guard-mirror; scaffold-module TC4 builds into a throwaway mirror. Retro: retros/2026-10-24-r10-buildsh-mirror-guard.md.
