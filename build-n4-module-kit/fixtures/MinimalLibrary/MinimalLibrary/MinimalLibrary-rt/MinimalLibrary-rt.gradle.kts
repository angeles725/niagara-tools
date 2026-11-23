/*
 * Copyright 2026 Angeles. All Rights Reserved.
 * MinimalLibrary-rt — minimal library module skeleton (rt profile, no Java, no palette).
 */

import com.tridium.gradle.plugins.module.util.ModulePart.RuntimeProfile.*

plugins {
  // The Niagara Module plugin configures the "moduleManifest" extension and the "jar" task.
  id("com.tridium.niagara-module")

  // The signing plugin configures the correct signing of modules. It requires
  // that the plugin also be applied to the root project.
  id("com.tridium.niagara-signing")

  // The niagara_home repositories convention plugin configures !bin/ext and
  // !modules as flat-file Maven repositories so that projects in this build can
  // depend on already-installed Niagara modules.
  id("com.tridium.convention.niagara-home-repositories")
}

description = "MinimalLibrary rt — minimal library module skeleton"

moduleManifest {
  moduleName.set("MinimalLibrary")
  runtimeProfile.set(rt)
}

dependencies {
  // NRE dependencies
  nre(":nre")

  // Niagara module dependencies: baja only, the wrapped library has no Niagara API dependency
  api(":baja")
}
