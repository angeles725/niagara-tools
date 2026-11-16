### Fixed — mcp-n4: n4_spy_page reads the registry's type pages
- `n4_spy_page` accepts one `module:Type` segment directly under `sysManagers/registryManager/types/` (e.g. `sysManagers/registryManager/types/control:NumericWritable`), the registry's page of that type; a `:` anywhere else (a scheme, an ORD, a segment after the type spec, a `..` trick) is still refused before any request.
