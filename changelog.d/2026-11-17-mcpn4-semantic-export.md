### Added — mcp-n4: n4_semantic_export, Haystack JSON of the hs: tags (O10)
- Read-only export of the components a NEQL query matches (default sites, spaces, equips and points) as a Project Haystack grid in the Haystack 4 JSON encoding: ids from slot paths, `dis`, every `hs:` tag and outbound `hs:` relation as a ref. Values that cannot be converted are listed in `skipped`; `save_as` writes a private file under `<state-dir>/exports/`.
- The conversion is pure functions in `go/internal/mcpn4/semantic`. Brick output is not produced.
