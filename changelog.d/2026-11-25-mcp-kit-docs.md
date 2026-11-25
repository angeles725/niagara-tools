### Changed

- `mcp-n4-kit/skill/SKILL.md` (retro `2026-11-02-dashboard-live-check-4-15`, issue #985): step 1 gets a version
  floor (the server's `serverInfo.version` or `n4_describe_session` `server_version` must be at least the launcher's
  `metadata.version`; a lower one is a stale or retired build: resolve the kit again, never continue) and names
  `scripts/check-skill-drift.sh --skill mcp-n4` as the check for an installed launcher that drifts from the tracked
  copy (Δ21 rule; the retro's `install-skill.sh --check` does not exist). Step 5 says to `n4_save_station` and stop
  only on a `verified`, `persisted: true` verdict before any forced station stop (Δ14).
- `mcp-n4-kit/README.md`: `--state-dir` documents the refusal of a group- or world-writable existing directory and its
  `chmod 700` hint (Δ16); the Bench client section says never to hand-write a stdio driver and points to `n4call`
  and `go/bin/live-smoke` (Δ20).
