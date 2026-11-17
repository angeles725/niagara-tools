### Fixed — build.sh keeps a jar that passed the gate when interrupted during the report

- A signal while report-module ran rolled back the jar that had already passed the gate; the gate now marks
  modules/ as verified when it passes.
- A gradle whose child kept the output open past 10 s no longer fails a passing build (the `WaitDelay` added in
  #795 is removed; only the signal handler's wait is bounded).
