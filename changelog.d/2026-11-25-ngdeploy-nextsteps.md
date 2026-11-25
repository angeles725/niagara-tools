### Changed

- `ng-deploy` (`go/internal/ngdeploy`): a deploy of Java classes (modes A and C) now follows its "restart the station"
  line with the exact commands, not only the advice (retro issue #993, D10). One `next:` line gives the stop and start
  commands (`toolbelt/lab-station.sh stop|start --home <NIAGARA_HOME> <station>`, with the script's real path;
  `<station>` and `<NIAGARA_HOME>` are left for the author; a Windows service or `station.exe` is stopped and started
  there, and lab-station stops only a station it started). A second names the mcp-n4 `post-deploy-check` prompt with
  `station=<station> module=<MODULE_NAME>` for after the restart. Mode B (browser reload) prints neither, and ng-deploy
  still stops or starts nothing. `--help` (After a deploy) lists both lines; `TestADeployThatWentThroughEndsWithWhatToDoNext`
  pins them.
