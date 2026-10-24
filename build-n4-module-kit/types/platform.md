# Platform — runtime profiles, Software Manager install rules, NRE memory on a JACE

What the host platform decides before a module's code runs: which profile jars a host holds, when an install stops
the station, and how much memory `-rt` code gets on a JACE. Packaging and `.dist` files: `types/distribution.md`;
profile meanings and the `se` host rule: `types/structure.md` § Layout.

Guides: `G/` = `niagara-help/versions/4.15.6.20/guides-clean/Platform/` (under niagara-research).

## 1 · Runtime profiles on a host `[ev: corpus B1341 §1341.1 rows 1-2]` [ev: retro devguide415-r2-packaging-platform-signing Δ9]

- A JACE has `-ux` and `-wb` jars only if the **Configure Runtime Profiles** wizard enabled those profiles. Before
  blaming a deploy for a missing `-ux` dashboard jar on a JACE, read the enabled profiles in Platform Administration.
- The wizard installs or uninstalls the profile jars and stops and restarts a running station; disabling down to
  rt+ux only is refused; for N4.0 the guide says UX and WB were selected together (`G/ChangingAModulesProfile-2D2BEE6C.txt:19-36` "The figure above shows typical settings for controllers").
  Whether 4.15 still ties them is open (niagara-research B1341-G1).
- A Windows host normally enables every profile, plus `doc` when it runs Workbench (`G/aPlat4PlatformAdminConfigureRuntimeProfiles.txt:9` "For any Windows-based host (provided it has a").

## 2 · Software Manager install and upgrade rules `[ev: corpus B1341 §1341.1 rows 4-6]` [ev: retro devguide415-r2-packaging-platform-signing Δ9]

| Operation | Station / host effect | Guide |
|---|---|---|
| Install a NEW module | can install with the station running, no reboot ("in some cases") | `G/aSoftwareManager.txt:18` "In some cases, you can install a new" |
| Upgrade or downgrade an installed module | the station stops and the host reboots | `G/aSoftwareManager.txt:18` "In some cases, you can install a new", `G/InstallingModulesInARemotePlatform-2C3324A0.txt:50` "If you are upgrading or downgrading modules, a" |
| Module needs a dependency not installed | Installing Additional Software flags the dependencies too | `…InstallingModulesInARemotePlatform-2C3324A0.txt:36-41` |
| Uninstall a module others depend on | Cannot Uninstall Module, unless the dependents are flagged too | `G/RemovingModulesFromARemotePlatform-2C318A1E.txt:7,20-22` "Modules you remove from a remote platform may" |
| Status "Requires Commissioning" | install only after the Commissioning Wizard | `G/aSoftwareManager.txt:15` "Module statuses of Out of Date and Not" |
| A module with a signature error | the install fails; a warning opens a dialog | `…InstallingModulesInARemotePlatform-2C3324A0.txt:20` |
| Station Copier: a needed module needs a core or QNX upgrade | the station install stops at station selection | `G/aStationCopier_35_changes.txt:9` "If any module needed by the station has" |

So: put controlled equipment in manual before any upgrade (the guide's own precondition, `…InstallingModulesInARemotePlatform-2C3324A0.txt:9`),
keep `<dependency>` lists to what the code uses (each one is pulled in and pins uninstall), and copy stations with
the Station Copier, which transcodes user passwords (`G/aStationCopier.txt:9` "Note: To copy stations, it is strongly recommended"). Recovery of a station that will
not boot: `types/distribution.md` § 10; File Transfer Client is not an install path (same §).

## 3 · NRE memory and flash on a JACE `[ev: corpus B1341 §1341.1 row 7]` `[ev: corpus B1284]` [ev: retro devguide415-r2-packaging-platform-signing Δ10]

- One fixed memory footprint serves every NRE pool; growing one pool shrinks another. Defaults on the Configure NRE
  Memory Pools window: System Reserve 0 MB, Heap 384 MB, Meta Space 128 MB (classes from every installed module),
  Code Cache 32 MB, about 352 MB free (`G/aPlatformQNX_ConfigureNreMemory.txt`). A change needs a controller reboot;
  too little memory can stop the station from starting or make it fail and restart (`G/ConfiguringMemoryToImprovePerforman-2993BF3C.txt:11` "CAUTION: Configuring a controller with insufficient memory allocations").
- Budget `-rt` code against that heap: bounded caches and queues, no per-poll allocation that grows with points
  (Full-GC pauses on a JACE heap: `types/math-control-libraries.md`). Every extra module jar costs Meta Space.
- On a controller the file system writes alarm and history data straight to flash, to free RAM
  (`G/ConfiguringMemoryToImprovePerforman-2993BF3C.txt:21` "The file system writes the alarm and history"). `[INFER]` flash has finite write endurance: keep
  history intervals and alarm rates at what the site needs; never log a fast-changing value COV without a tolerance.
- A Supervisor's station heap is `-Xmx` in `station.java.options` of Daemon Home `etc/nre.properties` — one key for
  every daemon-started station (corpus B1284 §1284.2.1).
