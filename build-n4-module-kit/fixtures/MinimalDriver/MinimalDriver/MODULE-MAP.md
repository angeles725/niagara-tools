# MODULE-MAP — MinimalDriver

modularity: strict

Read this map before editing: change only the files of the concern you are changing, and add a row (or a file to
a row) in the same change as a new file.

| Concern | Part | Files | Tests | Change here when |
|---|---|---|---|---|
| codec | rt | MinimalDriver-rt/src/com/angeles/MinimalDriver/model/MinimalDriverCodec.java | MinimalDriver-rt/srcTest/test/com/angeles/MinimalDriver/model/MinimalDriverCodecTest.java | the frame layout, a command or a checksum rule changes |
| network-device | rt | MinimalDriver-rt/src/com/angeles/MinimalDriver/BMinimalDriverNetwork.java; MinimalDriver-rt/src/com/angeles/MinimalDriver/BMinimalDriverDeviceFolder.java; MinimalDriver-rt/src/com/angeles/MinimalDriver/BMinimalDriverDevice.java; MinimalDriver-rt/src/com/angeles/MinimalDriver/BMinimalDriverUdpCommConfig.java; MinimalDriver-rt/src/com/angeles/MinimalDriver/MinimalDriverMessageFactory.java; MinimalDriver-rt/src/com/angeles/MinimalDriver/MinimalDriverRequest.java; MinimalDriver-rt/src/com/angeles/MinimalDriver/MinimalDriverResponse.java | MinimalDriver-rt/srcTest/test/com/angeles/MinimalDriver/BMinimalDriverLoopbackTest.java; MinimalDriver-rt/srcTest/test/com/angeles/MinimalDriver/MinimalDriverLoopbackDevice.java | the transport, the ping or the request/response correlation changes |
| points | rt | MinimalDriver-rt/src/com/angeles/MinimalDriver/BMinimalDriverPointDeviceExt.java; MinimalDriver-rt/src/com/angeles/MinimalDriver/BMinimalDriverPointFolder.java; MinimalDriver-rt/src/com/angeles/MinimalDriver/BMinimalDriverProxyExt.java | MinimalDriver-rt/srcTest/test/com/angeles/MinimalDriver/BMinimalDriverLoopbackTest.java | a point read, write or poll changes |
| discovery | rt | MinimalDriver-rt/src/com/angeles/MinimalDriver/BMinimalDriverDeviceDiscoveryPreferences.java; MinimalDriver-rt/src/com/angeles/MinimalDriver/BMinimalDriverDeviceDiscoveryLeaf.java; MinimalDriver-rt/src/com/angeles/MinimalDriver/BMinimalDriverPointDiscoveryPreferences.java; MinimalDriver-rt/src/com/angeles/MinimalDriver/BMinimalDriverPointDiscoveryLeaf.java | MinimalDriver-rt/srcTest/test/com/angeles/MinimalDriver/BMinimalDriverLoopbackTest.java | device or point discovery changes |
| point-manager | wb | MinimalDriver-wb/src/com/angeles/MinimalDriver/wb/BMinimalDriverPointManager.java | MinimalDriver-rt/srcTest/test/com/angeles/MinimalDriver/BMinimalDriverLoopbackTest.java | the Workbench point manager view changes |
| plumbing | rt, wb | MinimalDriver-rt/MinimalDriver-rt.gradle.kts; MinimalDriver-rt/module-include.xml; MinimalDriver-rt/moduleTest-include.xml; MinimalDriver-rt/module.palette; MinimalDriver-rt/module.lexicon; MinimalDriver-rt/module-permissions.xml; MinimalDriver-wb/MinimalDriver-wb.gradle.kts; MinimalDriver-wb/module-include.xml; MinimalDriver-wb/module.lexicon; MinimalDriver-wb/module-permissions.xml | — | build, packaging, palette or lexicon only |
