### Fixed — n4_replicate_points no longer flags Hex:-1 as a shared register

- The dry run's shared-register note treated `-1`, the address the station resets a duplicate writable point
  to, as a register; points and copies at `-1` are no longer noted.
