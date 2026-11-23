# MODULE-MAP — MinimalLibrary

modularity: strict
type: library

Read this map before editing: change only the files of the concern you are changing, and add a row (or a file to
a row) in the same change as a new file. `type: library` declares a module that is depended on, never dragged onto
a palette: its `-rt` part may ship no `module.palette` (`types/utility-lib.md` § 6, `types/structure.md` § Module map).

| Concern | Part | Files | Tests | Change here when |
|---|---|---|---|---|
| plumbing | rt | MinimalLibrary-rt/MinimalLibrary-rt.gradle.kts; MinimalLibrary-rt/module-include.xml; MinimalLibrary-rt/module-permissions.xml | — | build, packaging or the wrapped library's version |
