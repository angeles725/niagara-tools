### Added — n4_override_audit lists the writables that hold an override (issue #908, retro R8 ΔN7)

- New read-only tool `n4_override_audit(base, levels=[1, 8], max_writables=5000)`. It reads
  `select slotPath, out, overrideExpiration from <type>` for the numeric, boolean, enum and string writables and takes
  the active level from the out text (`@ 16` and `@ def` are no override, `@ 8` manual, `@ 1` emergency; the station's
  `where overrideExpiration != null` found no row in the lab, so every writable is read). Each override carries its
  level and kind, the out text, `expires` (null: `permanent: true`) and `in16` (the link feeding the writable's lowest
  input and, when its source is a schedule, that schedule's ORD, found by handle).
- `counts.permanent` 0 proves that no override is left without an expiry; `counts.by_level` shows every level the out
  texts hold. A writable type the station refuses is listed under `unreadable` and the others are still read; `levels`
  widens the list to link-driven inputs. The `handle` column of `schedule:ControlSchedule` is not yet checked on a
  live station: when it is refused, `in16.schedule` is null with a note and `in16.source_ord` still names the source.
