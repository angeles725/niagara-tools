### Added — strict lint-size, strict slot wall and the adapter branch budget

- `lint-size` reads the module map mode per file. With no map it is unchanged. Under a map its method rule covers
  every method and constructor of every class (Baja classes too) and counts a wrapped signature from its first
  line; class and method rows WARN in a `modularity: legacy` module and FAIL in a `modularity: strict` one. New
  `adapter-branches` WARN (both map modes): a Baja class that imports a `model/` peer and keeps more than 15
  hand-written `if`/`case`/`for`/`while`/`catch` branches.
- `verify-module --src` `slot-wall`: more than 15 `@NiagaraProperty` in one class now FAILs the gate in a strict
  module (still a WARN elsewhere).
