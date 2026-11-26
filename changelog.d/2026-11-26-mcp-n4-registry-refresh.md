### Fixed

- `n4_create_component` (mcp-n4, issue #1166) no longer refuses a type from a module deployed after the session
  read the station's type registry. The registry is cached for the session, so after a station restart with a new
  or upgraded module (stop, install, start) a type the session had cached as unknown stayed refused with "type ...
  is not known to the station" until `n4_connect` was called again. Before it refuses such a type the dry run now
  drops the cached registry and reads it from the station once more; a type the station now describes is planned
  as usual. A type that is still unknown keeps its refusal text, which ends with a note saying whether the registry
  was read for this call or re-read just now. A known type costs no extra call, so its plan is unchanged. A
  reconnect after a lost session already dropped the cache. Black-box tests: `blackbox/registry_restart_test.go`.
