<!-- ledger: kit -->
<!-- date: 2026-10-24 -->
<!-- last-change-note: quality round 10: station-snapshot runs the Go port -->
log: 2026-10-24 · quality round 10 → `toolbelt/station-snapshot.sh` is the thin wrapper over `go/cmd/station-snapshot` (`go/internal/stationsnapshot`: cp -p + chmod 0644 copies of config.bog and console*.txt, history/** size pointers, manifest.json with the bash layout); 58 goldens (57 recorded from the bash with a fake date, 1 truth case: a bad TMPDIR no longer empties the manifest) replay through golden.Runner; station-snapshot.bats and kit-links.bats green through the wrapper. [ev: retro go-station-snapshot]
