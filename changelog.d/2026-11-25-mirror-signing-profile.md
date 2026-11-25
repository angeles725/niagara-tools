### Fixed

- build-n4-module: a mirror build that passes `--gradle-prop niagaraSigningProfile=<file>` no longer warns that the jar is signed with the mirror home's throwaway key: the plugin signs with the named profile there too. (#1096)
