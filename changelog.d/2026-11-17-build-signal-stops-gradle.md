### Fixed — build.sh stops gradle before it puts modules/ back on an interrupt

- A SIGINT, SIGTERM or SIGHUP sent to build.sh's pid alone used to leave gradle running, and it could install
  its jar after the restore. The handler now passes the signal on to the running gradle and waits for it
  (killing it after 10 s) before it restores the pre-build jars.
