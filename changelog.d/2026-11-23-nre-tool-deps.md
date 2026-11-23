### Added — `nre-tool.sh deps`: a module's dependencies from a Niagara install (issue #851)

- `toolbelt/nre-tool.sh deps [--raw] <niagara_home> <module>` lists the module parts a module depends on, one per line.
  It runs the install's `nre devkit:com.tridium.devkit.util.DependencyChecker` (read-only; no station, no Workbench),
  including a Windows install's `bin/nre.exe` from WSL.
- Exit codes:
  - 1 when the install holds no such module; the end of nre's output goes to stderr.
  - 3 when there is no `bin/nre.exe` or `bin/nre`, or the module name is bad.
- `--raw` prints nre's whole output.
