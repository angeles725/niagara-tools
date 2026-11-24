### Added — mcp-n4: `n4_list_templates`, the deployed template instances with their info and configs (issue #933, retro R13 ΔM1)

- `n4_list_templates(base, max_instances=200)` lists, in one read-only call, every deployed template instance below
  `base`: one BQL read `select slotPath from template:TemplateConfig` (the instance's root is the parent of the
  `templateConfig`), one read of the TemplateService's `template:TemplateInfo` rows, then one load per instance. Each
  entry has the root ORD, template name, vendor, version, the input/output/relation counts TemplateInfo holds,
  `default_passwords` / `has_default_passwords`, `signature` (template, local, `modified`) and one `configs` entry per
  `ConfigBinding` with its source slot's current value and the slot it is bound to. A slot the station does not answer
  is null with a note, never guessed; a station that refuses the instance query is an error and one with no instance
  answers `instances: []`. The unbound counts Workbench shows are computed there and are not station slots, so they are
  not reported. A component deployed by `n4_deploy_template` is not listed: that tool does not link the deployed
  component to its template.
