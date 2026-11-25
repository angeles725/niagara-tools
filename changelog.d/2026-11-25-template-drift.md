### Changed

- `n4_deploy_template` (retro R13 ΔM5, issue #990): the dry run compares the `.ntpl` manifest's `<dependencies>`
  with the station's loaded modules (spy `sysManagers/moduleManager`, the page `n4_station_runtime` reads) and
  refuses the deploy before a token, naming each module the station does not have (`control-rt (4.15 minimum)`). A
  bare module name is met by any of its parts (`baja` by `baja-rt`). A module installed at a lower version than the
  manifest asks is a plan note, not a refusal: the station's own installer fails on a missing module and only warns on
  a version mismatch (`ApplicationTemplateInstaller.checkStationForCompatibleModules`, `TmplUtil.checkModuleDependencies`).
  A station that refuses the module page, a page without the Module/Vendor/Version table, a page cut before the end,
  or a client that cannot read spy pages leaves the dependencies unchecked, said in a plan note; the type registry
  check that follows is unchanged. Tests: `TestADeployRefusesAMissingManifestDependency` and three more in
  `writeops/template_deps_test.go`.
- `n4_list_templates` (retro R13 ΔM4, issue #990): each instance has `drift` (true when `ntplSignature` and
  `localSignature` differ, false when equal, null when either is unknown), and `signature.modified` carries the same
  value. An empty `localSignature` (the station never computed it) was reported `modified: true`; it is now null with an
  instance note, as is an empty or one-character `ntplSignature` (`BTemplateInfo.getModifiedState`). One reply note
  says the station refreshes `localSignature` only when its `updateSignatures` action runs (on service subscribe and in
  the upgrade job: `BTemplateService.doUpdateSignatures`, `BUpgradeTemplateJob`), so a stale value can hide an edit;
  the tool never invokes that write action. `counts.signature_modified` counts the true ones only. Tests:
  `TestLtDriftIsTrueFalseOrNull`, `TestListTemplatesLeavesDriftNullWhenASignatureIsUnknown`,
  `TestListTemplatesNotesThatLocalSignatureIsOnlyRefreshedByUpdateSignatures`.
