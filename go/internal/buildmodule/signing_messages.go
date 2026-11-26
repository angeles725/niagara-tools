package buildmodule

// Messages of the signing checks (signing_preflight.go, signing_key.go), stderr.
const (
	// msgNoSigningBlock: %[1]s the alias, %[2]s the root build file, %[3]s the template.
	msgNoSigningBlock = "build.sh: FAIL — niagaraSigningAlias=%[1]s is passed but %[2]s has no niagaraSigning {} block,\n" +
		"  so the signing plugin ignores the alias and the profile and signs with the mirror's throwaway key (NIAGARA4):\n" +
		"  the build would end at the verify gate (exit 50) with the wrong signer. Add the block, then build again:\n" +
		"    cat '%[3]s' >> '%[2]s'"
	// msgNoKeyIn: %[1]s the alias, %[2]s the profile.
	msgNoKeyIn = "build.sh: no private key for the alias %[1]s in the signing profile %[2]s"
	// msgNoKeyNoProfile: %[1]s the alias.
	msgNoKeyNoProfile = "build.sh: no private key for the alias %[1]s, and no niagaraSigningProfile is configured\n" +
		"  (--gradle-prop niagaraSigningProfile=<file>, ORG_GRADLE_PROJECT_niagaraSigningProfile or gradle.properties):\n" +
		"  the plugin used its default profile, which holds no such key"
	msgNoKeyProfileUnreadable = "  the profile cannot be read (%s)"
	msgNoKeyNoStore           = "  the profile names no niagara.signing.storepath, so there is no keystore to list"
	msgNoKeyStore             = "  keystore %[1]s (niagara.signing.storetype %[2]s)"
	msgNoKeyAliases           = "  the keystore holds the aliases: %s\n" +
		"  pass one of them: --gradle-prop niagaraSigningAlias=<one of them> (the passwords are not printed)"
	msgNoKeyCommand = "  could not list its aliases (%[3]s); run it yourself, the store password is the profile's\n" +
		"  niagara.signing.storepass entry, which is not printed here:\n" +
		"    keytool -list -keystore %[1]s -storetype %[2]s"
)
