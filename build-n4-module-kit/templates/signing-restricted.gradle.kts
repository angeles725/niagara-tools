// Hardened signing block for the ROOT build.gradle.kts of a RELEASE build (a CA-signed or token-signed jar).
// It is the same two blocks as signing-root.gradle.kts (same gradle properties, so lint-structure L15/L16 read it
// the same way); this file adds what a release needs around them. Use signing-root.gradle.kts for a development
// or throwaway-key build, and this one when the jar must be trusted at moduleVerificationMode=high (types/
// distribution.md § Restricted signing). Pass the alias and the profile as gradle properties, never as a host
// path in a tracked file (lint-structure L10 doctrine):
//   niagaraSigningAlias=<alias in the keystore, or the alias your HSM reports>
//   niagaraSigningProfile=<path to the profile, outside the repository>
// Through toolbelt/build.sh: --gradle-prop niagaraSigningAlias=... --gradle-prop niagaraSigningProfile=...
//
// What each setting does [ev: devguide security/codeSigning.txt:59-78] (Lab 5 signing_configuration.kts is the same):
//   allowDefaultProfile.set(false)  a missing or locked profile FAILS the build instead of signing with the
//                                   default dev key (Niagara4Modules).
//   aliases                         the certificate alias(es) that sign every module; the first alias is the one a
//                                   JarSigner (token) profile hands to jarsigner [ev: corpus B1234 §1234.5].
//   signingProfileFile              the profile: an XML file (RestrictedSigningProfile) or a Java .properties file
//                                   (JarSignerSigningProfile, a token): the same block loads either.
//
// The block alone is not the whole hardening. The PROFILE FILE must also be right (devguide codeSigning.txt:26-49,
// 111-135):
//   niagara.signing.profileType   com.tridium.gradle.plugins.signing.profile.RestrictedSigningProfile (fully
//                                 qualified): a wrong alias then fails the build; a Local profile mints a new
//                                 self-signed key instead. `createProfile` writes Restricted in plugin 7.6.17
//                                 [ev: corpus B1249 §1249.3]; plugin 7.6.22 [not verified].
//   niagara.signing.standardtsa   an RFC 3161 SHA-256 time-stamp server (the course uses the DigiCert one): without
//                                 it a signed jar fails validation once the certificate expires. It needs network
//                                 access at build time; verification in Niagara still works offline.
//   niagara.signing.keyalg / keysize / dname   the devguide sample uses RSA 3072; a commercial CA may impose its
//                                 own key size and dname rules (codeSigning.txt:49).
// Recipe: createProfile -> edit dname, add standardtsa -> generateCertificate -> exportCertificate --csr-only ->
// CA signs -> importCertificate (types/distribution.md § Restricted signing). After any signing change run
// `gradlew clean` before `gradlew jar`, as Lab 5 does (it deletes the existing jars before it re-signs them)
// [ev: course Lab Instructions.txt:1001-1018]. Check the result in Workbench: Tools > Module Info
// shows a green check.
//
// Not live-built as a separate template: the two blocks below are the same code as signing-root.gradle.kts, whose
// header records an offline build. The profile-file keys above are devguide/Lab 5 text, not a run.

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

// Optional, in a module's OWN gradle file, not here: sign one module with a different certificate; a list signs it
// with several [ev: devguide codeSigning.txt:93-107].
//   niagaraSigning {
//     aliases.set(listOf("MyOtherCertificate"))
//   }
//
// Token (HSM, PKCS11) signing: point niagaraSigningProfile at a JarSignerSigningProfile .properties file, outside the
// repository, with the HSM driver installed and tested first (devguide codeSigning.txt:189-276). Its keys:
//   niagara.signing.profileType=com.tridium.gradle.plugins.signing.profile.JarSignerSigningProfile
//   jarsigner.cmd=jarsigner
//   jarsigner.args+=-keystore      (ONE argument per line; `jarsigner.args+=-foo bar` does not work)
//   jarsigner.args+=NONE
//   jarsigner.args+=-storetype
//   jarsigner.args+=PKCS11
//   jarsigner.args+=-storepass
//   jarsigner.args+=${jarsigner.storepass}   (kept in <profile basename>.credentials beside the profile)
//   jarsigner.args+=-providerClass           (a newer jarsigner may need -addProvider then SunPKCS11)
//   jarsigner.args+=sun.security.pkcs11.SunPKCS11
//   jarsigner.args+=-providerArg
//   jarsigner.args+=${profile.path}${/}eToken.cfg
//   jarsigner.args+=-signedjar
//   jarsigner.args+=${destfile}
//   jarsigner.args+=${srcfile}
//   jarsigner.args+=${alias}
// With allowDefaultProfile off a missing token fails the build. A failing `gradlew jar` is debugged with
// `gradlew jar --info`, which prints the full jarsigner command: run it outside Gradle first. Only .jar files are
// signed this way [ev: corpus B1234 §1234.5, jarsigner bytecode of plugin 7.6.17]. [not verified] on a live token.
