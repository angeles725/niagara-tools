# Station diagnostics — engine stats, log levels, console messages

The triage read of a running station. How a module logs, exposes spy pages and surfaces faults is in
`types/observability.md`; console triage is `toolbelt/triage-console.sh`. `[ev: retro n4-course-parity-observability-horizons Δ4]`
Thread dumps, a remote debugger and the launcher flags an author uses are § 4.

Derivations: course deck/lab screenshots in corpus B1255 (§1255.7), B1256 (§1256.3), B1257 (§1257.10), B1236
(§1236.2). Code: `V=organized/baja/baja/vineflower/com/tridium` (4.14.0.162 decompile, under niagara-research).

## 1 · Engine-thread stall: Resource Manager statistics

The engine's resource report (Workbench Resource Manager view) holds `engine.scan.peak`, `engine.scan.peakInterscan`
(the longest gap between two engine scans after steady state, with `engine.scan.timeOfPeakInterscan`) and
`engine.queue.actions` (actions waiting for the engine thread, shown as `<now> (Peak <n>)`), next to the short,
medium and long timer queues. [CERT] `V/sys/engine/EngineManager.java:169,414-431`. A long `peakInterscan` and a
high action-queue peak date a callback or action that held the engine thread; correlate the time with the console.

## 2 · Log levels `[ev: retro n4-course-lexicon-logging-testing Δ5]`

- **Default INFO.** Every logger starts at INFO (course Lab 7 step 10; corpus B1236 §1236.2), so FINE lines are
  invisible until someone raises the category.
- **Raise one category, do not log louder.** Add the logger name (the module name, `types/observability.md` § 1) at
  FINE in Services › DebugService (type `baja:LoggingService`, `baja.lexicon:311`) or through `spy:/logSetup`
  (`V/sys/Nre.java:730`). FINE lines then reach the console but not LogHistory (`types/observability.md` § 1,
  `minimumSeverity`).
- **Two spy pages for "is it loaded, is it logging":** Spy › Log Setup lists every logger the station defined,
  while the DebugService lists only configured categories; Spy › sysManagers › registryManager lists the installed
  modules and each module's types (the place to confirm a type registered after a deploy).
  `[ev: corpus B1418 §1418.2 f]` `[ev: corpus B1417 §1417.3 f]` `[ev: retro course-coverage Δ13]`
- **Per-frame and per-poll traces go at FINE** behind `isLoggable(Level.FINE)`; INFO in a poll or receive body floods
  the console and LogHistory of every site.

## 3 · Console messages and what they mean

| Console text | Meaning | Action |
|---|---|---|
| `ActionInvokeException` → `ServerException` → `AccessControlException` | a module permission is missing; read the innermost exception (B1257 §1257.10 rule 5) | raise logger `security.niagaraPolicy` (`V/security/UIPermissionGroup.java:40`) to FINE: it prints the permission and the ProtectionDomain that failed (B1255 §1255.4). If that domain is a framework jar (`fox-rt.jar` in the course), the check reached a caller lower on the stack: wrap your call in `AccessController.doPrivileged` |
| SEVERE `Cannot stop component: <name>` | something threw out of `stopped()`/`descendantsStopped()` (`V/sys/engine/EngineUtil.java:105,123`) | in the course it was `shutdownNow()` denied without `MANAGE_EXECUTION`; the threads leak (B1255 §1255.7) |
| WARNING `Could not validate certificate path for entry … in module …` | the jar's signer is not trusted; logged only while verification is not required (`V/sys/module/ModuleClassLoader.java:77,497-503`) | it loads at `low` and fails at `medium`/`high` (B1257 §1257.10 rule 3); sign per `build-verify.md` § Signing per deploy target |
| a UI label equal to its lexicon key, or untranslated text | see `types/issues-and-gotchas.md` K1 (diagnostic lexicon) | — |

**Spanish-locale consoles print localized level names.** OpenJDK 8 `sun.util.logging.resources.logging_es`:
SEVERE `Grave`, WARNING `Advertencia`, INFO `Información`, CONFIG `Configurar`, FINE `Detallado`, FINER
`Muy Detallado`, FINEST `Más Detallado` (read from `jre/lib/rt.jar` of OpenJDK 1.8.0, 2026-10-23). A live course
station printed them upper-case (`INFORMACIÓN`, `GRAVE`, `DETALLADO`). `triage-console.sh` recognises all seven,
upper-case, in UTF-8 and in Latin-1, the two-word FINER and FINEST included (`isLevel`,
`go/internal/triageconsole`; golden cases `truth-es-levels` and `truth-informacion-level`, `TestLevelFieldTwoWord`).
Matching is exact: the unaccented `INFORMACION` is no level. `[ev: retro n4-course-lexicon-logging-testing Δ4]` `[ev: retro doc-coverage-audit Δ4]`

