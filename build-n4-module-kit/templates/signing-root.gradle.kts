// Signing block for the ROOT build.gradle.kts (the gradle root project, beside settings.gradle.kts).
// The New Module Wizard applies com.tridium.niagara-signing with no such block, so a missing or locked
// profile silently signs with the self-signed default key (Niagara4Modules) and the build stays green.
// With allowDefaultProfile off, the build FAILS instead. lint-structure.sh L15 WARNs a root without it.
// The alias and the profile path are per machine: pass them as gradle properties (the git-ignored
// ~/.gradle/gradle.properties or -P), never as a host path in a tracked file (lint-structure L10 doctrine):
//   niagaraSigningAlias=CodeSigning
//   niagaraSigningProfile=/path/to/code_signing.xml
// Form as taught in the N4 developer course (Lab 5 signing_configuration.kts); see build-verify.md
// § Signing per deploy target.

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
