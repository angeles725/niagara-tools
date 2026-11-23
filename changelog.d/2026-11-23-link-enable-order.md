### Changed

- `types/logic-authoring.md` and `types/dashboard.md`: disabling a running link re-activates it and writes the source value once more, so a write path that disables a link writes the target null after the disable; the code citations now point at `BLink.activate` and `SlotKnobs.propagate` (issue #920).
