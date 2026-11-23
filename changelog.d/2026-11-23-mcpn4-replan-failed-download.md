### Changed — mcp-n4: token refusal after a lost session, failed Spyder download detail

- A confirmation token issued in an earlier station session (the station dropped the session, or `n4_connect` ran since) is refused naming that cause and the way out: call the same tool with the same arguments and `dry_run=true`, check the new plan, execute with the new token. The refusal sends nothing and issues no token. A write execution that loses its session says the same.
- A failed or canceled `spyderDownload` / `compile` read-back adds `log_error` (the station's error when the job log could not be read), `device_state` (`deviceStatus`, `compileStatusFlag`, `sErrorStatus`) and a hint naming a renamed `BacnetNIFolder` slot as the known cause. A successful job is unchanged.
- A write whose path is below a Spyder `ControlProgram` that the station refuses names the cause: the program unloads on its own and must be loaded again.
