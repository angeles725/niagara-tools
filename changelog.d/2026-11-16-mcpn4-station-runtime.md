### Added — mcp-n4 n4_station_runtime: uptime, loaded modules and jars on disk

- `n4_station_runtime` reads, in one call, the station's uptime (spy `sysInfo`, seconds from its boot and current
  ticks), the module parts it has loaded with their versions (spy `sysManagers/moduleManager`) and, when the server
  has `--niagara-home` for the station, the jars of the installation's `modules` directory. `mismatches` names each
  difference (`version_differs`, `modified_after_boot`, `loaded_not_on_disk`, `not_loaded_modified_after_boot`) and
  `restart_needed` says whether a restart is needed to run what is on disk: `null`, with `restart_note`, when a part
  is unknown. Lists are capped by `max_rows`. Research retro row M4.
