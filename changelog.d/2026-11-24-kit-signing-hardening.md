### Added — kit: `templates/signing-restricted.gradle.kts` and a restricted-signing section in `types/distribution.md`

- `templates/signing-restricted.gradle.kts` is the release-build signing block: the same code as
  `signing-root.gradle.kts` (`allowDefaultProfile.set(false)`, `niagaraSigning` with the alias and profile as gradle
  properties, so lint-structure L15/L16 read it the same way), plus the profile-file settings the block cannot enforce
  (`RestrictedSigningProfile`, `niagara.signing.standardtsa`, key size), a per-module alias override and the
  `JarSignerSigningProfile` (PKCS11 token) keys, each cited to devguide `security/codeSigning.txt` or Lab 5.
- `types/distribution.md` § Restricted signing says when to use it, the five-step internal-CA recipe (createProfile,
  generateCertificate, exportCertificate --csr-only, importCertificate, trust the CA PEM), the June-2023 hardware-key
  rule, hardware-token signing as far as verified, and the `javaToolchains` JDK preflight. Plugin 7.6.22
  `createProfile` type and a live token build are marked `[not verified]`.
- `types/structure.md` L15 row names the new template.
