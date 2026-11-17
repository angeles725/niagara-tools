### Fixed — ng-deploy: the env file may not replace the schema-risk tool

- `SCHEMA_RISK_BIN` in the ng-deploy env file is refused (exit 10, file and line): it would put any executable in
  the place of the schema-risk precheck. The environment of the command still sets it.