## 4 · Debugging a running module: thread dumps, remote debugger, launcher flags `[ev: retro doc-coverage-audit Δ11]`

**Development or laboratory station only for § 4.2, and for every verb of § 4.1 except `threads`, `version`, `heap`
and `help`.** A debugger breakpoint suspends station threads, and a station whose engine thread is suspended runs no
control logic; `kill` exits without saving. On a controller that runs plant equipment, take a thread dump and read
the console, nothing more.

Guides: `GP/` = `niagara-help/versions/4.15.6.20/guides-clean/` (under niagara-research); corpus B1296 (station
console), B1297 (launcher flags), B1230 §1230.3 and B1249 §1249.1 (debugger). The console verbs and launcher flags
are read from 4.14.0.162 code; the 4.15 set is not certified yet (niagara-research B1296-G1), so run `help` first.

### 4.1 · Thread dump of a station

A thread dump prints the stack of every JVM thread to the **station output**; it is the first read when an action
never returns, a view hangs or the engine stalls (§ 1). Three ways reach the same `threads` console verb:

| Where | Steps | What you see |
|---|---|---|
| Platform › Application Director | select the station, press **Dump Threads** (also on the **Output Dialog** window) | the dump appended to the Application output area; **Stream To File** saves the output to a file first. [CERT-doc] `GP/Platform/platDaemon-ApplicationDirector.txt` |
| Supervisor › Provisioning station proxy | invoke the **Dump Threads** action of the station proxy | the dump in that station's console output; it needs the proxy's `Is Accepting Messages` true. [CERT-doc] `GP/Provisioning/provisioningNiagara-StationProxy.txt` |
| Station console (a station started with `station <name>` in a Niagara console) | type `threads` at the `niagara>` prompt | the dump in that console `[ev: corpus B1296 §1296.2]` |

- **The buttons are the console verb.** The platform daemon writes the verb as a line to the station's standard input,
  so Application Director **Dump Threads** is `threads` and **Save Bog** is `save`; `plat tellstation <station>
  threads` scripts the same thing. `[ev: corpus B1296 §1296.3]`
- **Reading it.** Find your thread by name: engine callbacks run on `Nre:Engine` (`types/logic.md`, HogsPage
  bullet), the console itself is `Nre:Console` `[ev: corpus B1296 §1296.1]`, and the station thread map is corpus
  B1238. A thread parked inside your class on a lock or a socket read is the stall. Take two dumps some seconds
  apart: a frame that sits in both is blocked, one that moved is only slow `[INFER]`.
- **Risk.** A dump only prints `[INFER]`; it is the one debugging step acceptable on a production station. Start
  Stream To File before a dump you mean to keep: the daemon's output buffer has a fixed size (**Output Settings**,
  same guide page).

Station console verbs (the `help` text lists the first eight; no official guide page lists them; the page titled
"nre (station) commands" documents the `nre` launcher) `[ev: corpus B1296 §1296.2, §1296.5]`:

| Verb | Effect | On a production station |
|---|---|---|
| `help`, `version`, `heap` | usage; JVM, home, host id and versions; total, free and used memory | safe |
| `threads` | JVM thread dump to the station output | safe |
| `save` | asynchronous save of `config.bog` | safe |
| `gc` | runs `System.gc()` | a pause on a controller heap; not on a live plant |
| `quit` | orderly shutdown, then the JVM exits | stops the station |
| `kill` | exits at once with exit code 0: no save, no orderly stop, and the daemon reads a clean exit, so Restart on Failure does not bring the station back `[INFER]` `[ev: corpus B1296 §1296.4]` | never |
| `syncmodules` (hidden) | re-syncs the module registry with the jars on disk | laboratory |
| `settestargs`, `runtests` (hidden) | drive the `-watch` test loop (corpus B1300) | laboratory |

A console prompt appears only on a station started from a console; a daemon-started station has none and takes the
verbs through the daemon. Stop a console station with `quit`, never Ctrl+C: it would not save.
`[ev: corpus B1296 §1296.1-1296.2]`

### 4.2 · Attaching a remote Java debugger

1. **Licence.** The native launcher refuses any JVM option that contains `agentlib`, `agentpath` or `javaagent`
   unless the host licence holds the `developer` feature of the Tridium licence; the refusal is a `FATAL: Can not use
   Java agent argument` line. `[ev: corpus B1249 §1249.1]`
