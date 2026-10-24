/*
 * Copyright 2026 Angeles. All Rights Reserved.
 * MinimalDriver-wb — minimal correct N4 NDriver point manager (wb profile).
 */

import com.tridium.gradle.plugins.bajadoc.task.Bajadoc
import com.tridium.gradle.plugins.module.util.ModulePart.RuntimeProfile.*

plugins {
  // The Niagara Module plugin configures the "moduleManifest" extension and the
  // "jar" and "moduleTestJar" tasks.
  id("com.tridium.niagara-module")

  // The signing plugin configures the correct signing of modules. It requires
  // that the plugin also be applied to the root project.
  id("com.tridium.niagara-signing")

  // The bajadoc plugin configures the generation of Bajadoc for a module.
  id("com.tridium.bajadoc")

  // Configures JaCoCo for the "niagaraTest" task of this module.
  id("com.tridium.niagara-jacoco")

  // The Annotation processors plugin adds default dependencies on ":nre"
  // for the "annotationProcessor" and "moduleTestAnnotationProcessor"
  // configurations by creating a single "niagaraAnnotationProcessor"
  // configuration they extend from.
  id("com.tridium.niagara-annotation-processors")

  // The niagara_home repositories convention plugin configures !bin/ext and
  // !modules as flat-file Maven repositories so that projects in this build can
  // depend on already-installed Niagara modules.
  id("com.tridium.convention.niagara-home-repositories")
}

description = "MinimalDriver wb — minimal correct N4 NDriver point manager"

moduleManifest {
  moduleName.set("MinimalDriver")
  runtimeProfile.set(wb)
}

// See documentation at module://docDeveloper/doc/build.html#dependencies for the supported
// dependency types
dependencies {
  // NRE dependencies
  nre(":nre")

  // Niagara module dependencies
  api(":baja")
  api(":alarm-rt")                       // module.xml lists the rt part's dependencies too (verify-module phantom-dep)
  api(":control-rt")
  api(":driver-rt")
  api(":ndriver-rt")
  api(":driver-wb")                      // BPointManager
  api(":ndriver-wb")                     // BNPointManager (NDriver point manager with discovery)
  api(":workbench-wb")                   // BAbstractManager
  api(":bajaui-wb")                      // BWidget (the manager is a widget)
  api(project(":MinimalDriver-rt"))      // the driver types the manager is an agent on

  // Test Niagara module dependencies
  moduleTestImplementation(":test-wb")
  moduleTestImplementation("junit:junit:4.13.2")
}

tasks.named<Bajadoc>("bajadoc") {
  includePackage("com.angeles.MinimalDriver.wb")
}
