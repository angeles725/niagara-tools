### Added — logic rule LC11 and the LP1 lab restart proof

- `build-n4-module-kit/types/logic.md` LC11 (extends LC10): a control cycle that runs before `atSteadyState()` can
  command outputs before the boot safety holds (power-on stagger, minimum off across a restart, anti-short-cycle)
  are armed, and can see a config "change" on every restart when its "last valid" state is not seeded. The rule:
  arm the holds first in the cycle, write OFF while held, keep output commands TRANSIENT, and seed the derived
  state with the persisted slots; with a source-structural test pattern.
- `build-n4-module-kit/build-verify.md` § Lab restart proof (LP1): the restart recipe that exposes both (non-default
  timers, every demand input linked, inputs written during boot, timestamped reads from before `Station Started`,
  each backup file missing once, proven cleanup).
