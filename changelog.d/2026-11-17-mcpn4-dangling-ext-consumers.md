### Fixed — mcp-n4: dangling outputs count alarm and history extensions as consumers (OD007)
- `n4_find_dangling_outputs` no longer reports a point whose `out` is read by its own alarm or history extension (no link involved); it lists it in `consumed_by_extension` with the extension type. The commissioning report's `dangling_outputs` check skips it too (client A had 13 such false positives).
