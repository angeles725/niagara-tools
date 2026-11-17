### Fixed — build.sh stops gradle before it puts modules/ back on an interrupt

- A SIGINT, SIGTERM or SIGHUP sent to build.sh's pid alone used to leave gradle running, and it could install
  its jar after the restore. The handler now takes the restore lock, passes the signal on to the gradle the run
  started and waits for it (killing it after 10 s, with a bounded wait after the kill), and no gradle starts once
  a signal arrived. A gradle daemon (outside a mirror home) cancels on its own.
