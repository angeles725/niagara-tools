# MODULE-MAP — ModuleA

modularity: strict

Read this map before editing: change only the files of the concern you are changing, and add a row (or a file to
a row) in the same change as a new file.

| Concern | Part | Files | Tests | Change here when |
|---|---|---|---|---|
| export | rt | ModuleA-rt/src/com/example/ModuleA/BModuleAService.java; ModuleA-rt/src/com/example/ModuleA/BModuleAExportJob.java; ModuleA-rt/src/com/example/ModuleA/model/ModuleAExport.java | ModuleA-rt/srcTest/test/com/example/ModuleA/model/ModuleAExportTest.java | the service's slots, the job's loop or the export's line format change |
| plumbing | rt | ModuleA-rt/ModuleA-rt.gradle.kts; ModuleA-rt/module-include.xml; ModuleA-rt/module.palette; ModuleA-rt/module.lexicon; ModuleA-rt/module-permissions.xml | — | build, packaging, palette or lexicon only |
