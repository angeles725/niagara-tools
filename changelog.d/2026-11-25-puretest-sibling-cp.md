### Changed

- build-n4-module: `run-pure-test.sh` compiles the Baja adapters against `NIAGARA_HOME/modules` and the `build/libs` jars of the module's sibling parts, so a wb or ux adapter using rt types compiles; a compile error names the sibling jars it used. (#1075)
