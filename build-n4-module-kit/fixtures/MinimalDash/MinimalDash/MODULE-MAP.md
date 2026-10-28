# MODULE-MAP — MinimalDash

modularity: strict

Read this map before editing: change only the files of the concern you are changing, and add a row (or a file to
a row) in the same change as a new file.

| Concern | Part | Files | Tests | Change here when |
|---|---|---|---|---|
| control | rt | MinimalDash-rt/src/com/angeles/MinimalDash/model/MinimalDashLogic.java; MinimalDash-rt/src/com/angeles/MinimalDash/BMinimalDash.java | MinimalDash-rt/srcTest/test/com/angeles/MinimalDash/model/MinimalDashLogicTest.java | a facade slot or its validity guard changes |
| api-routing | ux | MinimalDash-ux/src/com/angeles/MinimalDash/ux/route/BMinimalDashServlet.java; MinimalDash-ux/src/com/angeles/MinimalDash/ux/route/MinimalDashDispatch.java | MinimalDash-ux/srcTest/test/com/angeles/MinimalDash/ux/route/MinimalDashDispatchTest.java | a route, a request guard or a JSON response changes |
| frontend | ux | MinimalDash-ux/src/rc/** | MinimalDash-ux/srcTest/rc/spec/MinimalDashSpec.js | the page, its styles or its scripts change |
| station-test | rt | — | MinimalDash-rt/srcTest/test/com/angeles/MinimalDash/BMinimalDashLinkTest.java; MinimalDash-rt/srcTest/test/com/angeles/MinimalDash/BMinimalDashStationTestBase.java | a link-in slot or the station-side behaviour changes |
| plumbing | rt, ux | MinimalDash-rt/MinimalDash-rt.gradle.kts; MinimalDash-rt/module-include.xml; MinimalDash-rt/moduleTest-include.xml; MinimalDash-rt/module.palette; MinimalDash-rt/module.lexicon; MinimalDash-rt/module-permissions.xml; MinimalDash-ux/MinimalDash-ux.gradle.kts; MinimalDash-ux/module-include.xml; MinimalDash-ux/module.palette; MinimalDash-ux/module.lexicon; MinimalDash-ux/module-permissions.xml | — | build, packaging, palette or lexicon only |
