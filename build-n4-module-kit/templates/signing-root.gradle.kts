// Signing block for the ROOT build.gradle.kts (the gradle root project, beside settings.gradle.kts).
// The New Module Wizard applies com.tridium.niagara-signing with no such block, so a missing or locked
// profile silently signs with the self-signed default key (Niagara4Modules) and the build stays green.
// With allowDefaultProfile off, the build FAILS instead. lint-structure.sh L15 WARNs a root without it.
// The alias and the profile path are per machine: pass them as gradle properties (the git-ignored
// ~/.gradle/gradle.properties or -P), never as a host path in a tracked file (lint-structure L10 doctrine):
//   niagaraSigningAlias=CodeSigning
//   niagaraSigningProfile=/path/to/code_signing.xml
// toolbelt/build.sh does not forward -P flags: through build.sh, export ORG_GRADLE_PROJECT_niagaraSigningAlias
// and ORG_GRADLE_PROJECT_niagaraSigningProfile instead (kit issue #308).
// This is a property-driven variant of the N4 developer course block (Lab 5 signing_configuration.kts), which
// sets the alias and the profile path as literals; the two signingServices/niagaraSigning settings are the same.
// Offline build, N4 4.14.0.162 mirror (Honeywell OptimizerSupervisor), niagara-module plugin 7.6.17, Gradle 7.6,
// a module from scaffold-module.sh built via build.sh [CERT-live local build]:
//   no properties                 -> configure passes, :<MOD>-rt:jar FAILS ("property
//                                    'signingAwareArchive.aliases' doesn't have a configured value"), exit 30
//   gradlew --offline help, dummy -P alias/profile -> BUILD SUCCESSFUL (the Provider-API form configures)
//   ORG_GRADLE_PROJECT_* to a missing profile file -> :jar FAILS ("specifies file ... which doesn't exist"), exit 30
//   throwaway self-signed JCEKS key + hand-written profile -> BUILD SUCCESSFUL, the jar carries
//                                    META-INF/<ALIAS8>.SF/.RSA signed by that key (jarsigner -verify: verified)
//   template as-is, no profile    -> :jar FAILS ("The default profile is not available. Specify one explicitly")
//   allowDefaultProfile.set(false) removed, no profile -> exit 0, signed with an auto-created Niagara4Modules
//                                    "For Development Purposes Only" key: the silent dev-key build this block prevents
// So the form compiles, configures, signs with the given profile, and fails closed without one.
// See build-verify.md § Signing per deploy target.

signingServices {
  // Disable the default profile: a missing profile fails the build instead of signing with the dev key
  signingProfileFactory {
    allowDefaultProfile.set(false)
  }
}

niagaraSigning {
  aliases.set(providers.gradleProperty("niagaraSigningAlias").map { listOf(it) })
  signingProfileFile.set(layout.projectDirectory.file(providers.gradleProperty("niagaraSigningProfile")))
}
