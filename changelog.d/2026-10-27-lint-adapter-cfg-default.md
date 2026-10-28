### Added — lint-adapter-cfg-default

- New `toolbelt/lint-adapter-cfg-default.sh <module-root>...` (audit A7-D1): WARNs a pure-model `Cfg` field whose
  default (its literal initializer, or 0/false) differs from the `@NiagaraProperty` default of the slot the Baja
  adapter fills it from (`cfg.f = getSlot();`, through `.getMillis()`/`.getSeconds()` for a BRelTime), so a pure
  test of `new Cfg()` runs the configuration the station starts with. Not yet run by `report-module`.
