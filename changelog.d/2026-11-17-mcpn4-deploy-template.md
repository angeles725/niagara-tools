### Added — mcp-n4: `n4_deploy_template` deploys a component template (.ntpl) with its settings

- `n4_deploy_template(template_file, parent_ord, name?, settings?)` adds the template component of a local `.ntpl`
  under `parent_ord` in one op, with a dry run and a confirmation token like every write. `settings` fills the
  manifest's exposed settings (`num` checked against `min`/`max`, `bool`, `str`); a required setting the template
  leaves empty must be given; an unknown or metadata setting, a station or application template and a type the
  station's registry does not describe (a missing module) are refused before a token.
- The plan notes the template, its dependencies, each setting's value and slot, and what is not carried: the
  template configuration and every reference by template handle (links, ORD values), passwords, slot facets and
  categories. The read-back checks every planned node; the inverse removes the deployed component.
- Live-certified on the 4.15.6.20 lab station (deploy, read-back, rollback).
