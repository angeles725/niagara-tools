### Added — lint-slot-default-type

- New `toolbelt/lint-slot-default-type.sh <src-root>`: FAILs an `@NiagaraProperty` of type `int`, `long`, `float`,
  `double`, `boolean` or `String` whose `defaultValue` literal has a Java type the slot refuses (for example `int` with
  `"false"`, `float` with `"1.5"`): the module compiles and the type fails to load at runtime. An `int` literal on a
  `long`/`float`/`double` slot stays clean. Not yet run by `report-module`.
