### Fixed

- build-n4-module: `scaffold-module.sh --type service` modules compile: the service-job template's `fileName` default is now the escaped literal `"\"<Mod>-export.csv\""` (Slot-o-Matic copied the bare text into `newProperty(...)` as code). A test over the real templates rejects an unescaped String `defaultValue`. (#1093)