2. **Where the option goes.** On the console command line that starts the station (or Workbench), through the
   launcher's JVM pass-through `-@`: `station <name> -@agentlib:jdwp=transport=dt_socket,server=y,suspend=n,address=8000`.
   The launcher turns `-@X` into the JVM option `-X`, so the body carries no dash of its own. [CERT-doc] course Lab
   Instructions Appendix A `[ev: corpus B1230 §1230.3]`; `-@option` is the documented pass-through, [CERT-doc]
   `GP/User/dlStationConsoleCommands.txt`.
3. **What you see.** The console prints `Listening for transport dt_socket at address: 8000`; in the IDE create a
   Remote JVM Debug configuration (attach, socket, host and port 8000) and it reports the connection. Use
   `suspend=y` to stop at startup code such as `started()`. `[ev: corpus B1257 §1257.7]` `[ev: corpus B1230 §1230.3]`
4. **Switching it off.** The option lives only on that command line: `quit` the station and start it without it.

How to pass the agent option to a station the platform daemon starts is not covered by our sources; debug a station
you start from a console.

**Risks, and why this never goes on a production station.** A breakpoint on the engine thread stops every component's
`execute()` and every timer until you resume, so outputs hold their last value and safeties do not run `[INFER]`; a
station whose engine stops answering can be terminated `[INFER]` by the engine watchdog and then restarted by the daemon
(Restart on Failure, [CERT-doc] `GP/Platform/platDaemon-ApplicationDirector.txt`). `server=y` opens a listening TCP
port on the host for as long as the station runs.

### 4.3 · Launcher and console flags an author uses `[ev: corpus B1297]`

Every Java launcher shares one grammar: `-name:value`, split at the first colon, names case-sensitive (`-locale:en_US`
works, `-locale=en_US` does not); `-?`, `-help` and `-usage` print the usage. `[ev: corpus B1297 §1297.1]`

| Command | Use | Notes |
|---|---|---|
| `nre -modules:<pattern>` | list the installed modules that match | is my jar seen, and which version. [CERT-doc] `GP/User/dlStationConsoleCommands.txt` |
| `nre -buildreg` | force a registry rebuild | after replacing a jar by hand in a laboratory install |
| `nre -props`, `-hostid`, `-licenses`, `-version` | system properties, host id, licence summary, NRE version | `-licenses` answers "does this host hold my feature" (`types/security.md` § 5) |
| `nre -rp:<key>[,<key>]` (in the usage text, not in the guide) | restrict the runtime profiles loaded to `ux`, `wb`, `se`, `doc` | reproduces a controller that lacks `wb` on a workstation `[ev: corpus B1297 §1297.2]` |
| `<launcher> -@<jvm option>` | pass one option to the JVM | at least two characters after `-@`, no dash: `-@Xmx512m` `[ev: corpus B1249 §1249.1]` |
| `station <name> -bootFile:<file>` (no guide page) | boot from another file of the station folder instead of `config.bog` | try a saved copy without overwriting the live file `[ev: corpus B1297 §1297.3]` |
| `wb -profile:<type>`, `wb -locale:<x>`, `wb -theme:<name>` (`-theme` is not in the guide) | open Workbench in a profile, a locale or a theme | checks a lexicon or a theme module without changing user options `[ev: corpus B1297 §1297.4]` |
| `test <module>`, `test <module>:<Type>`, `test <module>:<Type>.<method>`, `-v:<1-10>` | run TestNG types | `types/moduleTest.md`; only registered `BTestNg` types run `[ev: corpus B1297 §1297.6]` |
| `nre -watch` / `test -watch` with console `settestargs` | the rebuild-and-rerun test loop | needs the `developer` licence feature `[ev: corpus B1297 §1297.2]` |
| `niagarad /console`, `/trace`, `/echostation` (no guide page) | run the platform daemon in the foreground, force every log filter to FINE, copy station output into daemon output | laboratory host only `[ev: corpus B1297 §1297.5]` |

- **A JVM property on the command line can be ignored.** `-@Dkey=value` over a key of the command-line blacklist
  (`niagara.moduleVerificationMode`, `program.requireSigning` among the defaults) is dropped with an `Ignoring command
  line property` warning unless the launching user can lock `defaults/system.properties`; edit that file with
  Workbench and the stations stopped instead. `[ev: corpus B1297 §1297.2]`
- **Station exit codes the daemon reads:** `-2` missing arguments, `-3` not licensed, `-4` station folder or boot
  file missing, `-5` database not loadable, `-7` boot failed or boot file locked; the daemon does not restart after
  any of them. `[ev: corpus B1296 §1296.4]`
- **`NRE_CONSOLE=false`** in the environment starts a station without its console thread. `[ev: corpus B1296 §1296.1]`
