### Added — lint-ntpl-manifest-deps: a template module jar as input, and the guide's manifest spelling (retro row K4)

- A template archive without `template-manifest.xml` is read through `template.manifest.xml`, the spelling of the
  Templates guide ("Example template.manifest.xml file"); with both, the hyphenated one alone is read.
- `lint-ntpl-manifest-deps.sh <template-module.jar>` checks each `*.ntpl` and `*.napl` entry of a template module
  jar as a shipped template, each row naming `<jar>!/<entry>`, within the bounds of a `.ntpl` (32 MiB per template
  inflated, 128 MiB for all). A jar declares no build dependencies: dependencies are judged against the
  installation alone. The lint engine gains `Spec.FileRoot`, an opt-in that only this rule sets; every other lint
  still refuses a file root ("not a directory", exit 3).
