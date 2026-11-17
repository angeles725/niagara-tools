### Added — mcp-n4: Spyder section of n4_commissioning_report

- `n4_commissioning_report` section `spyder` (asked in `sections`, or added by `spyder_spec`): lists each
  Honeywell BACnet Spyder left disabled, with `deviceStatus` other than 1, `compileStatusFlag` other than 0
  or a non-empty `sErrorStatus`; with `spyder_spec` (`{<Spyder>: {<point>: {sensor_type, input_low,
  input_high, output_low, output_high, terminal}}}`) it compares each IOComp with the table. The default
  sections are unchanged.
