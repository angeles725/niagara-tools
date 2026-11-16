### Fixed — ng-deploy: the run lock says whose it is (retro rows D5 and D6 of the 2026-11-15 coordination stretch)

- The run lock is written in a temporary file beside it and hard-linked to `.ng-deploy.lock`, so it holds its
  content from the moment it is there: a run that loses the lock race reads who holds it instead of "it does not
  say which run made it; the file is 0s old" for a live run. A file system without hard links gets the lock in
  two steps, as before, and a NOTICE says so.
