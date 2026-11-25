### Changed

- `build-n4-module-kit/build-verify.md` § Lab restart proof (LP1): a control round (run the same boot on the previous
  jar first; a proof that cannot show the defect on the unfixed jar proves nothing) and a recipe for a real boot edge
  on a link-fed input (link while the consumer ignores the slot, set the target slot directly, then switch the
  mode; read between writes, two writes less than ~1 s apart can coalesce `[INFER]`). Retro R5 D4, D5; issue #988.
- `build-n4-module-kit/types/logic.md` LC11 item 5: a hold flag set only in `atSteadyState()` is false in the boot
  window, and a refusal sits before any shared resource is taken, not in the callee that runs after the caller
  committed. Retro R5 D6; issue #988.
