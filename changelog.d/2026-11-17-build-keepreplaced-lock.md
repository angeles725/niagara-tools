### Fixed — build.sh no longer keeps a stale mirror copy when interrupted

- On a mirror home, the step that keeps a replaced jar (`.mirror-replaced/`) now waits for an interrupt's
  restore to finish, so it no longer keeps a copy or prints a "replaced" row for a jar the restore puts back.
