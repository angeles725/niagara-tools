# MODULE-MAP — MinimalPan

modularity: strict

Read this map before editing: change only the files of the concern you are changing, and add a row (or a file to
a row) in the same change as a new file.

| Concern | Part | Files | Tests | Change here when |
|---|---|---|---|---|
| control | rt | MinimalPan-rt/src/com/angeles/MinimalPan/model/MinimalPanLogic.java; MinimalPan-rt/src/com/angeles/MinimalPan/BMinimalPan.java | MinimalPan-rt/srcTest/test/com/angeles/MinimalPan/model/MinimalPanLogicTest.java | a control decision (the interval guard) or the component's slots change |
| station-test | rt | — | MinimalPan-rt/srcTest/test/com/angeles/MinimalPan/BMinimalPanLinkTest.java; MinimalPan-rt/srcTest/test/com/angeles/MinimalPan/BMinimalPanStationTestBase.java | a link-in slot or the station-side behaviour changes |
| plumbing | rt | MinimalPan-rt/MinimalPan-rt.gradle.kts; MinimalPan-rt/module-include.xml; MinimalPan-rt/moduleTest-include.xml; MinimalPan-rt/module.palette; MinimalPan-rt/module.lexicon; MinimalPan-rt/module-permissions.xml | — | build, packaging, palette or lexicon only |
