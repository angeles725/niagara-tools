/*
 * Copyright 2026 Example. All Rights Reserved.
 * ModuleA-rt — a station service with a background export job (rt profile).
 */

import com.tridium.gradle.plugins.bajadoc.task.Bajadoc
import com.tridium.gradle.plugins.module.util.ModulePart.RuntimeProfile.*

plugins {
  id("com.tridium.niagara-module")
  id("com.tridium.niagara-signing")
  id("com.tridium.bajadoc")
  id("com.tridium.niagara-jacoco")
  // Adds ":nre" to the "annotationProcessor" configurations: what slotomatic needs.
  id("com.tridium.niagara-annotation-processors")
  id("com.tridium.convention.niagara-home-repositories")
}

description = "ModuleA rt — a station service with a background export job"

moduleManifest {
  moduleName.set("ModuleA")
  runtimeProfile.set(rt)
}

dependencies {
  nre(":nre")

  api(":baja")

  moduleTestImplementation(":test-wb")
  moduleTestImplementation("junit:junit:4.13.2")
}

tasks.named<Bajadoc>("bajadoc") {
  includePackage("com.example.ModuleA")
}
