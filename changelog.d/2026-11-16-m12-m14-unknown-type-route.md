### Fixed — mcp-n4: an unknown-type refusal names the stale-registry route; lab module-swap step (retro rows M12, M14)

- `n4_create_component`'s dry-run refusal of a type the station's registry does not describe now also says what to
  do when the module is installed and running: the station's registry may be stale, so stop the station, touch the
  module jar, start it until its log shows "Rebuilding registry", then reconnect with `n4_connect`. On a lab
  station a module jar removed and re-added under the same name left the registry "Up-to-date" without the module's
  types; the pre-check (#659) refused a client module type that a binary older than the pre-check had created.
- `mcp-n4-kit/METHODOLOGY.md` § 3 adds the lab module-swap step: touch the jar before the start and check the start
  log for the registry rebuild.
