/*
 * Copyright 2026 Angeles. All Rights Reserved.
 * MinimalDash-ux — minimal correct N4 dashboard servlet skeleton (ux profile).
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

description = "MinimalDash ux — minimal correct N4 dashboard servlet skeleton"

moduleManifest {
  moduleName.set("MinimalDash")
  runtimeProfile.set(ux)
}

// See documentation at module://docDeveloper/doc/build.html#dependencies for the supported
// dependency types
dependencies {
  // NRE dependencies
  nre(":nre")

  // Niagara module dependencies
  api(":baja")
  api(":web-rt")                         // BWebServlet, WebOp (browser dashboard servlet)
  api(project(":MinimalDash-rt"))        // BMinimalDash facade
  compileOnly("javax.servlet:javax.servlet-api:3.1.0")  // HttpServletRequest/Response

  // Test Niagara module dependencies
  moduleTestImplementation(":test-wb")
  moduleTestImplementation("junit:junit:4.13.2")
}

// Package the browser frontend (index.html and other rc/ assets) into the jar;
// the servlet serves these via getClassLoader().getResourceAsStream("rc/…").
tasks.named<Jar>("jar") {
  from("src/rc") {
    include("**/*")
    into("rc")
  }
}

tasks.named<Bajadoc>("bajadoc") {
  includePackage("com.angeles.MinimalDash.ux")
}
