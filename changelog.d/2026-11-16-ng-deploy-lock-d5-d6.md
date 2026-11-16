### Fixed — ng-deploy: the run lock says whose it is (retro rows D5 and D6 of the 2026-11-15 coordination stretch)

- The run lock is written in a temporary file beside it and hard-linked to `.ng-deploy.lock`, so it holds its
  content from the moment it is there: a run that loses the lock race reads who holds it instead of "it does not
  say which run made it; the file is 0s old" for a live run. A file system without hard links gets the lock in
  two steps, as before, and a NOTICE says so.
- The run lock records the program of the run (`command ng-deploy`), and the refusal names the check that tells a
  killed run from a live one, `ps -p <pid> -o comm=`, and what it must print, instead of `ps -p <pid>`, which a pid
  given again to another program passes. A lock of this host whose pid runs another program gets a second line,
  "pid <pid> runs <other>, not an ng-deploy run: ...". A lock an earlier version wrote (three lines) is still read.
