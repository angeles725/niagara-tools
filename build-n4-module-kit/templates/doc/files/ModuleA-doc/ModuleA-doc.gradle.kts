/*
 * Copyright 2026 Example. All Rights Reserved.
 * ModuleA-doc — the help part of ModuleA (runtime profile doc; types/structure.md § -doc help profile).
 */

import com.tridium.gradle.plugins.module.util.ModulePart.RuntimeProfile.*

plugins {
  id("com.tridium.niagara-module")
  id("com.tridium.niagara-signing")
  id("com.tridium.convention.niagara-home-repositories")

  // Packages the help pages and the TOC; bajadoc-module aggregates the API reference of the parts named below.
  id("com.tridium.niagara-doc")
  id("com.tridium.bajadoc-module")
}

description = "ModuleA doc — help pages"

// The same module name as the code part: Workbench finds a type's page in the module that registers the type.
moduleManifest {
  moduleName.set("ModuleA")
  runtimeProfile.set(doc)
}

dependencies {
  indexJars(":nre")
  indexJars(":baja")
  indexJars(":html-wb")
  indexJars(":help-wb")

  // Add bajadocs(project(":ModuleA-wb")) for a part with a -wb profile.
  bajadocs(project(":ModuleA-rt"))
}

tasks.named<Copy>("docCopy") {
  from("src") {
    include("doc/**/*.*")
  }
}
